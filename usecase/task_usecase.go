package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

type taskUsecase struct {
	repo      domain.TaskRepository
	queue     domain.TaskQueue
	runner    domain.DockerRunner
	dockerCfg domain.DockerConfig

	running   map[string]context.CancelFunc
	mu        sync.Mutex
	cancelCtx context.CancelFunc
}

func NewTaskUsecase(repo domain.TaskRepository, queue domain.TaskQueue, runner domain.DockerRunner, dockerCfg domain.DockerConfig) domain.TaskUsecase {
	return &taskUsecase{
		repo:      repo,
		queue:     queue,
		runner:    runner,
		dockerCfg: dockerCfg,
		running:   make(map[string]context.CancelFunc),
	}
}

func (u *taskUsecase) HandleWebhook(ctx context.Context, payload domain.WebhookPayload) (domain.Task, error) {
	active, err := u.repo.FindActiveByIssue(ctx, payload.Repository, payload.IssueNumber)
	if err != nil {
		return domain.Task{}, fmt.Errorf("check active tasks: %w", err)
	}
	if len(active) > 0 {
		return domain.Task{}, fmt.Errorf("issue already has an active task")
	}

	task, err := domain.NewTask(payload)
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}
	if err := u.repo.Create(ctx, task); err != nil {
		return domain.Task{}, fmt.Errorf("persist task: %w", err)
	}

	u.queue.Enqueue(task)
	slog.Info("task enqueued", "task_id", task.ID, "issue", task.IssueNumber, "repo", task.Repository)
	return task, nil
}

func (u *taskUsecase) GetTask(ctx context.Context, id string) (domain.Task, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *taskUsecase) ListTasks(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error) {
	return u.repo.List(ctx, status)
}

func (u *taskUsecase) CancelTask(ctx context.Context, id string) error {
	task, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	if task.Status == domain.TaskStatusRunning && u.runner != nil {
		if err := u.runner.StopContainer(ctx, task.ContainerID); err != nil {
			slog.Warn("failed to stop container", "container_id", task.ContainerID, "error", err)
		}
	}

	u.mu.Lock()
	if cancel, ok := u.running[id]; ok {
		cancel()
		delete(u.running, id)
	}
	u.mu.Unlock()

	return u.repo.UpdateFinished(ctx, id, domain.TaskStatusCancelled, "cancelled")
}

func (u *taskUsecase) GetTaskLogs(ctx context.Context, id string) (string, error) {
	task, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	return task.Log, nil
}

func (u *taskUsecase) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	u.cancelCtx = cancel

	u.recoverPending(ctx)

	go u.scheduleLoop(ctx)
	slog.Info("scheduler started", "max_concurrency", u.dockerCfg.MaxConcurrency)
	return nil
}

func (u *taskUsecase) Stop() {
	if u.cancelCtx != nil {
		u.cancelCtx()
	}
	u.queue.Close()
	slog.Info("scheduler stopped")
}

func (u *taskUsecase) recoverPending(ctx context.Context) {
	tasks, err := u.repo.List(ctx, domain.TaskStatusPending)
	if err != nil {
		slog.Error("recover pending tasks", "error", err)
		return
	}
	for _, t := range tasks {
		u.queue.Enqueue(t)
	}
	slog.Info("recovered pending tasks", "count", len(tasks))

	running, err := u.repo.List(ctx, domain.TaskStatusRunning)
	if err != nil {
		slog.Error("recover running tasks", "error", err)
		return
	}
	for _, t := range running {
		if u.runner != nil {
			go u.watchContainer(ctx, t)
		}
	}
	slog.Info("recovered running tasks", "count", len(running))
}

func (u *taskUsecase) scheduleLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-u.queue.Dequeue():
			if !ok {
				return
			}
			u.mu.Lock()
			currentRunning := len(u.running)
			u.mu.Unlock()

			if currentRunning >= u.dockerCfg.MaxConcurrency {
				u.queue.Enqueue(task)
				continue
			}

			if u.runner == nil {
				slog.Error("no docker runner configured")
				continue
			}

			containerID, err := u.runner.StartContainer(ctx, task)
			if err != nil {
				slog.Error("start container", "task_id", task.ID, "error", err)
				if updateErr := u.repo.UpdateFinished(ctx, task.ID, domain.TaskStatusFailed, err.Error()); updateErr != nil {
					slog.Error("update failed status", "task_id", task.ID, "error", updateErr)
				}
				continue
			}

			if err := u.repo.UpdateStatus(ctx, task.ID, domain.TaskStatusRunning, containerID); err != nil {
				slog.Error("update running status", "task_id", task.ID, "error", err)
			}
			task.ContainerID = containerID
			task.Status = domain.TaskStatusRunning

			go u.watchContainer(ctx, task)
		}
	}
}

func (u *taskUsecase) watchContainer(ctx context.Context, task domain.Task) {
	taskCtx, cancel := context.WithCancel(ctx)
	u.mu.Lock()
	u.running[task.ID] = cancel
	u.mu.Unlock()

	defer func() {
		cancel()
		u.mu.Lock()
		delete(u.running, task.ID)
		u.mu.Unlock()
	}()

	logCh, err := u.runner.ContainerLogs(taskCtx, task.ContainerID)
	if err != nil {
		slog.Error("get container logs", "task_id", task.ID, "error", err)
	}

	var logBuf string
	for line := range logCh {
		logBuf += line
		if err := u.repo.AppendLog(taskCtx, task.ID, line); err != nil {
			slog.Error("append log", "task_id", task.ID, "error", err)
		}
	}

	exitCode, err := u.runner.WaitContainer(taskCtx, task.ContainerID)
	if err != nil {
		slog.Error("wait container", "task_id", task.ID, "error", err)
		if updateErr := u.repo.UpdateFinished(ctx, task.ID, domain.TaskStatusFailed, logBuf); updateErr != nil {
			slog.Error("update failed status", "task_id", task.ID, "error", updateErr)
		}
		return
	}

	status := domain.TaskStatusSucceeded
	if exitCode != 0 {
		status = domain.TaskStatusFailed
	}
	if err := u.repo.UpdateFinished(ctx, task.ID, status, logBuf); err != nil {
		slog.Error("update finished status", "task_id", task.ID, "error", err)
	}
	slog.Info("task completed", "task_id", task.ID, "status", status, "exit_code", exitCode)
}

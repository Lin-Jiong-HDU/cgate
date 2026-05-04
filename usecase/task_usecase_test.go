package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/queue"
	"github.com/Lin-Jiong-HDU/go-project-template/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock Repository ---

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) Create(ctx context.Context, task domain.Task) error {
	return m.Called(ctx, task).Error(0)
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (domain.Task, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Task), args.Error(1)
}

func (m *mockRepo) List(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error) {
	args := m.Called(ctx, status)
	return args.Get(0).([]domain.Task), args.Error(1)
}

func (m *mockRepo) UpdateStatus(ctx context.Context, id string, status domain.TaskStatus, containerID string) error {
	return m.Called(ctx, id, status, containerID).Error(0)
}

func (m *mockRepo) AppendLog(ctx context.Context, id string, log string) error {
	return m.Called(ctx, id, log).Error(0)
}

func (m *mockRepo) UpdateFinished(ctx context.Context, id string, status domain.TaskStatus, log string) error {
	return m.Called(ctx, id, status, log).Error(0)
}

func (m *mockRepo) FindActiveByIssue(ctx context.Context, repository string, issueNumber int) ([]domain.Task, error) {
	args := m.Called(ctx, repository, issueNumber)
	return args.Get(0).([]domain.Task), args.Error(1)
}

// --- Mock DockerRunner ---

type mockRunner struct {
	mock.Mock
}

func (m *mockRunner) StartContainer(ctx context.Context, task domain.Task) (string, error) {
	args := m.Called(ctx, task)
	return args.String(0), args.Error(1)
}

func (m *mockRunner) StopContainer(ctx context.Context, containerID string) error {
	return m.Called(ctx, containerID).Error(0)
}

func (m *mockRunner) CleanupTask(ctx context.Context, taskID string, containerID string) error {
	return m.Called(ctx, taskID, containerID).Error(0)
}

func (m *mockRunner) ContainerLogs(ctx context.Context, containerID string) (<-chan string, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		ch := make(chan string)
		close(ch)
		return ch, args.Error(1)
	}
	return args.Get(0).(<-chan string), args.Error(1)
}

func (m *mockRunner) WaitContainer(ctx context.Context, containerID string) (int, error) {
	args := m.Called(ctx, containerID)
	return args.Int(0), args.Error(1)
}

func (m *mockRunner) IsRunning(ctx context.Context, containerID string) (bool, error) {
	args := m.Called(ctx, containerID)
	return args.Bool(0), args.Error(1)
}

// --- Tests ---

func TestHandleWebhook_CreatesAndEnqueues(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Close()

	repo.On("FindActiveByIssue", mock.Anything, "owner/repo", 42).Return([]domain.Task{}, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Task")).Return(nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1}, nil)
	payload := domain.WebhookPayload{
		IssueNumber: 42,
		Title:       "Add feature [claude bot]",
		Repository:  "owner/repo",
		Author:      "user",
		URL:         "https://github.com/owner/repo/issues/42",
	}

	task, err := uc.HandleWebhook(context.Background(), payload)
	assert.NoError(t, err)
	assert.Equal(t, domain.TaskStatusPending, task.Status)
	assert.Equal(t, 42, task.IssueNumber)
	repo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domain.Task"))
}

func TestHandleWebhook_RejectsDuplicateActive(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Close()

	existing := domain.Task{ID: "existing", Status: domain.TaskStatusRunning}
	repo.On("FindActiveByIssue", mock.Anything, "owner/repo", 42).Return([]domain.Task{existing}, nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1}, nil)
	payload := domain.WebhookPayload{
		IssueNumber: 42,
		Title:       "Add feature",
		Repository:  "owner/repo",
	}

	_, err := uc.HandleWebhook(context.Background(), payload)
	assert.Error(t, err)
}

func TestGetTask_ReturnsTask(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Close()

	expected := domain.Task{ID: "task-1", Title: "test"}
	repo.On("GetByID", mock.Anything, "task-1").Return(expected, nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1}, nil)
	got, err := uc.GetTask(context.Background(), "task-1")
	assert.NoError(t, err)
	assert.Equal(t, "task-1", got.ID)
}

func TestListTasks_DelegatesToRepo(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Close()

	tasks := []domain.Task{{ID: "1"}, {ID: "2"}}
	repo.On("List", mock.Anything, domain.TaskStatusPending).Return(tasks, nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1}, nil)
	got, err := uc.ListTasks(context.Background(), domain.TaskStatusPending)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestCancelTask_StopsRunningContainer(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	runner := new(mockRunner)
	q := queue.New()
	defer q.Close()

	task := domain.Task{ID: "t1", Status: domain.TaskStatusRunning, ContainerID: "c1"}
	repo.On("GetByID", mock.Anything, "t1").Return(task, nil)
	runner.On("StopContainer", mock.Anything, "c1").Return(nil)
	repo.On("UpdateFinished", mock.Anything, "t1", domain.TaskStatusCancelled, "cancelled").Return(nil)

	uc := usecase.NewTaskUsecase(repo, q, runner, domain.DockerConfig{MaxConcurrency: 1}, nil)
	err := uc.CancelTask(context.Background(), "t1")
	assert.NoError(t, err)
	runner.AssertCalled(t, "StopContainer", mock.Anything, "c1")
	runner.AssertNotCalled(t, "CleanupTask", mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelTask_PendingTask_MarksCancelled(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Close()

	task := domain.Task{ID: "t1", Status: domain.TaskStatusPending}
	repo.On("GetByID", mock.Anything, "t1").Return(task, nil)
	repo.On("UpdateFinished", mock.Anything, "t1", domain.TaskStatusCancelled, "cancelled").Return(nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1}, nil)
	err := uc.CancelTask(context.Background(), "t1")
	assert.NoError(t, err)
	repo.AssertCalled(t, "UpdateFinished", mock.Anything, "t1", domain.TaskStatusCancelled, "cancelled")
}

func TestWatchContainer_CleansUpOnSuccess(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	runner := new(mockRunner)
	q := queue.New()
	defer q.Close()

	cleanupCh := make(chan struct{})
	runner.On("StartContainer", mock.Anything, mock.AnythingOfType("domain.Task")).Return("c2", nil)
	repo.On("List", mock.Anything, domain.TaskStatusPending).Return([]domain.Task{}, nil)
	repo.On("List", mock.Anything, domain.TaskStatusRunning).Return([]domain.Task{}, nil)
	repo.On("UpdateStatus", mock.Anything, mock.AnythingOfType("string"), domain.TaskStatusRunning, "c2").Return(nil)
	runner.On("ContainerLogs", mock.Anything, "c2").Return(nil, nil)
	runner.On("WaitContainer", mock.Anything, "c2").Return(0, nil)
	runner.On("CleanupTask", mock.Anything, mock.AnythingOfType("string"), "c2").Return(nil).Run(func(_ mock.Arguments) {
		close(cleanupCh)
	})
	repo.On("UpdateFinished", mock.Anything, mock.AnythingOfType("string"), domain.TaskStatusSucceeded, mock.Anything).Return(nil)
	repo.On("FindActiveByIssue", mock.Anything, "owner/repo", 99).Return([]domain.Task{}, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Task")).Return(nil)

	uc := usecase.NewTaskUsecase(repo, q, runner, domain.DockerConfig{MaxConcurrency: 1}, nil)
	require.NoError(t, uc.Start(context.Background()))
	defer uc.Stop()

	_, err := uc.HandleWebhook(context.Background(), domain.WebhookPayload{
		IssueNumber: 99,
		Title:       "test [claude bot]",
		Repository:  "owner/repo",
	})
	require.NoError(t, err)

	select {
	case <-cleanupCh:
	case <-time.After(2 * time.Second):
		t.Fatal("CleanupTask was not called within timeout")
	}
}

func TestHandleWebhook_RejectsUnauthorizedAuthor(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Close()

	allowedAuthors := []string{"trusted-user", "admin"}

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1}, allowedAuthors)
	payload := domain.WebhookPayload{
		IssueNumber: 42,
		Title:       "Add feature [claude bot]",
		Repository:  "owner/repo",
		Author:      "random-attacker",
		URL:         "https://github.com/owner/repo/issues/42",
	}

	_, err := uc.HandleWebhook(context.Background(), payload)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestHandleWebhook_AllowsAuthorizedAuthor(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Close()

	allowedAuthors := []string{"trusted-user", "admin"}
	repo.On("FindActiveByIssue", mock.Anything, "owner/repo", 42).Return([]domain.Task{}, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Task")).Return(nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1}, allowedAuthors)
	payload := domain.WebhookPayload{
		IssueNumber: 42,
		Title:       "Add feature [claude bot]",
		Repository:  "owner/repo",
		Author:      "trusted-user",
		URL:         "https://github.com/owner/repo/issues/42",
	}

	task, err := uc.HandleWebhook(context.Background(), payload)
	assert.NoError(t, err)
	assert.Equal(t, domain.TaskStatusPending, task.Status)
}

func TestHandleWebhook_EmptyAllowlist_AllowsAll(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Close()

	repo.On("FindActiveByIssue", mock.Anything, "owner/repo", 42).Return([]domain.Task{}, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Task")).Return(nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1}, nil)
	payload := domain.WebhookPayload{
		IssueNumber: 42,
		Title:       "Add feature [claude bot]",
		Repository:  "owner/repo",
		Author:      "anyone",
		URL:         "https://github.com/owner/repo/issues/42",
	}

	task, err := uc.HandleWebhook(context.Background(), payload)
	assert.NoError(t, err)
	assert.Equal(t, domain.TaskStatusPending, task.Status)
}

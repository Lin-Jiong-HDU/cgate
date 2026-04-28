package domain

import "context"

type TaskUsecase interface {
	HandleWebhook(ctx context.Context, payload WebhookPayload) (Task, error)
	GetTask(ctx context.Context, id string) (Task, error)
	ListTasks(ctx context.Context, status TaskStatus) ([]Task, error)
	CancelTask(ctx context.Context, id string) error
	GetTaskLogs(ctx context.Context, id string) (string, error)
	Start(ctx context.Context) error
	Stop()
}

type DockerRunner interface {
	StartContainer(ctx context.Context, task Task) (containerID string, err error)
	StopContainer(ctx context.Context, containerID string) error
	ContainerLogs(ctx context.Context, containerID string) (<-chan string, error)
	WaitContainer(ctx context.Context, containerID string) (exitCode int, err error)
	IsRunning(ctx context.Context, containerID string) (bool, error)
}

type TaskQueue interface {
	Enqueue(task Task)
	Dequeue() <-chan Task
	Len() int
	Close()
}

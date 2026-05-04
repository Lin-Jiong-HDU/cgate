package domain

import "context"

type TaskRepository interface {
	Create(ctx context.Context, task Task) error
	GetByID(ctx context.Context, id string) (Task, error)
	List(ctx context.Context, status TaskStatus) ([]Task, error)
	UpdateStatus(ctx context.Context, id string, status TaskStatus, containerID string) error
	AppendLog(ctx context.Context, id string, log string) error
	UpdateFinished(ctx context.Context, id string, status TaskStatus, log string) error
	FindActiveByIssue(ctx context.Context, repository string, issueNumber int) ([]Task, error)
	FindActiveByPR(ctx context.Context, repository string, prNumber int) ([]Task, error)
}

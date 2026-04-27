package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

type taskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) domain.TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task domain.Task) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (id, issue_number, title, body, author, repository, html_url, status, container_id, log, created_at, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.IssueNumber, task.Title, task.Body, task.Author, task.Repository, task.HTMLURL, task.Status, task.ContainerID, task.Log, task.CreatedAt, task.StartedAt, task.FinishedAt,
	)
	return err
}

func (r *taskRepository) GetByID(ctx context.Context, id string) (domain.Task, error) {
	var task domain.Task
	var startedAt, finishedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, issue_number, title, body, author, repository, html_url, status, container_id, log, created_at, started_at, finished_at
		 FROM tasks WHERE id = ?`, id,
	).Scan(&task.ID, &task.IssueNumber, &task.Title, &task.Body, &task.Author, &task.Repository, &task.HTMLURL, &task.Status, &task.ContainerID, &task.Log, &task.CreatedAt, &startedAt, &finishedAt)

	if err != nil {
		return task, fmt.Errorf("get task %s: %w", id, err)
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		task.FinishedAt = &finishedAt.Time
	}
	return task, nil
}

func (r *taskRepository) List(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error) {
	var rows *sql.Rows
	var err error

	if status == "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, issue_number, title, body, author, repository, html_url, status, container_id, log, created_at, started_at, finished_at
			 FROM tasks ORDER BY created_at DESC`)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, issue_number, title, body, author, repository, html_url, status, container_id, log, created_at, started_at, finished_at
			 FROM tasks WHERE status = ? ORDER BY created_at DESC`, status)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanTasks(rows)
}

func (r *taskRepository) UpdateStatus(ctx context.Context, id string, status domain.TaskStatus, containerID string) error {
	if status == domain.TaskStatusRunning {
		now := time.Now()
		_, err := r.db.ExecContext(ctx,
			`UPDATE tasks SET status = ?, container_id = ?, started_at = ? WHERE id = ?`,
			status, containerID, now, id)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, container_id = ? WHERE id = ?`,
		status, containerID, id)
	return err
}

func (r *taskRepository) AppendLog(ctx context.Context, id string, log string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET log = log || ? WHERE id = ?`, log, id)
	return err
}

func (r *taskRepository) UpdateFinished(ctx context.Context, id string, status domain.TaskStatus, log string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, log = ?, finished_at = ? WHERE id = ?`,
		status, log, now, id)
	return err
}

func (r *taskRepository) FindActiveByIssue(ctx context.Context, repository string, issueNumber int) ([]domain.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, issue_number, title, body, author, repository, html_url, status, container_id, log, created_at, started_at, finished_at
		 FROM tasks WHERE repository = ? AND issue_number = ? AND status IN ('pending', 'running')`,
		repository, issueNumber)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanTasks(rows)
}

func scanTasks(rows *sql.Rows) ([]domain.Task, error) {
	var tasks []domain.Task
	for rows.Next() {
		var task domain.Task
		var startedAt, finishedAt sql.NullTime
		err := rows.Scan(&task.ID, &task.IssueNumber, &task.Title, &task.Body, &task.Author, &task.Repository, &task.HTMLURL, &task.Status, &task.ContainerID, &task.Log, &task.CreatedAt, &startedAt, &finishedAt)
		if err != nil {
			return nil, err
		}
		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			task.FinishedAt = &finishedAt.Time
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

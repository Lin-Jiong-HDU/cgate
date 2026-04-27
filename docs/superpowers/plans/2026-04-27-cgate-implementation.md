# CGate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a backend service that receives GitHub Issue webhooks and manages parallel Claude Code instances in Docker containers.

**Architecture:** Event-driven with in-memory task queue, SQLite persistence, and Docker container pool. Clean Architecture layers: domain → repository → usecase → controller → route. Infrastructure (Docker runner, queue) in `internal/`, called through domain interfaces.

**Tech Stack:** Go 1.24, `net/http` stdlib, `github.com/docker/docker`, `github.com/mattn/go-sqlite3`, `github.com/spf13/viper`, `github.com/stretchr/testify`

---

## File Map

| File | Responsibility |
|------|---------------|
| `domain/task.go` | Task entity, TaskStatus enum, WebhookPayload struct |
| `domain/task_test.go` | Task entity constructor tests |
| `domain/repository.go` | TaskRepository interface |
| `domain/usecase.go` | TaskUsecase interface + DockerRunner interface + TaskQueue interface |
| `domain/config.go` | Config entities (ServerConfig, DockerConfig, etc.) |
| `repository/sqlite.go` | SQLite initialization, schema migration |
| `repository/sqlite_test.go` | Migration tests |
| `repository/task_repository.go` | TaskRepository implementation |
| `repository/task_repository_test.go` | CRUD tests |
| `internal/queue/queue.go` | In-memory FIFO task queue |
| `internal/queue/queue_test.go` | Queue behavior tests |
| `internal/docker/runner.go` | DockerRunner implementation using Docker Engine API |
| `internal/docker/runner_test.go` | Runner tests with mocked Docker client |
| `usecase/task_usecase.go` | TaskUsecase implementation: enqueue, schedule, cancel, query |
| `usecase/task_usecase_test.go` | TDD for scheduling logic, concurrency, error paths |
| `api/middleware/auth.go` | Webhook secret validation middleware |
| `api/middleware/auth_test.go` | Auth middleware tests |
| `api/controller/webhook_controller.go` | POST /webhook/github handler |
| `api/controller/webhook_controller_test.go` | Webhook handler tests |
| `api/controller/task_controller.go` | GET/POST /api/tasks handlers |
| `api/controller/task_controller_test.go` | Task API handler tests |
| `api/route/route.go` | Route registration, DI wiring |
| `bootstrap/bootstrap.go` | Config loading, DB init, DI assembly |
| `cmd/main.go` | Entry point, calls bootstrap |
| `config.yaml` | Default configuration |
| `Dockerfile` | cgate container image |
| `docker-compose.yml` | Deployment composition |
| `Makefile` | Updated build targets |

---

### Task 1: Domain Layer — Entities and Interfaces

**Files:**
- Create: `domain/task.go`
- Create: `domain/task_test.go`
- Create: `domain/repository.go`
- Create: `domain/usecase.go`
- Create: `domain/config.go`

- [ ] **Step 1: Create domain/config.go**

```go
package domain

import "time"

type Config struct {
	Server ServerConfig
	Docker DockerConfig
	Queue  QueueConfig
	GitHub GitHubConfig
}

type ServerConfig struct {
	Port int
}

type DockerConfig struct {
	Image          string
	MaxConcurrency int
	Timeout        time.Duration
	SettingsPath   string
}

type QueueConfig struct {
	MaxRetries int
}

type GitHubConfig struct {
	PAT           string
	WebhookSecret string
}
```

- [ ] **Step 2: Create domain/task.go**

```go
package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID          string
	IssueNumber int
	Title       string
	Body        string
	Author      string
	Repository  string
	HTMLURL     string
	Status      TaskStatus
	ContainerID string
	Log         string
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

type WebhookPayload struct {
	Action     string `json:"action"`
	IssueNumber int   `json:"issue_number"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Author     string `json:"author"`
	Labels     []string `json:"labels"`
	URL        string `json:"url"`
	Repository string `json:"repository"`
	CreatedAt  string `json:"created_at"`
}

func NewTask(payload WebhookPayload) Task {
	id := generateID()
	return Task{
		ID:          id,
		IssueNumber: payload.IssueNumber,
		Title:       payload.Title,
		Body:        payload.Body,
		Author:      payload.Author,
		Repository:  payload.Repository,
		HTMLURL:     payload.URL,
		Status:      TaskStatusPending,
		CreatedAt:   time.Now(),
	}
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
```

- [ ] **Step 3: Create domain/task_test.go**

```go
package domain_test

import (
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

func TestNewTask_ConstructsFromPayload(t *testing.T) {
	t.Parallel()
	payload := domain.WebhookPayload{
		Action:      "opened",
		IssueNumber: 42,
		Title:       "Add login page [claude bot]",
		Body:        "Please add a login page",
		Author:      "testuser",
		Repository:  "owner/repo",
		URL:         "https://github.com/owner/repo/issues/42",
	}

	task := domain.NewTask(payload)

	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
	if task.IssueNumber != 42 {
		t.Errorf("expected IssueNumber 42, got %d", task.IssueNumber)
	}
	if task.Title != payload.Title {
		t.Errorf("expected Title %q, got %q", payload.Title, task.Title)
	}
	if task.Repository != "owner/repo" {
		t.Errorf("expected Repository owner/repo, got %s", task.Repository)
	}
	if task.Status != domain.TaskStatusPending {
		t.Errorf("expected Status pending, got %s", task.Status)
	}
	if task.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if task.StartedAt != nil {
		t.Error("expected nil StartedAt")
	}
	if task.FinishedAt != nil {
		t.Error("expected nil FinishedAt")
	}
}

func TestNewTask_GeneratesUniqueIDs(t *testing.T) {
	t.Parallel()
	payload := domain.WebhookPayload{IssueNumber: 1}
	task1 := domain.NewTask(payload)
	task2 := domain.NewTask(payload)

	if task1.ID == task2.ID {
		t.Error("expected unique IDs")
	}
}
```

- [ ] **Step 4: Create domain/repository.go**

```go
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
}
```

- [ ] **Step 5: Create domain/usecase.go**

```go
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
```

- [ ] **Step 6: Run tests to verify**

Run: `go vet ./... && go test ./domain/ -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add domain/
git commit -m "feat(domain): add task entity, config, and interfaces"
```

---

### Task 2: In-Memory Task Queue

**Files:**
- Create: `internal/queue/queue.go`
- Create: `internal/queue/queue_test.go`

- [ ] **Step 1: Write the failing tests for queue**

Create `internal/queue/queue_test.go`:

```go
package queue_test

import (
	"sync/atomic"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/queue"
)

func TestQueue_EnqueueDequeue(t *testing.T) {
	t.Parallel()
	q := queue.New()
	defer q.Close()

	task := domain.Task{ID: "test-1"}
	q.Enqueue(task)

	got := <-q.Dequeue()
	if got.ID != "test-1" {
		t.Errorf("expected ID test-1, got %s", got.ID)
	}
}

func TestQueue_DequeueBlocksWhenEmpty(t *testing.T) {
	t.Parallel()
	q := queue.New()
	defer q.Close()

	var received atomic.Int32
	go func() {
		got := <-q.Dequeue()
		_ = got
		received.Store(1)
	}()

	if received.Load() == 1 {
		t.Error("Dequeue should block when queue is empty")
	}

	q.Enqueue(domain.Task{ID: "unblock"})
	// Give goroutine time to receive
	for i := 0; i < 100; i++ {
		if received.Load() == 1 {
			break
		}
	}
	if received.Load() != 1 {
		t.Error("expected Dequeue to unblock after Enqueue")
	}
}

func TestQueue_Len(t *testing.T) {
	t.Parallel()
	q := queue.New()
	defer q.Close()

	if q.Len() != 0 {
		t.Errorf("expected Len 0, got %d", q.Len())
	}

	q.Enqueue(domain.Task{ID: "1"})
	q.Enqueue(domain.Task{ID: "2"})

	if q.Len() != 2 {
		t.Errorf("expected Len 2, got %d", q.Len())
	}
}

func TestQueue_FIFOOrder(t *testing.T) {
	t.Parallel()
	q := queue.New()
	defer q.Close()

	q.Enqueue(domain.Task{ID: "first"})
	q.Enqueue(domain.Task{ID: "second"})

	got1 := <-q.Dequeue()
	got2 := <-q.Dequeue()

	if got1.ID != "first" {
		t.Errorf("expected first, got %s", got1.ID)
	}
	if got2.ID != "second" {
		t.Errorf("expected second, got %s", got2.ID)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/queue/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement the queue**

Create `internal/queue/queue.go`:

```go
package queue

import (
	"sync"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

type queue struct {
	ch   chan domain.Task
	once sync.Once
}

func New() domain.TaskQueue {
	return &queue{
		ch: make(chan domain.Task, 256),
	}
}

func (q *queue) Enqueue(task domain.Task) {
	q.ch <- task
}

func (q *queue) Dequeue() <-chan domain.Task {
	return q.ch
}

func (q *queue) Len() int {
	return len(q.ch)
}

func (q *queue) Close() {
	q.once.Do(func() {
		close(q.ch)
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/queue/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/queue/
git commit -m "feat(queue): add in-memory FIFO task queue"
```

---

### Task 3: SQLite Repository

**Files:**
- Create: `repository/sqlite.go`
- Create: `repository/sqlite_test.go`
- Create: `repository/task_repository.go`
- Create: `repository/task_repository_test.go`

- [ ] **Step 1: Install SQLite dependency**

Run: `go get github.com/mattn/go-sqlite3`

- [ ] **Step 2: Write the failing test for SQLite init**

Create `repository/sqlite_test.go`:

```go
package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/repository"
)

func TestInitDB_CreatesSchema(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT count(*) FROM tasks").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query tasks table: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows, got %d", count)
	}
}

func TestInitDB_Idempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db1, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("first InitDB failed: %v", err)
	}
	db1.Close()

	db2, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("second InitDB failed: %v", err)
	}
	defer db2.Close()

	// Should not error — schema already exists
	var count int
	err = db2.QueryRow("SELECT count(*) FROM tasks").Scan(&count)
	if err != nil {
		t.Fatalf("failed after idempotent migration: %v", err)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./repository/ -v -run TestInitDB`
Expected: FAIL — package does not exist

- [ ] **Step 4: Implement SQLite init**

Create `repository/sqlite.go`:

```go
package repository

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		issue_number INTEGER NOT NULL,
		title TEXT NOT NULL,
		body TEXT DEFAULT '',
		author TEXT DEFAULT '',
		repository TEXT NOT NULL,
		html_url TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		container_id TEXT DEFAULT '',
		log TEXT DEFAULT '',
		created_at DATETIME NOT NULL,
		started_at DATETIME,
		finished_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_repo_issue ON tasks(repository, issue_number);
	`
	_, err := db.Exec(schema)
	return err
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./repository/ -v -run TestInitDB`
Expected: PASS

- [ ] **Step 6: Write the failing tests for TaskRepository**

Create `repository/task_repository_test.go`:

```go
package repository_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/repository"
)

func setupRepo(t *testing.T) domain.TaskRepository {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return repository.NewTaskRepository(db)
}

func makeTask() domain.Task {
	return domain.Task{
		ID:          "test-id-1",
		IssueNumber: 42,
		Title:       "Add login page",
		Body:        "Please add a login page",
		Author:      "testuser",
		Repository:  "owner/repo",
		HTMLURL:     "https://github.com/owner/repo/issues/42",
		Status:      domain.TaskStatusPending,
		CreatedAt:   time.Now().Truncate(time.Millisecond),
	}
}

func TestTaskRepository_CreateAndGetByID(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()
	task := makeTask()

	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != task.ID {
		t.Errorf("expected ID %s, got %s", task.ID, got.ID)
	}
	if got.IssueNumber != task.IssueNumber {
		t.Errorf("expected IssueNumber %d, got %d", task.IssueNumber, got.IssueNumber)
	}
	if got.Status != task.Status {
		t.Errorf("expected Status %s, got %s", task.Status, got.Status)
	}
	if got.Title != task.Title {
		t.Errorf("expected Title %s, got %s", task.Title, got.Title)
	}
}

func TestTaskRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}
}

func TestTaskRepository_UpdateStatus(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()
	task := makeTask()

	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = repo.UpdateStatus(ctx, task.ID, domain.TaskStatusRunning, "container-123")
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, _ := repo.GetByID(ctx, task.ID)
	if got.Status != domain.TaskStatusRunning {
		t.Errorf("expected running, got %s", got.Status)
	}
	if got.ContainerID != "container-123" {
		t.Errorf("expected container-123, got %s", got.ContainerID)
	}
	if got.StartedAt == nil {
		t.Error("expected non-nil StartedAt after UpdateStatus to running")
	}
}

func TestTaskRepository_AppendLog(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()
	task := makeTask()

	repo.Create(ctx, task)
	repo.AppendLog(ctx, task.ID, "line1\n")
	repo.AppendLog(ctx, task.ID, "line2\n")

	got, _ := repo.GetByID(ctx, task.ID)
	if got.Log != "line1\nline2\n" {
		t.Errorf("expected appended log, got %q", got.Log)
	}
}

func TestTaskRepository_UpdateFinished(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()
	task := makeTask()

	repo.Create(ctx, task)
	repo.UpdateFinished(ctx, task.ID, domain.TaskStatusFailed, "error output")

	got, _ := repo.GetByID(ctx, task.ID)
	if got.Status != domain.TaskStatusFailed {
		t.Errorf("expected failed, got %s", got.Status)
	}
	if got.Log != "error output" {
		t.Errorf("expected error output, got %s", got.Log)
	}
	if got.FinishedAt == nil {
		t.Error("expected non-nil FinishedAt")
	}
}

func TestTaskRepository_List_FilterByStatus(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()

	task1 := makeTask()
	task1.ID = "id-1"
	task2 := makeTask()
	task2.ID = "id-2"

	repo.Create(ctx, task1)
	repo.Create(ctx, task2)
	repo.UpdateStatus(ctx, task2.ID, domain.TaskStatusRunning, "c1")

	pending, _ := repo.List(ctx, domain.TaskStatusPending)
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}
	running, _ := repo.List(ctx, domain.TaskStatusRunning)
	if len(running) != 1 {
		t.Errorf("expected 1 running, got %d", len(running))
	}
	all, _ := repo.List(ctx, "")
	if len(all) != 2 {
		t.Errorf("expected 2 total, got %d", len(all))
	}
}

func TestTaskRepository_FindActiveByIssue(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()

	task := makeTask()
	repo.Create(ctx, task)

	active, _ := repo.FindActiveByIssue(ctx, "owner/repo", 42)
	if len(active) != 1 {
		t.Errorf("expected 1 active, got %d", len(active))
	}

	none, _ := repo.FindActiveByIssue(ctx, "other/repo", 42)
	if len(none) != 0 {
		t.Errorf("expected 0 for other repo, got %d", len(none))
	}

	// After task finishes, should not be active
	repo.UpdateFinished(ctx, task.ID, domain.TaskStatusSucceeded, "")
	finished, _ := repo.FindActiveByIssue(ctx, "owner/repo", 42)
	if len(finished) != 0 {
		t.Errorf("expected 0 active after finished, got %d", len(finished))
	}
}
```

- [ ] **Step 7: Run tests to verify they fail**

Run: `go test ./repository/ -v -run TestTaskRepository`
Expected: FAIL — NewTaskRepository does not exist

- [ ] **Step 8: Implement TaskRepository**

Create `repository/task_repository.go`:

```go
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
	defer rows.Close()

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
	defer rows.Close()

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
```

- [ ] **Step 9: Run all repository tests**

Run: `go test ./repository/ -v`
Expected: ALL PASS

- [ ] **Step 10: Commit**

```bash
git add repository/
git commit -m "feat(repository): add SQLite task repository with migration"
```

---

### Task 4: Docker Runner

**Files:**
- Create: `internal/docker/runner.go`
- Create: `internal/docker/runner_test.go`

- [ ] **Step 1: Install Docker dependency**

Run: `go get github.com/docker/docker`

- [ ] **Step 2: Write the failing test for DockerRunner**

Create `internal/docker/runner_test.go`:

```go
package docker_test

import (
	"context"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/docker"
)

func TestNewRunner_InvalidImage(t *testing.T) {
	t.Parallel()
	cfg := domain.DockerConfig{
		Image:          "nonexistent-image-that-does-not-exist:latest",
		MaxConcurrency: 1,
		Timeout:        0, // use default
		SettingsPath:   "/dev/null",
	}
	r := docker.NewRunner(cfg, "fake-api-key", "fake-github-token", "http://localhost:8080")

	_, err := r.StartContainer(context.Background(), domain.Task{
		ID:          "test-id",
		IssueNumber: 1,
		Title:       "test",
		Repository:  "owner/repo",
	})
	if err == nil {
		t.Error("expected error with nonexistent image")
	}
}

func TestNewRunner_StopNonexistent(t *testing.T) {
	t.Parallel()
	cfg := domain.DockerConfig{
		Image:          "alpine:latest",
		MaxConcurrency: 1,
		SettingsPath:   "/dev/null",
	}
	r := docker.NewRunner(cfg, "", "", "")

	err := r.StopContainer(context.Background(), "nonexistent-container-id")
	if err == nil {
		t.Error("expected error stopping nonexistent container")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/docker/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 4: Implement DockerRunner**

Create `internal/docker/runner.go`:

```go
package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

type runner struct {
	cli         *client.Client
	cfg         domain.DockerConfig
	apiKey      string
	githubToken string
	cgateURL    string
}

func NewRunner(cfg domain.DockerConfig, apiKey, githubToken, cgateURL string) domain.DockerRunner {
	cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	return &runner{
		cli:         cli,
		cfg:         cfg,
		apiKey:      apiKey,
		githubToken: githubToken,
		cgateURL:    cgateURL,
	}
}

func (r *runner) StartContainer(ctx context.Context, task domain.Task) (string, error) {
	timeout := r.cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	prompt := fmt.Sprintf("处理 Issue #%d: %s\n%s", task.IssueNumber, task.Title, task.Body)

	env := []string{
		fmt.Sprintf("ANTHROPIC_API_KEY=%s", r.apiKey),
		fmt.Sprintf("GITHUB_TOKEN=%s", r.githubToken),
		fmt.Sprintf("CGATE_URL=%s", r.cgateURL),
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: fmt.Sprintf("/tmp/cgate/repos/%s", task.ID),
			Target: "/workspace",
		},
	}

	if r.cfg.SettingsPath != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: r.cfg.SettingsPath,
			Target: "/root/.claude/settings.json",
			ReadOnly: true,
		})
	}

	resp, err := r.cli.ContainerCreate(ctx, &container.Config{
		Image: r.cfg.Image,
		Env:   env,
		Cmd:   []string{"claude", "--max-turns", "15", "--prompt", prompt},
		Tty:   false,
	}, &container.HostConfig{
		Mounts: mounts,
	}, nil, nil, fmt.Sprintf("cgate-%s", task.ID))
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	if err := r.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		r.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("start container: %w", err)
	}

	return resp.ID, nil
}

func (r *runner) StopContainer(ctx context.Context, containerID string) error {
	return r.cli.ContainerStop(ctx, containerID, container.StopOptions{})
}

func (r *runner) ContainerLogs(ctx context.Context, containerID string) (<-chan string, error) {
	reader, err := r.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("container logs: %w", err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		defer reader.Close()

		pr, pw := io.Pipe()
		go func() {
			stdcopy.StdDump(pw, reader)
			pw.Close()
		}()

		buf := make([]byte, 4096)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				ch <- string(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	return ch, nil
}

func (r *runner) WaitContainer(ctx context.Context, containerID string) (int, error) {
	statusCh, errCh := r.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case status := <-statusCh:
		return int(status.StatusCode), nil
	case err := <-errCh:
		return -1, err
	}
}

func (r *runner) IsRunning(ctx context.Context, containerID string) (bool, error) {
	inspect, err := r.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}
	return inspect.State.Running, nil
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/docker/ -v`
Expected: PASS (tests verify error cases only; integration tests require Docker daemon)

- [ ] **Step 6: Commit**

```bash
git add internal/docker/
git commit -m "feat(docker): add Docker container runner using Engine API"
```

---

### Task 5: Task Usecase — Scheduler and Business Logic

**Files:**
- Create: `usecase/task_usecase.go`
- Create: `usecase/task_usecase_test.go`

- [ ] **Step 1: Install testify**

Run: `go get github.com/stretchr/testify`

- [ ] **Step 2: Write the failing tests for TaskUsecase**

Create `usecase/task_usecase_test.go`:

```go
package usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/queue"
	"github.com/Lin-Jiong-HDU/go-project-template/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) Create(ctx context.Context, task domain.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
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
	args := m.Called(ctx, id, status, containerID)
	return args.Error(0)
}

func (m *mockRepo) AppendLog(ctx context.Context, id string, log string) error {
	args := m.Called(ctx, id, log)
	return args.Error(0)
}

func (m *mockRepo) UpdateFinished(ctx context.Context, id string, status domain.TaskStatus, log string) error {
	args := m.Called(ctx, id, status, log)
	return args.Error(0)
}

func (m *mockRepo) FindActiveByIssue(ctx context.Context, repository string, issueNumber int) ([]domain.Task, error) {
	args := m.Called(ctx, repository, issueNumber)
	return args.Get(0).([]domain.Task), args.Error(1)
}

type mockRunner struct {
	mock.Mock
	started []string
	mu      sync.Mutex
}

func (m *mockRunner) StartContainer(ctx context.Context, task domain.Task) (string, error) {
	args := m.Called(ctx, task)
	m.mu.Lock()
	m.started = append(m.started, "started")
	m.mu.Unlock()
	return args.String(0), args.Error(1)
}

func (m *mockRunner) StopContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
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

func TestHandleWebhook_CreatesAndEnqueues(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Stop()

	repo.On("FindActiveByIssue", mock.Anything, "owner/repo", 42).Return([]domain.Task{}, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Task")).Return(nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1})
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
	defer q.Stop()

	existing := domain.Task{ID: "existing", Status: domain.TaskStatusRunning}
	repo.On("FindActiveByIssue", mock.Anything, "owner/repo", 42).Return([]domain.Task{existing}, nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1})
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
	defer q.Stop()

	expected := domain.Task{ID: "task-1", Title: "test"}
	repo.On("GetByID", mock.Anything, "task-1").Return(expected, nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1})
	got, err := uc.GetTask(context.Background(), "task-1")
	assert.NoError(t, err)
	assert.Equal(t, "task-1", got.ID)
}

func TestListTasks_DelegatesToRepo(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Stop()

	tasks := []domain.Task{{ID: "1"}, {ID: "2"}}
	repo.On("List", mock.Anything, domain.TaskStatusPending).Return(tasks, nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1})
	got, err := uc.ListTasks(context.Background(), domain.TaskStatusPending)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestCancelTask_StopsRunningContainer(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	runner := new(mockRunner)
	q := queue.New()
	defer q.Stop()

	task := domain.Task{ID: "t1", Status: domain.TaskStatusRunning, ContainerID: "c1"}
	repo.On("GetByID", mock.Anything, "t1").Return(task, nil)
	runner.On("StopContainer", mock.Anything, "c1").Return(nil)
	repo.On("UpdateFinished", mock.Anything, "t1", domain.TaskStatusFailed, mock.Anything).Return(nil)

	uc := usecase.NewTaskUsecase(repo, q, runner, domain.DockerConfig{MaxConcurrency: 1})
	err := uc.CancelTask(context.Background(), "t1")
	assert.NoError(t, err)
	runner.AssertCalled(t, "StopContainer", mock.Anything, "c1")
}

func TestCancelTask_PendingTask_RemovesFromQueue(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	q := queue.New()
	defer q.Stop()

	task := domain.Task{ID: "t1", Status: domain.TaskStatusPending}
	repo.On("GetByID", mock.Anything, "t1").Return(task, nil)
	repo.On("UpdateFinished", mock.Anything, "t1", domain.TaskStatusFailed, "cancelled").Return(nil)

	uc := usecase.NewTaskUsecase(repo, q, nil, domain.DockerConfig{MaxConcurrency: 1})
	err := uc.CancelTask(context.Background(), "t1")
	assert.NoError(t, err)
	repo.AssertCalled(t, "UpdateFinished", mock.Anything, "t1", domain.TaskStatusFailed, "cancelled")
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./usecase/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 4: Implement TaskUsecase**

Create `usecase/task_usecase.go`:

```go
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

	task := domain.NewTask(payload)
	if err := u.repo.Create(ctx, task); err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
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

	return u.repo.UpdateFinished(ctx, id, domain.TaskStatusFailed, "cancelled")
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
				u.repo.UpdateFinished(ctx, task.ID, domain.TaskStatusFailed, err.Error())
				continue
			}

			u.repo.UpdateStatus(ctx, task.ID, domain.TaskStatusRunning, containerID)
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
		u.repo.AppendLog(taskCtx, task.ID, line)
	}

	exitCode, err := u.runner.WaitContainer(taskCtx, task.ContainerID)
	if err != nil {
		slog.Error("wait container", "task_id", task.ID, "error", err)
		u.repo.UpdateFinished(ctx, task.ID, domain.TaskStatusFailed, logBuf)
		return
	}

	status := domain.TaskStatusSucceeded
	if exitCode != 0 {
		status = domain.TaskStatusFailed
	}
	u.repo.UpdateFinished(ctx, task.ID, status, logBuf)
	slog.Info("task completed", "task_id", task.ID, "status", status, "exit_code", exitCode)
}
```

- [ ] **Step 5: Update queue interface — rename Stop to Close**

The queue's `Close()` method was defined as `Close()` in domain but we used `Stop()` in the test. Fix the test to use `Close()` instead:

In `usecase/task_usecase_test.go`, change all `q.Stop()` to `q.Close()`.

- [ ] **Step 6: Run tests**

Run: `go test ./usecase/ -v`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add usecase/
git commit -m "feat(usecase): add task scheduling with concurrency control"
```

---

### Task 6: Auth Middleware

**Files:**
- Create: `api/middleware/auth.go`
- Create: `api/middleware/auth_test.go`

- [ ] **Step 1: Write the failing test**

Create `api/middleware/auth_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/api/middleware"
)

func TestWebhookAuth_ValidSecret(t *testing.T) {
	t.Parallel()
	handler := middleware.WebhookAuth("my-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Webhook-Secret", "my-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestWebhookAuth_InvalidSecret(t *testing.T) {
	t.Parallel()
	handler := middleware.WebhookAuth("my-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Webhook-Secret", "wrong-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestWebhookAuth_MissingHeader(t *testing.T) {
	t.Parallel()
	handler := middleware.WebhookAuth("my-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./api/middleware/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement auth middleware**

Create `api/middleware/auth.go`:

```go
package middleware

import (
	"net/http"
)

func WebhookAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Webhook-Secret") != secret {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./api/middleware/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add api/middleware/
git commit -m "feat(middleware): add webhook secret auth middleware"
```

---

### Task 7: Webhook Controller

**Files:**
- Create: `api/controller/webhook_controller.go`
- Create: `api/controller/webhook_controller_test.go`

- [ ] **Step 1: Write the failing test**

Create `api/controller/webhook_controller_test.go`:

```go
package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/api/controller"
	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUsecase struct {
	mock.Mock
}

func (m *mockUsecase) HandleWebhook(ctx context.Context, payload domain.WebhookPayload) (domain.Task, error) {
	args := m.Called(ctx, payload)
	return args.Get(0).(domain.Task), args.Error(1)
}

func (m *mockUsecase) GetTask(ctx context.Context, id string) (domain.Task, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Task), args.Error(1)
}

func (m *mockUsecase) ListTasks(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error) {
	args := m.Called(ctx, status)
	return args.Get(0).([]domain.Task), args.Error(1)
}

func (m *mockUsecase) CancelTask(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockUsecase) GetTaskLogs(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *mockUsecase) Start(ctx context.Context) error {
	return nil
}

func (m *mockUsecase) Stop() {}

func TestWebhookHandler_ValidPayload(t *testing.T) {
	t.Parallel()
	uc := new(mockUsecase)
	handler := controller.NewWebhookHandler(uc)

	payload := domain.WebhookPayload{
		Action:      "opened",
		IssueNumber: 42,
		Title:       "Add login [claude bot]",
		Body:        "Please implement",
		Author:      "user",
		Repository:  "owner/repo",
		URL:         "https://github.com/owner/repo/issues/42",
	}
	body, _ := json.Marshal(payload)

	uc.On("HandleWebhook", mock.Anything, payload).Return(domain.Task{ID: "task-1", Status: domain.TaskStatusPending}, nil)

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	uc.AssertCalled(t, "HandleWebhook", mock.Anything, payload)
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	uc := new(mockUsecase)
	handler := controller.NewWebhookHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWebhookHandler_DuplicateIssue(t *testing.T) {
	t.Parallel()
	uc := new(mockUsecase)
	handler := controller.NewWebhookHandler(uc)

	payload := domain.WebhookPayload{IssueNumber: 42, Title: "test", Repository: "owner/repo"}
	body, _ := json.Marshal(payload)

	uc.On("HandleWebhook", mock.Anything, payload).Return(domain.Task{}, fmt.Errorf("issue already has an active task"))

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}
```

Note: The test file needs `import "context"` and `import "fmt"` — add those to the imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./api/controller/ -v -run TestWebhook`
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement WebhookHandler**

Create `api/controller/webhook_controller.go`:

```go
package controller

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

type webhookHandler struct {
	usecase domain.TaskUsecase
}

func NewWebhookHandler(uc domain.TaskUsecase) http.Handler {
	h := &webhookHandler{usecase: uc}
	return http.HandlerFunc(h.ServeHTTP)
}

func (h *webhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var payload domain.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	task, err := h.usecase.HandleWebhook(r.Context(), payload)
	if err != nil {
		if err.Error() == "issue already has an active task" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		slog.Error("handle webhook", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"id":     task.ID,
		"status": string(task.Status),
	})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./api/controller/ -v -run TestWebhook`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add api/controller/webhook_controller.go api/controller/webhook_controller_test.go
git commit -m "feat(controller): add webhook handler for GitHub Issue events"
```

---

### Task 8: Task Controller (REST API)

**Files:**
- Create: `api/controller/task_controller.go`
- Create: `api/controller/task_controller_test.go`

- [ ] **Step 1: Write the failing test**

Create `api/controller/task_controller_test.go`:

```go
package controller_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/api/controller"
	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTaskListHandler_ReturnsTasks(t *testing.T) {
	t.Parallel()
	uc := new(mockUsecase)
	handler := controller.NewTaskListHandler(uc)

	tasks := []domain.Task{{ID: "1"}, {ID: "2"}}
	uc.On("ListTasks", mock.Anything, domain.TaskStatus("")).Return(tasks, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result []domain.Task
	json.NewDecoder(rec.Body).Decode(&result)
	assert.Len(t, result, 2)
}

func TestTaskListHandler_FilterByStatus(t *testing.T) {
	t.Parallel()
	uc := new(mockUsecase)
	handler := controller.NewTaskListHandler(uc)

	uc.On("ListTasks", mock.Anything, domain.TaskStatusRunning).Return([]domain.Task{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks?status=running", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	uc.AssertCalled(t, "ListTasks", mock.Anything, domain.TaskStatusRunning)
}

func TestTaskDetailHandler_Found(t *testing.T) {
	t.Parallel()
	uc := new(mockUsecase)
	handler := controller.NewTaskDetailHandler(uc)

	task := domain.Task{ID: "task-1", Title: "test"}
	uc.On("GetTask", mock.Anything, "task-1").Return(task, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1", nil)
	req.SetPathValue("id", "task-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskDetailHandler_NotFound(t *testing.T) {
	t.Parallel()
	uc := new(mockUsecase)
	handler := controller.NewTaskDetailHandler(uc)

	uc.On("GetTask", mock.Anything, "nonexistent").Return(domain.Task{}, fmt.Errorf("not found"))

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskCancelHandler(t *testing.T) {
	t.Parallel()
	uc := new(mockUsecase)
	handler := controller.NewTaskCancelHandler(uc)

	uc.On("CancelTask", mock.Anything, "task-1").Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/task-1/cancel", nil)
	req.SetPathValue("id", "task-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskLogsHandler(t *testing.T) {
	t.Parallel()
	uc := new(mockUsecase)
	handler := controller.NewTaskLogsHandler(uc)

	uc.On("GetTaskLogs", mock.Anything, "task-1").Return("log output here", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1/logs", nil)
	req.SetPathValue("id", "task-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "log output here")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./api/controller/ -v -run TestTask`
Expected: FAIL — functions not defined

- [ ] **Step 3: Implement task controller handlers**

Create `api/controller/task_controller.go`:

```go
package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

func NewTaskListHandler(uc domain.TaskUsecase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := domain.TaskStatus(r.URL.Query().Get("status"))
		tasks, err := uc.ListTasks(r.Context(), status)
		if err != nil {
			slog.Error("list tasks", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	})
}

func NewTaskDetailHandler(uc domain.TaskUsecase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		task, err := uc.GetTask(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	})
}

func NewTaskCancelHandler(uc domain.TaskUsecase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := uc.CancelTask(r.Context(), id); err != nil {
			slog.Error("cancel task", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
	})
}

func NewTaskLogsHandler(uc domain.TaskUsecase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		logs, err := uc.GetTaskLogs(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(logs))
	})
}
```

- [ ] **Step 4: Run all controller tests**

Run: `go test ./api/controller/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add api/controller/task_controller.go api/controller/task_controller_test.go
git commit -m "feat(controller): add task REST API handlers"
```

---

### Task 9: Route Registration and DI Wiring

**Files:**
- Create: `api/route/route.go`
- Create: `bootstrap/bootstrap.go`
- Create: `cmd/main.go`
- Modify: `config.yaml`

- [ ] **Step 1: Create route registration**

Create `api/route/route.go`:

```go
package route

import (
	"net/http"

	"github.com/Lin-Jiong-HDU/go-project-template/api/controller"
	"github.com/Lin-Jiong-HDU/go-project-template/api/middleware"
	"github.com/Lin-Jiong-HDU/go-project-template/domain"
)

func NewMux(uc domain.TaskUsecase, webhookSecret string) *http.ServeMux {
	mux := http.NewServeMux()

	webhookHandler := middleware.WebhookAuth(webhookSecret)(controller.NewWebhookHandler(uc))
	mux.Handle("POST /webhook/github", webhookHandler)

	mux.Handle("GET /api/tasks", controller.NewTaskListHandler(uc))
	mux.Handle("GET /api/tasks/{id}", controller.NewTaskDetailHandler(uc))
	mux.Handle("POST /api/tasks/{id}/cancel", controller.NewTaskCancelHandler(uc))
	mux.Handle("GET /api/tasks/{id}/logs", controller.NewTaskLogsHandler(uc))

	return mux
}
```

- [ ] **Step 2: Install viper and create bootstrap**

Run: `go get github.com/spf13/viper`

Create `bootstrap/bootstrap.go`:

```go
package bootstrap

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/spf13/viper"
	"github.com/Lin-Jiong-HDU/go-project-template/api/route"
	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/docker"
	"github.com/Lin-Jiong-HDU/go-project-template/internal/queue"
	"github.com/Lin-Jiong-HDU/go-project-template/repository"
	"github.com/Lin-Jiong-HDU/go-project-template/usecase"
)

type App struct {
	Server *http.Server
	UC     domain.TaskUsecase
	DB     *sql.DB
}

func Init() (*App, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/app")

	viper.AutomaticEnv()
	viper.BindEnv("github.webhook_secret", "GITHUB_WEBHOOK_SECRET")
	viper.BindEnv("github.pat", "GITHUB_PAT")
	viper.BindEnv("docker.settings_path", "DOCKER_SETTINGS_PATH")

	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("no config file found, using defaults and env vars")
	}

	var cfg domain.Config
	cfg.Server.Port = viper.GetInt("server.port")
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	cfg.Docker.Image = viper.GetString("docker.image")
	if cfg.Docker.Image == "" {
		cfg.Docker.Image = "claude-code-runner:latest"
	}
	cfg.Docker.MaxConcurrency = viper.GetInt("docker.max_concurrency")
	if cfg.Docker.MaxConcurrency == 0 {
		cfg.Docker.MaxConcurrency = 3
	}
	cfg.Docker.Timeout = viper.GetDuration("docker.timeout")
	if cfg.Docker.Timeout == 0 {
		cfg.Docker.Timeout = 30 * 1e9 // 30 minutes
	}
	cfg.Docker.SettingsPath = viper.GetString("docker.settings_path")
	cfg.Queue.MaxRetries = viper.GetInt("queue.max_retries")
	if cfg.Queue.MaxRetries == 0 {
		cfg.Queue.MaxRetries = 1
	}
	cfg.GitHub.WebhookSecret = viper.GetString("github.webhook_secret")
	cfg.GitHub.PAT = viper.GetString("github.pat")

	dbPath := viper.GetString("database.path")
	if dbPath == "" {
		dbPath = "./data/cgate.db"
	}
	os.MkdirAll("./data", 0755)

	db, err := repository.InitDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	taskRepo := repository.NewTaskRepository(db)
	taskQueue := queue.New()
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	cgateURL := os.Getenv("CGATE_URL")
	runner := docker.NewRunner(cfg.Docker, apiKey, cfg.GitHub.PAT, cgateURL)

	uc := usecase.NewTaskUsecase(taskRepo, taskQueue, runner, cfg.Docker)

	mux := route.NewMux(uc, cfg.GitHub.WebhookSecret)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &App{
		Server: server,
		UC:     uc,
		DB:     db,
	}, nil
}
```

- [ ] **Step 3: Update cmd/main.go**

Replace `main.go` contents:

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Lin-Jiong-HDU/go-project-template/bootstrap"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	app, err := bootstrap.Init()
	if err != nil {
		return err
	}
	defer app.DB.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.UC.Start(ctx); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", app.Server.Addr)
		if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
	case err := <-errCh:
		return err
	}

	app.UC.Stop()
	return app.Server.Shutdown(context.Background())
}
```

Note: needs `import "net/http"` for `http.ErrServerClosed`.

- [ ] **Step 4: Create default config.yaml**

```yaml
server:
  port: 8080

docker:
  image: "claude-code-runner:latest"
  max_concurrency: 3
  timeout: "30m"
  settings_path: "./settings.json"

github:
  webhook_secret: "${GITHUB_WEBHOOK_SECRET}"
  pat: "${GITHUB_PAT}"

queue:
  max_retries: 1
```

- [ ] **Step 5: Run build to verify compilation**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 6: Run all tests**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add api/route/ bootstrap/ cmd/main.go config.yaml
git rm main.go
git commit -m "feat: wire DI, add bootstrap, config, and route registration"
```

---

### Task 10: Dockerfile and docker-compose.yml

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Modify: `Makefile`

- [ ] **Step 1: Create Dockerfile**

```dockerfile
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /cgate ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates docker-cli
COPY --from=builder /cgate /usr/local/bin/cgate

EXPOSE 8080
ENTRYPOINT ["cgate"]
```

- [ ] **Step 2: Create docker-compose.yml**

```yaml
services:
  cgate:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./settings.json:/app/settings.json:ro
      - ./data:/app/data
    environment:
      - GITHUB_WEBHOOK_SECRET=${GITHUB_WEBHOOK_SECRET}
      - GITHUB_PAT=${GITHUB_PAT}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - CGATE_URL=${CGATE_URL}
    restart: unless-stopped
```

- [ ] **Step 3: Update Makefile**

Replace `Makefile` contents:

```makefile
.PHONY: build test lint run tidy clean docker-build docker-up docker-down

BINARY_NAME=cgate

build:
	go build -o bin/$(BINARY_NAME) ./cmd/

test:
	go test ./... -v -cover

lint:
	golangci-lint run

run:
	go run ./cmd/

tidy:
	go mod tidy

clean:
	rm -rf bin/

docker-build:
	docker build -t cgate:latest .

docker-up:
	docker compose up -d

docker-down:
	docker compose down
```

- [ ] **Step 4: Run vet and build**

Run: `go vet ./... && go build ./...`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add Dockerfile docker-compose.yml Makefile
git commit -m "build: add Dockerfile, docker-compose, and update Makefile"
```

---

### Task 11: Pipeline Gates — Final Verification

**Files:** None (verification only)

- [ ] **Step 1: Run go vet**

Run: `go vet ./...`
Expected: Zero output

- [ ] **Step 2: Run go build**

Run: `go build ./...`
Expected: Zero output

- [ ] **Step 3: Run go test**

Run: `go test ./... -cover`
Expected: All tests pass

- [ ] **Step 4: Run golangci-lint**

Run: `golangci-lint run`
Expected: No new violations

- [ ] **Step 5: Fix any issues found and re-run from Step 1**

---

## Self-Review

**Spec coverage:**
- Architecture (event-driven + queue): Task 2, 4, 5
- Data model (Task, Config): Task 1
- SQLite persistence: Task 3
- Docker container management: Task 4
- Scheduler + concurrency: Task 5
- Webhook endpoint: Task 7
- REST API endpoints: Task 8
- Auth middleware: Task 6
- Route + DI: Task 9
- Deployment (Dockerfile, compose): Task 10
- Pipeline gates: Task 11

**Placeholder scan:** No TBD/TODO found. All steps contain complete code.

**Type consistency:** All interfaces defined in Task 1 are implemented in subsequent tasks. Method signatures match across mock implementations and concrete types. `domain.TaskQueue.Close()` used consistently.

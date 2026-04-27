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
	t.Cleanup(func() { _ = db.Close() })
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

	got, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
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

	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.AppendLog(ctx, task.ID, "line1\n"); err != nil {
		t.Fatalf("AppendLog line1: %v", err)
	}
	if err := repo.AppendLog(ctx, task.ID, "line2\n"); err != nil {
		t.Fatalf("AppendLog line2: %v", err)
	}

	got, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Log != "line1\nline2\n" {
		t.Errorf("expected appended log, got %q", got.Log)
	}
}

func TestTaskRepository_UpdateFinished(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()
	task := makeTask()

	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.UpdateFinished(ctx, task.ID, domain.TaskStatusFailed, "error output"); err != nil {
		t.Fatalf("UpdateFinished: %v", err)
	}

	got, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
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

	if err := repo.Create(ctx, task1); err != nil {
		t.Fatalf("Create task1: %v", err)
	}
	if err := repo.Create(ctx, task2); err != nil {
		t.Fatalf("Create task2: %v", err)
	}
	if err := repo.UpdateStatus(ctx, task2.ID, domain.TaskStatusRunning, "c1"); err != nil {
		t.Fatalf("UpdateStatus task2: %v", err)
	}

	pending, err := repo.List(ctx, domain.TaskStatusPending)
	if err != nil {
		t.Fatalf("List pending: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}
	running, err := repo.List(ctx, domain.TaskStatusRunning)
	if err != nil {
		t.Fatalf("List running: %v", err)
	}
	if len(running) != 1 {
		t.Errorf("expected 1 running, got %d", len(running))
	}
	all, err := repo.List(ctx, "")
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 total, got %d", len(all))
	}
}

func TestTaskRepository_FindActiveByIssue(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()

	task := makeTask()
	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("Create: %v", err)
	}

	active, err := repo.FindActiveByIssue(ctx, "owner/repo", 42)
	if err != nil {
		t.Fatalf("FindActiveByIssue: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("expected 1 active, got %d", len(active))
	}

	none, err := repo.FindActiveByIssue(ctx, "other/repo", 42)
	if err != nil {
		t.Fatalf("FindActiveByIssue other: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected 0 for other repo, got %d", len(none))
	}

	if err := repo.UpdateFinished(ctx, task.ID, domain.TaskStatusSucceeded, ""); err != nil {
		t.Fatalf("UpdateFinished: %v", err)
	}
	finished, err := repo.FindActiveByIssue(ctx, "owner/repo", 42)
	if err != nil {
		t.Fatalf("FindActiveByIssue after finish: %v", err)
	}
	if len(finished) != 0 {
		t.Errorf("expected 0 active after finished, got %d", len(finished))
	}
}

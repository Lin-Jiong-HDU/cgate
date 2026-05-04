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
		TaskType:    domain.TaskTypeIssue,
	}
}

func makePRReviewTask() domain.Task {
	return domain.Task{
		ID:          "pr-task-1",
		IssueNumber: 0,
		Title:       "Fix review comments",
		Body:        "/claude fix-review",
		Author:      "reviewer",
		Repository:  "owner/repo",
		HTMLURL:     "https://github.com/owner/repo/pull/55",
		Status:      domain.TaskStatusPending,
		CreatedAt:   time.Now().Truncate(time.Millisecond),
		TaskType:    domain.TaskTypePRReview,
		PRNumber:    55,
		CommentID:   99999,
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
	if got.TaskType != domain.TaskTypeIssue {
		t.Errorf("expected TaskType issue, got %s", got.TaskType)
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

func TestTaskRepository_FindActiveByPR(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()

	task := makePRReviewTask()
	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("Create: %v", err)
	}

	active, err := repo.FindActiveByPR(ctx, "owner/repo", 55)
	if err != nil {
		t.Fatalf("FindActiveByPR: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("expected 1 active, got %d", len(active))
	}
	if active[0].TaskType != domain.TaskTypePRReview {
		t.Errorf("expected TaskType pr_review, got %s", active[0].TaskType)
	}
	if active[0].PRNumber != 55 {
		t.Errorf("expected PRNumber 55, got %d", active[0].PRNumber)
	}
	if active[0].CommentID != 99999 {
		t.Errorf("expected CommentID 99999, got %d", active[0].CommentID)
	}

	none, err := repo.FindActiveByPR(ctx, "owner/repo", 99)
	if err != nil {
		t.Fatalf("FindActiveByPR other PR: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected 0 for other PR, got %d", len(none))
	}

	if err := repo.UpdateFinished(ctx, task.ID, domain.TaskStatusSucceeded, ""); err != nil {
		t.Fatalf("UpdateFinished: %v", err)
	}
	finished, err := repo.FindActiveByPR(ctx, "owner/repo", 55)
	if err != nil {
		t.Fatalf("FindActiveByPR after finish: %v", err)
	}
	if len(finished) != 0 {
		t.Errorf("expected 0 active after finished, got %d", len(finished))
	}
}

func TestTaskRepository_FindActiveByPR_DoesNotReturnIssueTasks(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()

	issueTask := makeTask()
	issueTask.IssueNumber = 55
	if err := repo.Create(ctx, issueTask); err != nil {
		t.Fatalf("Create issue task: %v", err)
	}

	active, err := repo.FindActiveByPR(ctx, "owner/repo", 55)
	if err != nil {
		t.Fatalf("FindActiveByPR: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("expected 0 active for PR lookup on issue task, got %d", len(active))
	}
}

func TestTaskRepository_PRReviewTask_RoundTrip(t *testing.T) {
	t.Parallel()
	repo := setupRepo(t)
	ctx := context.Background()
	task := makePRReviewTask()

	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.TaskType != domain.TaskTypePRReview {
		t.Errorf("expected TaskType pr_review, got %s", got.TaskType)
	}
	if got.PRNumber != 55 {
		t.Errorf("expected PRNumber 55, got %d", got.PRNumber)
	}
	if got.CommentID != 99999 {
		t.Errorf("expected CommentID 99999, got %d", got.CommentID)
	}
}

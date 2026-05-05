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

	task, err := domain.NewTask(payload)
	if err != nil {
		t.Fatalf("NewTask returned error: %v", err)
	}

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
	if task.TaskType != domain.TaskTypeIssue {
		t.Errorf("expected TaskType issue, got %s", task.TaskType)
	}
	if task.PRNumber != 0 {
		t.Errorf("expected PRNumber 0, got %d", task.PRNumber)
	}
	if task.CommentID != 0 {
		t.Errorf("expected CommentID 0, got %d", task.CommentID)
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

func TestNewTask_PRReviewPayload(t *testing.T) {
	t.Parallel()
	payload := domain.WebhookPayload{
		Action:      "created",
		TriggerType: "pr_review",
		PRNumber:    99,
		CommentID:   12345,
		Title:       "Fix the bug",
		Body:        "/claude fix-review",
		Author:      "reviewer",
		Repository:  "owner/repo",
		URL:         "https://github.com/owner/repo/pull/99",
	}

	task, err := domain.NewTask(payload)
	if err != nil {
		t.Fatalf("NewTask returned error: %v", err)
	}

	if task.TaskType != domain.TaskTypePRReview {
		t.Errorf("expected TaskType pr_review, got %s", task.TaskType)
	}
	if task.PRNumber != 99 {
		t.Errorf("expected PRNumber 99, got %d", task.PRNumber)
	}
	if task.CommentID != 12345 {
		t.Errorf("expected CommentID 12345, got %d", task.CommentID)
	}
}

func TestNewTask_GeneratesUniqueIDs(t *testing.T) {
	t.Parallel()
	payload := domain.WebhookPayload{IssueNumber: 1}
	task1, _ := domain.NewTask(payload)
	task2, _ := domain.NewTask(payload)

	if task1.ID == task2.ID {
		t.Error("expected unique IDs")
	}
}

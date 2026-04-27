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
	task1, _ := domain.NewTask(payload)
	task2, _ := domain.NewTask(payload)

	if task1.ID == task2.ID {
		t.Error("expected unique IDs")
	}
}

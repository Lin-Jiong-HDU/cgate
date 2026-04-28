package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/api/controller"
	"github.com/Lin-Jiong-HDU/go-project-template/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockUsecase implements domain.TaskUsecase for testing
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
	return m.Called(ctx, id).Error(0)
}

func (m *mockUsecase) GetTaskLogs(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *mockUsecase) Start(ctx context.Context) error { return nil }
func (m *mockUsecase) Stop()                          {}

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

	uc.On("HandleWebhook", mock.Anything, payload).Return(domain.Task{}, domain.ErrActiveTaskExists)

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

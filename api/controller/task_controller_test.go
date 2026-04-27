package controller_test

import (
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
	assert.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
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

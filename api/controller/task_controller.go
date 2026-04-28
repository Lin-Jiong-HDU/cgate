package controller

import (
	"encoding/json"
	"errors"
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
		if err := json.NewEncoder(w).Encode(tasks); err != nil {
			slog.Error("encode response", "error", err)
		}
	})
}

func NewTaskDetailHandler(uc domain.TaskUsecase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		task, err := uc.GetTask(r.Context(), id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			slog.Error("get task", "error", err, "id", id)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(task); err != nil {
			slog.Error("encode response", "error", err)
		}
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
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"}); err != nil {
			slog.Error("encode response", "error", err)
		}
	})
}

func NewTaskLogsHandler(uc domain.TaskUsecase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		logs, err := uc.GetTaskLogs(r.Context(), id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			slog.Error("get task logs", "error", err, "id", id)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := w.Write([]byte(logs)); err != nil {
			slog.Error("write response", "error", err)
		}
	})
}

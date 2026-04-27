package controller

import (
	"encoding/json"
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
		errMsg := err.Error()
		if errMsg == "issue already has an active task" {
			http.Error(w, errMsg, http.StatusConflict)
			return
		}
		slog.Error("handle webhook", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"id":     task.ID,
		"status": string(task.Status),
	}); err != nil {
		slog.Error("encode response", "error", err)
	}
}

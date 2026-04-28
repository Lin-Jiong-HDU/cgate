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

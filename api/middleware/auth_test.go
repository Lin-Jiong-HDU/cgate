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

func TestWebhookAuth_EmptySecret(t *testing.T) {
	t.Parallel()
	handler := middleware.WebhookAuth("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

package middleware

import (
	"crypto/subtle"
	"net/http"
)

func WebhookAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				http.Error(w, "webhook secret not configured", http.StatusInternalServerError)
				return
			}
			provided := r.Header.Get("X-Webhook-Secret")
			if subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

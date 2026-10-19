package middleware

import (
	"log/slog"
	"net/http"
)

func MaxBytesMiddleware(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("MaxBytesMiddleware: before", "path", r.URL.Path, "content-length", r.ContentLength)
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
			slog.Info("MaxBytesMiddleware: after", "path", r.URL.Path)
		})
	}
}

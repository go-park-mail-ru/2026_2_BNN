package middleware

import (
	"bnn/internal/pkg/ctxutil"
	"net/http"

	uuid "github.com/satori/go.uuid"
)

type ctxKey string

const RequestIDKey ctxKey = "request_id"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.NewV4().String()

		ctx := ctxutil.WithRequestID(r.Context(), reqID)

		w.Header().Set("X-Request-Id", reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

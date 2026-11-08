package middleware

import (
	"net/http"

	"bnn/internal/pkg/ctxutil"

	uuid "github.com/satori/go.uuid"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.NewV4().String()

		ctx := ctxutil.WithRequestID(r.Context(), reqID)

		w.Header().Set("X-Request-Id", reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

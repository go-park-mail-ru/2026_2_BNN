package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/pkg/middleware"

	"github.com/stretchr/testify/assert"
)

func TestMaxBytesMiddleware(t *testing.T) {
	limit := int64(10)
	mw := middleware.MaxBytesMiddleware(limit)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 20)
		_, err := r.Body.Read(buf)
		if err != nil && err.Error() == "http: request body too large" {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Body within limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("12345")))
		rec := httptest.NewRecorder()

		mw(nextHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Body exceeds limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("12345678901234567890")))
		rec := httptest.NewRecorder()

		mw(nextHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})
}

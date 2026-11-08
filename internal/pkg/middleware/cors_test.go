package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/pkg/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCORS(t *testing.T) {
	allowedOrigin := "http://localhost:3000"
	mw, err := middleware.NewCORS(allowedOrigin)
	require.NoError(t, err)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Allowed Origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", allowedOrigin)
		rec := httptest.NewRecorder()

		mw(nextHandler).ServeHTTP(rec, req)

		assert.Equal(t, allowedOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("Disallowed Origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://evil.com")
		rec := httptest.NewRecorder()

		mw(nextHandler).ServeHTTP(rec, req)

		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("Preflight OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", allowedOrigin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()

		mw(nextHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
	})

	t.Run("No Origins Error", func(t *testing.T) {
		_, err := middleware.NewCORS("  ,  ")
		assert.ErrorIs(t, err, middleware.ErrNoOrigins)
	})
}

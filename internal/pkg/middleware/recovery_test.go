package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/pkg/middleware"

	"github.com/stretchr/testify/assert"
)

func TestRecovery(t *testing.T) {
	tests := []struct {
		name     string
		handler  http.HandlerFunc
		expected int
	}{
		{
			name:     "panic",
			handler:  func(w http.ResponseWriter, r *http.Request) { panic("boom") },
			expected: http.StatusInternalServerError,
		},
		{
			name:     "no panic",
			handler:  func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) },
			expected: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			middleware.Recovery(tt.handler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expected, rec.Code)
		})
	}
}

func TestRecoveryAbortHandler(t *testing.T) {
	handler := middleware.Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})
}

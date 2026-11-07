package cookies_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/pkg/cookies"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCookies(t *testing.T) {
	t.Run("Set and Middleware", func(t *testing.T) {
		mw := cookies.Middleware

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookies.Set(r.Context(), &http.Cookie{Name: "test_cookie", Value: "123"})
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mw(nextHandler).ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		found := false
		for _, c := range resp.Cookies() {
			if c.Name == "test_cookie" && c.Value == "123" {
				found = true
				break
			}
		}
		assert.True(t, found, "cookie should be set in response")
	})

	t.Run("Get", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})

		mw := cookies.Middleware
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			val, err := cookies.Get(r.Context(), "session")
			require.NoError(t, err)
			assert.Equal(t, "abc", val)
		})

		rec := httptest.NewRecorder()
		mw(nextHandler).ServeHTTP(rec, req)
	})
}

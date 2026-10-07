package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/models"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentUser(t *testing.T) {
	tests := []struct {
		name         string
		prepare      func(t *testing.T, s *Service) *http.Request
		expectedCode int
		expectLogin  string
	}{
		{
			name: "OK: authenticated user gets own profile",
			prepare: func(t *testing.T, s *Service) *http.Request {
				t.Helper()

				u := addTestUser(t, s, "profileuser", "password123")

				req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
				req = req.WithContext(
					context.WithValue(req.Context(), UserIDKey, u.ID.String()),
				)

				return req
			},
			expectedCode: http.StatusOK,
			expectLogin:  "profileuser",
		},
		{
			name: "Unauthorized: no userID in context",
			prepare: func(t *testing.T, s *Service) *http.Request {
				t.Helper()

				return httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Not found: userID not in storage",
			prepare: func(t *testing.T, s *Service) *http.Request {
				t.Helper()

				req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
				req = req.WithContext(
					context.WithValue(
						req.Context(),
						UserIDKey,
						"00000000-0000-0000-0000-000000000000",
					),
				)

				return req
			},
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService(testSecret)
			req := tt.prepare(t, s)

			rec := httptest.NewRecorder()
			s.GetCurrentUser(rec, req)

			require.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectedCode != http.StatusOK {
				assert.Empty(t, rec.Body.String())

				return
			}

			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var u models.User
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&u))

			assert.Equal(t, tt.expectLogin, u.Login)
			assert.NotEqual(t, "", u.ID.String())
			assert.Nil(t, u.PasswordHash, "password hash must not be exposed")
			assert.Equal(t, "default_avatar.jpg", u.Avatar)
			assert.Equal(t, 1, u.Version)
		})
	}
}

func TestGetCurrentUser_ViaRouter(t *testing.T) {
	s := NewService(testSecret)
	u := addTestUser(t, s, "routeruser", "password123")

	token, err := s.GenerateToken(u.ID.String(), u.Version)
	require.NoError(t, err)

	router := mux.NewRouter()
	router.Handle(
		"/api/auth/me",
		s.Middleware(http.HandlerFunc(s.GetCurrentUser)),
	).Methods(http.MethodGet)

	t.Run("OK with valid cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: CookieName, Value: token})

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var got models.User
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))

		assert.Equal(t, "routeruser", got.Login)
		assert.Equal(t, u.ID.String(), got.ID.String())
		assert.Nil(t, got.PasswordHash)
	})

	t.Run("Unauthorized without cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Unauthorized with invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: CookieName, Value: "not-a-jwt"})

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Unauthorized with stale version", func(t *testing.T) {
		stale, err := s.GenerateToken(u.ID.String(), u.Version+1)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: CookieName, Value: stale})

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	auth.Init("test-secret-for-unit-tests-only")
	os.Exit(m.Run())
}

func createUser(t *testing.T, login, password string) (string, *http.Cookie) {
	t.Helper()

	body := strings.NewReader(`{"login":"` + login + `","password":"` + password + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", body)
	rec := httptest.NewRecorder()

	auth.SignUp(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "signup must succeed")

	var user models.User
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&user))

	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			cookie = c
			break
		}
	}
	require.NotNil(t, cookie, "signup must set auth cookie")

	return user.ID.String(), cookie
}

func TestGetCurrentUser(t *testing.T) {
	tests := []struct {
		name         string
		prepare      func(t *testing.T) (*http.Request, string)
		expectedCode int
		expectLogin  string
	}{
		{
			name: "OK: authenticated user gets own profile",
			prepare: func(t *testing.T) (*http.Request, string) {
				userID, _ := createUser(t, "profileuser", "password123")
				req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
				req = req.WithContext(
					context.WithValue(req.Context(), auth.UserIDKey, userID),
				)
				return req, "profileuser"
			},
			expectedCode: http.StatusOK,
			expectLogin:  "profileuser",
		},
		{
			name: "Unauthorized: no userID in context",
			prepare: func(t *testing.T) (*http.Request, string) {
				req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
				return req, ""
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Not found: userID not in storage",
			prepare: func(t *testing.T) (*http.Request, string) {
				req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
				req = req.WithContext(
					context.WithValue(
						req.Context(),
						auth.UserIDKey,
						"00000000-0000-0000-0000-000000000000",
					),
				)
				return req, ""
			},
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, expectedLogin := tt.prepare(t)

			rec := httptest.NewRecorder()
			GetCurrentUser(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectedCode == http.StatusOK {
				var user models.User
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&user))

				assert.Equal(t, expectedLogin, user.Login)
				assert.NotEqual(t, "", user.ID.String())
				assert.Nil(t, user.PasswordHash, "password hash must not be exposed")
				assert.Equal(t, "default_avatar.jpg", user.Avatar)
				assert.Equal(t, 1, user.Version)
			}
		})
	}
}

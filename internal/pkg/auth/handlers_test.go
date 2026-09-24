package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain инициализирует jwtSecret один раз для всех тестов пакета.
// Без этого GenerateToken вернёт "jwt secret is not initialized".
func TestMain(m *testing.M) {
	Init("test-secret-for-unit-tests-only")
	os.Exit(m.Run())
}

func resetUsers() {
	usersMu.Lock()
	defer usersMu.Unlock()
	users = make(map[string]models.User)
}

func addTestUser(t *testing.T, login, password string) models.User {
	t.Helper()

	hash, err := hashPassword(password)
	require.NoError(t, err)

	now := time.Now().UTC()
	user := models.User{
		ID:           uuid.NewV4(),
		Login:        login,
		PasswordHash: hash,
		Avatar:       "/static/default_avatar",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	usersMu.Lock()
	users[login] = user
	usersMu.Unlock()

	return user
}

func TestSignIn(t *testing.T) {
	tests := []struct {
		name         string
		prepare      func(t *testing.T)
		body         string
		expectedCode int
		expectCookie bool
		expectNoHash bool
	}{
		{
			name:         "OK sign in with valid credentials",
			prepare:      func(t *testing.T) { resetUsers(); addTestUser(t, "testuser", "password123") },
			body:         `{"login":"testuser","password":"password123"}`,
			expectedCode: http.StatusOK,
			expectCookie: true,
			expectNoHash: true,
		},
		{
			name:         "Sign in with unknown login",
			prepare:      func(t *testing.T) { resetUsers() },
			body:         `{"login":"nobody","password":"password123"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Sign in with wrong password",
			prepare:      func(t *testing.T) { resetUsers(); addTestUser(t, "testuser", "password123") },
			body:         `{"login":"testuser","password":"wrong-password"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Sign in with invalid JSON",
			prepare:      func(t *testing.T) { resetUsers() },
			body:         `{"login":"testuser","password":`,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			SignIn(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectCookie {
				var found *http.Cookie
				for _, c := range rec.Result().Cookies() {
					if c.Name == CookieName {
						found = c
						break
					}
				}
				require.NotNil(t, found, "auth cookie must be set")
				assert.NotEmpty(t, found.Value)
				assert.True(t, found.HttpOnly)
			}

			if tt.expectNoHash {
				var user models.User
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&user))
				assert.Nil(t, user.PasswordHash)
				assert.Equal(t, "testuser", user.Login)
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	hash, err := hashPassword("correct-password")
	require.NoError(t, err)

	assert.True(t, verifyPassword("correct-password", hash))
	assert.False(t, verifyPassword("wrong-password", hash))
	assert.False(t, verifyPassword("correct-password", []byte("garbage")))
}

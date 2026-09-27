package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
		Avatar:       "default_avatar.jpg",
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

func TestSignUp(t *testing.T) {
	makeBody := func(login, password string) string {
		body, err := json.Marshal(map[string]string{
			"login":    login,
			"password": password,
		})
		require.NoError(t, err)

		return string(body)
	}

	validBody := makeBody("testuser", "password123")

	tests := []struct {
		name         string
		body         string
		existingUser bool
		expectedCode int
	}{
		{
			name:         "successful signup",
			body:         validBody,
			expectedCode: http.StatusCreated,
		},
		{
			name:         "login already exists",
			body:         validBody,
			existingUser: true,
			expectedCode: http.StatusConflict,
		},
		{
			name:         "minimum lengths",
			body:         makeBody("abc", "12345678"),
			expectedCode: http.StatusCreated,
		},
		{
			name:         "maximum lengths",
			body:         makeBody(strings.Repeat("a", 20), strings.Repeat("b", 128)),
			expectedCode: http.StatusCreated,
		},
		{
			name:         "unicode login",
			body:         makeBody(strings.Repeat("я", 20), "password123"),
			expectedCode: http.StatusCreated,
		},
		{
			name:         "login too short",
			body:         makeBody("ab", "password123"),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "login too long",
			body:         makeBody(strings.Repeat("a", 21), "password123"),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "password too short",
			body:         makeBody("testuser", "1234567"),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "password too long",
			body:         makeBody("testuser", strings.Repeat("a", 129)),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid JSON",
			body:         `{"login":`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "missing fields",
			body:         `{}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "two JSON objects",
			body:         validBody + `{}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "body too large",
			body:         validBody + strings.Repeat(" ", 4096),
			expectedCode: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUsers()
			t.Cleanup(resetUsers)

			if tt.existingUser {
				addTestUser(t, "testuser", "password123")
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/auth/signup",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			SignUp(rec, req)

			require.Equal(t, tt.expectedCode, rec.Code)

			usersMu.RLock()
			userCount := len(users)
			usersMu.RUnlock()

			if tt.expectedCode != http.StatusCreated {
				assert.Empty(t, rec.Body.String())
				assert.Empty(t, rec.Result().Cookies())

				if tt.existingUser {
					assert.Equal(t, 1, userCount)
				} else {
					assert.Equal(t, 0, userCount)
				}

				return
			}

			assert.Equal(t, 1, userCount)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var response map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

			assert.NotContains(t, response, "password")
			assert.NotContains(t, response, "password_hash")
			assert.NotContains(t, response, "PasswordHash")
			assert.Equal(t, "default_avatar.jpg", response["avatar"])

			var sent struct {
				Login    string `json:"login"`
				Password string `json:"password"`
			}
			require.NoError(t, json.Unmarshal([]byte(tt.body), &sent))

			assert.Equal(t, sent.Login, response["login"])

			usersMu.RLock()
			storedUser, exists := users[sent.Login]
			usersMu.RUnlock()

			require.True(t, exists)
			assert.Equal(t, storedUser.ID.String(), response["id"])
			assert.NotEmpty(t, storedUser.PasswordHash)
			assert.NotEqual(t, sent.Password, string(storedUser.PasswordHash))
			assert.True(t, verifyPassword(sent.Password, storedUser.PasswordHash))
			assert.False(t, storedUser.CreatedAt.IsZero())
			assert.Equal(t, storedUser.CreatedAt, storedUser.UpdatedAt)

			var authCookie *http.Cookie
			for _, cookie := range rec.Result().Cookies() {
				if cookie.Name == CookieName {
					authCookie = cookie
					break
				}
			}

			require.NotNil(t, authCookie)
			assert.True(t, authCookie.HttpOnly)
			assert.Equal(t, "/", authCookie.Path)

			claims, err := ParseToken(authCookie.Value)
			require.NoError(t, err)
			assert.Equal(t, storedUser.ID.String(), claims.UserID)
		})
	}
}

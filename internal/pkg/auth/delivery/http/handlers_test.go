package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-for-unit-tests-only"

func addTestUser(t *testing.T, s *Service, login, password string) models.User {
	t.Helper()

	hash, err := hashPassword(password)
	require.NoError(t, err)

	now := time.Now().UTC()
	user := models.User{
		ID:           uuid.NewV4(),
		Login:        login,
		PasswordHash: hash,
		Avatar:       "default_avatar.jpg",
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.mu.Lock()
	s.users[login] = user
	s.usersByID[user.ID.String()] = user
	s.mu.Unlock()

	return user
}

func TestSignIn(t *testing.T) {
	withUser := func(t *testing.T, s *Service) {
		addTestUser(t, s, "testuser", "password123")
	}

	tests := []struct {
		name         string
		prepare      func(t *testing.T, s *Service)
		body         string
		expectedCode int
		expectCookie bool
		expectNoHash bool
	}{
		{
			name:         "OK sign in with valid credentials",
			prepare:      withUser,
			body:         `{"login":"testuser","password":"password123"}`,
			expectedCode: http.StatusOK,
			expectCookie: true,
			expectNoHash: true,
		},
		{
			name:         "Sign in with unknown login",
			body:         `{"login":"nobody","password":"password123"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Sign in with wrong password",
			prepare:      withUser,
			body:         `{"login":"testuser","password":"wrong-password"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Sign in with invalid JSON",
			body:         `{"login":"testuser","password":`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Sign in with missing fields",
			body:         `{}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Sign in with empty fields",
			body:         `{"login":"","password":""}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Sign in with too short login",
			prepare:      withUser,
			body:         `{"login":"ab","password":"password123"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Sign in with too short password",
			prepare:      withUser,
			body:         `{"login":"testuser","password":"short"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Sign in with wrong field type",
			body:         `{"login":123,"password":"password123"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Sign in with oversized body",
			body: `{"login":"testuser","password":"password123"}` +
				strings.Repeat(" ", maxRequestBodySize),
			expectedCode: http.StatusRequestEntityTooLarge,
		},
		{
			name:         "Sign in with two JSON objects",
			body:         `{"login":"testuser","password":"password123"}{}`,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService(testSecret)

			if tt.prepare != nil {
				tt.prepare(t, s)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			s.SignIn(rec, req)

			require.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectedCode != http.StatusOK {
				assert.Empty(t, rec.Body.String())
				assert.Empty(t, rec.Result().Cookies())

				return
			}

			assert.Equal(
				t,
				"application/json",
				rec.Header().Get("Content-Type"),
			)

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
				var response map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

				assert.NotContains(t, response, "password")
				assert.NotContains(t, response, "password_hash")
				assert.NotContains(t, response, "PasswordHash")

				assert.Equal(t, "testuser", response["login"])
				assert.Equal(t, "default_avatar.jpg", response["avatar"])
				assert.NotEmpty(t, response["id"])
				assert.NotEmpty(t, response["created_at"])
				assert.NotEmpty(t, response["updated_at"])
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
			s := NewService(testSecret)

			if tt.existingUser {
				addTestUser(t, s, "testuser", "password123")
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/auth/signup",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			s.SignUp(rec, req)

			require.Equal(t, tt.expectedCode, rec.Code)

			s.mu.RLock()
			userCount := len(s.users)
			s.mu.RUnlock()

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

			s.mu.RLock()
			storedUser, exists := s.users[sent.Login]
			s.mu.RUnlock()

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

			claims, err := s.ParseToken(authCookie.Value)
			require.NoError(t, err)
			assert.Equal(t, storedUser.ID.String(), claims.UserID)
		})
	}
}

func TestLogout(t *testing.T) {
	s := NewService(testSecret)
	user := addTestUser(t, s, "testuser", "password123")

	token, err := s.GenerateToken(user.ID.String(), user.Version)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req = req.WithContext(
		context.WithValue(req.Context(), UserIDKey, user.ID.String()),
	)
	rec := httptest.NewRecorder()

	s.Logout(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String(), "logout must not write a body")

	var cleared *http.Cookie

	for _, c := range rec.Result().Cookies() {
		if c.Name == CookieName {
			cleared = c

			break
		}
	}

	require.NotNil(t, cleared, "logout must set a cookie with CookieName")
	assert.Empty(t, cleared.Value, "cookie value must be cleared")
	assert.Equal(t, "/", cleared.Path)
	assert.Equal(t, -1, cleared.MaxAge, "cookie must be expired")
	assert.True(t, cleared.HttpOnly)
	assert.True(t, cleared.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cleared.SameSite)

	s.mu.RLock()
	updatedUser, exists := s.usersByID[user.ID.String()]
	s.mu.RUnlock()

	require.True(t, exists, "user must still exist after logout")
	assert.Equal(t, user.Version+1, updatedUser.Version, "logout must bump user version")

	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	protected := s.Middleware(router)

	checkReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	checkReq.AddCookie(&http.Cookie{Name: CookieName, Value: token})

	checkRec := httptest.NewRecorder()
	protected.ServeHTTP(checkRec, checkReq)

	assert.Equal(t, http.StatusUnauthorized, checkRec.Code,
		"old token must be invalidated after logout")
}

func TestLogout_OverridesExistingCookie(t *testing.T) {
	s := NewService(testSecret)
	user := addTestUser(t, s, "testuser", "password123")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "some-old-jwt"})
	req = req.WithContext(
		context.WithValue(req.Context(), UserIDKey, user.ID.String()),
	)

	rec := httptest.NewRecorder()
	s.Logout(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var cleared *http.Cookie

	for _, c := range rec.Result().Cookies() {
		if c.Name == CookieName {
			cleared = c

			break
		}
	}

	require.NotNil(t, cleared)
	assert.Empty(t, cleared.Value)
	assert.Equal(t, -1, cleared.MaxAge)
}

func TestLogout_UnauthorizedWithoutUserInContext(t *testing.T) {
	s := NewService(testSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	s.Logout(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Empty(t, rec.Result().Cookies(),
		"no cookie must be cleared when user is not authenticated")
}

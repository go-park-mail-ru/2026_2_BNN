package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"bnn/internal/models"
	"bnn/internal/pkg/httpjson"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/argon2"
)

const (
	maxRequestBodySize = 4096

	minLoginLength = 3
	maxLoginLength = 20

	minPasswordLength = 8
	maxPasswordLength = 128

	memory      uint32 = 19 * 1024
	iterations  uint32 = 2
	parallelism uint8  = 1
	keyLength   uint32 = 32
	saltLength         = 16
)

type ContextKey string

const UserIDKey ContextKey = "userID"

func GetUserID(r *http.Request) (string, bool) {
	id, ok := r.Context().Value(UserIDKey).(string)
	return id, ok
}

func validateCredentials(login, password string) error {
	loginLength := utf8.RuneCountInString(login)
	if loginLength < minLoginLength || loginLength > maxLoginLength {
		return errors.New("invalid login length")
	}

	passwordLength := utf8.RuneCountInString(password)
	if passwordLength < minPasswordLength || passwordLength > maxPasswordLength {
		return errors.New("invalid password length")
	}

	return nil
}

func (s *Service) SignUp(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing signup request")

	var req models.SignUpRequest
	if err := httpjson.Read(w, r, &req, maxRequestBodySize); err != nil {
		slog.Warn("Failed to read signup request", "error", err)
		w.WriteHeader(httpjson.ReadErrorStatus(err))
		return
	}

	if err := validateCredentials(req.Login, req.Password); err != nil {
		slog.Warn("Invalid signup credentials", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()

	user := models.User{
		ID:           uuid.NewV4(),
		Login:        req.Login,
		PasswordHash: passwordHash,
		Avatar:       "default_avatar.jpg",
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.mu.Lock()
	if _, exists := s.users[req.Login]; exists {
		s.mu.Unlock()
		slog.Warn("Signup rejected: login already exists")
		w.WriteHeader(http.StatusConflict)
		return
	}
	s.users[req.Login] = user
	s.usersByID[user.ID.String()] = user
	s.mu.Unlock()

	token, err := s.GenerateToken(user.ID.String(), user.Version)
	if err != nil {
		slog.Error("Failed to generate token", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	setAuthCookie(w, token)

	httpjson.Write(w, http.StatusCreated, user)
}

func (s *Service) SignIn(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing sign in request")

	var req models.SignInRequest
	if err := httpjson.Read(w, r, &req, maxRequestBodySize); err != nil {
		slog.Warn("Failed to read sign in request", "error", err)
		w.WriteHeader(httpjson.ReadErrorStatus(err))
		return
	}

	if err := validateCredentials(req.Login, req.Password); err != nil {
		slog.Warn("Invalid sign in credentials", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	user, exists := s.users[req.Login]
	s.mu.RUnlock()

	if !exists || !verifyPassword(req.Password, user.PasswordHash) {
		slog.Warn("Invalid login or password")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, err := s.GenerateToken(user.ID.String(), user.Version)
	if err != nil {
		slog.Error("Failed to generate token", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	setAuthCookie(w, token)

	httpjson.Write(w, http.StatusOK, user)
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing logout request")

	userID, ok := GetUserID(r)
	if !ok {
		slog.Warn("User not authenticated")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	s.mu.Lock()

	user, exists := s.usersByID[userID]
	if !exists {
		s.mu.Unlock()
		slog.Warn("User not found", "user_id", userID)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user.Version++
	user.UpdatedAt = time.Now().UTC()

	s.usersByID[userID] = user
	s.users[user.Login] = user

	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			slog.Warn("Auth cookie missing", "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		claims, err := s.ParseToken(cookie.Value)
		if err != nil {
			slog.Warn("Invalid token", "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		s.mu.RLock()
		user, exists := s.usersByID[claims.UserID]
		s.mu.RUnlock()

		if !exists || user.Version != claims.Version {
			slog.Warn("Token version mismatch", "user_id", claims.UserID)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing get current user request")

	userID, ok := GetUserID(r)
	if !ok {
		slog.Warn("User not authenticated")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	u, exists := s.UserByID(userID)
	if !exists {
		slog.Warn("User not found", "user_id", userID)
		w.WriteHeader(http.StatusNotFound)

		return
	}

	httpjson.Write(w, http.StatusOK, u)
}

func hashPassword(password string) ([]byte, error) {

	salt := make([]byte, saltLength)

	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate password salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)

	saltBase64 := base64.RawStdEncoding.EncodeToString(salt)
	hashBase64 := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		iterations,
		parallelism,
		saltBase64,
		hashBase64,
	)

	return []byte(encoded), nil
}

func verifyPassword(password string, encodedHash []byte) bool {
	parts := strings.Split(string(encodedHash), "$")

	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}

	var (
		hashMemory      uint32
		hashIterations  uint32
		hashParallelism uint8
	)

	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &hashMemory, &hashIterations, &hashParallelism)
	if err != nil || hashIterations < 1 || hashParallelism < 1 {
		return false
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(expectedHash) == 0 {
		return false
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		hashIterations,
		hashMemory,
		hashParallelism,
		uint32(len(expectedHash)),
	)

	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1
}

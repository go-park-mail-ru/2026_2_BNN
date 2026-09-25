package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/argon2"
)

const (
	maxRequestBodySize = 4096

	minLoginLength = 3
	maxLoginLength = 20

	minPasswordLength = 8
	maxPasswordLength = 128
)

var users = make(map[string]models.User)
var usersMu sync.RWMutex

type ContextKey string

const UserIDKey ContextKey = "userID"

func GetUserID(r *http.Request) (string, bool) {
	id, ok := r.Context().Value(UserIDKey).(string)
	return id, ok
}

func SignUp(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing signup request")

	var req models.SignUpRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		if sizeErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			slog.Warn("Request body too large", "limit", sizeErr.Limit)
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		slog.Warn("Failed to read request body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(requestBody, &req); err != nil {
		slog.Warn("Failed to decode request JSON", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	loginLength := utf8.RuneCountInString(req.Login)
	passwordLength := utf8.RuneCountInString(req.Password)

	if loginLength < minLoginLength || loginLength > maxLoginLength {
		slog.Warn("Invalid login length")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if passwordLength < minPasswordLength || passwordLength > maxPasswordLength {
		slog.Warn("Invalid password length")
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
		Avatar:       "/static/default_avatar.jpg",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	usersMu.Lock()
	if _, exists := users[req.Login]; exists {
		usersMu.Unlock()

		slog.Warn("Signup rejected: login already exists")
		w.WriteHeader(http.StatusConflict)
		return
	}
	users[req.Login] = user
	usersMu.Unlock()

	token, err := GenerateToken(user.ID.String())
	if err != nil {
		slog.Error("Failed to generate token", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	setAuthCookie(w, token)

	body, err := json.Marshal(user)
	if err != nil {
		slog.Error("Failed to encode signup response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write signup response", "error", err)
	}
}

func SignIn(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing sign in request")

	var req models.SignInRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		if sizeErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			slog.Warn("Request body too large", "limit", sizeErr.Limit)
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		slog.Warn("Failed to read request body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(requestBody, &req); err != nil {
		slog.Warn("Failed to decode request JSON", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	usersMu.RLock()
	user, exists := users[req.Login]
	usersMu.RUnlock()

	if !exists || !verifyPassword(req.Password, user.PasswordHash) {
		slog.Warn("Invalid login or password")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, err := GenerateToken(user.ID.String())
	if err != nil {
		slog.Error("Failed to generate token", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	setAuthCookie(w, token)

	body, err := json.Marshal(user)
	if err != nil {
		slog.Error("Failed to marshal user", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write response", "error", err)
	}
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			slog.Warn("Auth cookie missing", "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		claims, err := ParseToken(cookie.Value)
		if err != nil {
			slog.Warn("Invalid token", "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func hashPassword(password string) ([]byte, error) {
	const (
		memory      uint32 = 19 * 1024
		iterations  uint32 = 2
		parallelism uint8  = 1
		keyLength   uint32 = 32
		saltLength         = 16
	)

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

	expectedVersion := fmt.Sprintf("v=%d", argon2.Version)

	if parts[2] != expectedVersion {
		return false
	}

	if parts[3] != "m=19456,t=2,p=1" {
		return false
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) != 16 {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(expectedHash) != 32 {
		return false
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		2,
		19*1024,
		1,
		32,
	)

	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1
}

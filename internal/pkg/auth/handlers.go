package auth

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"bnn/internal/models"
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

	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		slog.Warn("Failed to decode signup request", "error", err)
		http.Error(w, "некорректный JSON или слишком большой запрос", http.StatusBadRequest)
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		slog.Warn("Unexpected data after signup JSON object")
		http.Error(w, "ожидается один JSON-объект", http.StatusBadRequest)
		return
	}

	loginLength := utf8.RuneCountInString(req.Login)
	passwordLength := utf8.RuneCountInString(req.Password)

	if loginLength < 3 || loginLength > 32 {
		slog.Warn("Invalid login length")
		http.Error(w, "логин должен содержать от 3 до 32 символов", http.StatusBadRequest)
		return
	}

	if passwordLength < 8 || passwordLength > 128 {
		slog.Warn("Invalid password length")
		http.Error(w, "пароль должен содержать от 8 до 128 символов", http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		http.Error(w, "ошибка сервера", http.StatusInternalServerError)
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

	body, err := json.Marshal(user)
	if err != nil {
		slog.Error("Failed to encode signup response", "error", err)
		http.Error(w, "ошибка серва", http.StatusInternalServerError)
		return
	}

	usersMu.Lock()
	if _, exists := users[req.Login]; exists {
		usersMu.Unlock()

		slog.Warn("Signup rejected: login already exists")
		http.Error(w, "логин уже занят", http.StatusConflict)
		return
	}
	users[req.Login] = user
	usersMu.Unlock()
	
	token, err := GenerateToken(user.ID.String())
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	setAuthCookie(w, token)

	safeUser := user
	safeUser.PasswordHash = nil

	body, err := json.Marshal(safeUser)
	if err != nil {
		slog.Error("failed to marshal user", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write signup response", "error", err)
	}
}

func SignIn(w http.ResponseWriter, r *http.Request) {
	slog.Info("user sign in")

	var req models.SignInRequest

	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		slog.Warn("failed to decode sign in request", "error", err)
		http.Error(w, "invalid JSON or request too large", http.StatusBadRequest)
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		slog.Warn("extra data in sign in request")
		http.Error(w, "expected a single JSON object", http.StatusBadRequest)
		return
	}

	usersMu.RLock()
	user, exists := users[req.Login]
	usersMu.RUnlock()

	if !exists || !verifyPassword(req.Password, user.PasswordHash) {
		slog.Warn("invalid login or password")
		http.Error(w, "invalid login or password", http.StatusUnauthorized)
		return
	}

	token, err := GenerateToken(user.ID.String())
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	setAuthCookie(w, token)

	safeUser := user
	safeUser.PasswordHash = nil

	body, err := json.Marshal(safeUser)
	if err != nil {
		slog.Error("failed to marshal user", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(body); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			slog.Warn("auth cookie missing", "error", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := ParseToken(cookie.Value)
		if err != nil {
			slog.Warn("invalid token", "error", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func hashPassword(password string) ([]byte, error) {
	salt := rand.Text()

	hash, err := pbkdf2.Key(
		sha256.New,
		password,
		[]byte(salt),
		600_000,
		32,
	)
	if err != nil {
		return nil, err
	}

	encoded := fmt.Sprintf("pbkdf2-sha256$600000$%s$%x", salt, hash)

	return []byte(encoded), nil
}

func verifyPassword(password string, encodedHash []byte) bool {
	parts := strings.Split(string(encodedHash), "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}

	salt := []byte(parts[2])

	expected, err := hex.DecodeString(parts[3])
	if err != nil {
		return false
	}

	actual, err := pbkdf2.Key(
		sha256.New,
		password,
		salt,
		iterations,
		len(expected),
	)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(expected, actual) == 1
}
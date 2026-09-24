package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"

	"encoding/base64"

	"golang.org/x/crypto/argon2"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

var users = make(map[string]models.User)
var usersMu sync.RWMutex

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

	now := time.Now()

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write signup response", "error", err)
	}
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

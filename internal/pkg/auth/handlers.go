package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

var users = make(map[string]models.User)
var usersMu sync.RWMutex

func SignUp(w http.ResponseWriter, r *http.Request) {
	slog.Info("регестрация пользователя")

	var req models.SignUpRequest

	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		slog.Warn("не удалось прочитать запрос регистрации", "error", err)
		http.Error(w, "некорректный JSON или слишком большой запрос", http.StatusBadRequest)
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		slog.Warn("лишние данные в запросе регистрации")
		http.Error(w, "ожидается один JSON-объект", http.StatusBadRequest)
		return
	}

	loginLength := utf8.RuneCountInString(req.Login)
	passwordLength := utf8.RuneCountInString(req.Password)

	if loginLength < 3 || loginLength > 32 {
		slog.Warn("некорректная длина логина")
		http.Error(w, "логин должен содержать от 3 до 32 символов", http.StatusBadRequest)
		return
	}

	if passwordLength < 8 || passwordLength > 128 {
		slog.Warn("некорректная длина пароля")
		http.Error(w, "пароль должен содержать от 8 до 128 символов", http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		slog.Error("ошибка хеширования пароля", "error", err)
		http.Error(w, "ошибка сервера", http.StatusInternalServerError)
		return
	}

	now := time.Now()

	user := models.User{
		ID:           uuid.NewV4().String(),
		Login:        req.Login,
		PasswordHash: passwordHash,
		Avatar:       "/static/default_avatar",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	body, err := json.Marshal(user)
	if err != nil {
		slog.Error("ошибка формирования JSON", "error", err)
		http.Error(w, "ошибка серва", http.StatusInternalServerError)
		return
	}

	usersMu.Lock()

	if _, exists := users[req.Login]; exists {
		usersMu.Unlock()

		slog.Warn("попытка регистрации с занятым логином")
		http.Error(w, "логин уже занят", http.StatusConflict)
		return
	}

	users[req.Login] = user

	usersMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(body); err != nil {
		slog.Error("ошибка отправки ответа", "error", err)
	}
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

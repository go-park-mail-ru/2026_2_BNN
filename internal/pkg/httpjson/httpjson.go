package httpjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"bnn/internal/models"
)

var ErrBodyTooLarge = errors.New("request body too large")

func Write(w http.ResponseWriter, status int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write response", "error", err)
	}
}

func Read(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			return fmt.Errorf("%w: limit %d bytes", ErrBodyTooLarge, sizeErr.Limit)
		}

		return fmt.Errorf("read body: %w", err)
	}

	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	return nil
}

func ReadErrorStatus(err error) int {
	if errors.Is(err, ErrBodyTooLarge) {
		return http.StatusRequestEntityTooLarge
	}

	return http.StatusBadRequest
}

func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, models.ErrValidation):
		status = http.StatusBadRequest
	case errors.Is(err, models.ErrInvalidCredentials), errors.Is(err, models.ErrInvalidToken):
		status = http.StatusUnauthorized
	case errors.Is(err, models.ErrUserNotFound), errors.Is(err, models.ErrNoteNotFound):
		status = http.StatusNotFound
	case errors.Is(err, models.ErrUserExists):
		status = http.StatusConflict
	}

	if status == http.StatusInternalServerError {
		slog.Error("Request failed", "error", err)
	} else {
		slog.Warn("Request failed", "error", err)
	}

	w.WriteHeader(status)
}

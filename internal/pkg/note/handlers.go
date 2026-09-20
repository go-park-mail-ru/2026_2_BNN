package note

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"bnn/internal/models"
)

var notes = []models.Note{
	{
		ID:        "1",
		Title:     "Изучаю Go",
		CreatedBy: "1",
		BlocksID:  []string{},
	},
	{
		ID:        "2",
		Title:     "Изучаю HTTP в Go",
		CreatedBy: "1",
		BlocksID:  []string{},
	},
}

var notesMu sync.RWMutex

func ListNotes(w http.ResponseWriter, r *http.Request) {
	slog.Info("получение спсика заметок")

	limit := 10
	offset := 0

	query := r.URL.Query()

	if query.Has("limit") {
		value, err := strconv.Atoi(query.Get("limit"))

		if err != nil || value < 1 || value > 100 {
			slog.Warn("Некорректный limit")

			http.Error(
				w,
				"limit должен быть целым число от 1 до 100",
				http.StatusBadRequest,
			)
			return
		}
		limit = value
	}

	if query.Has("offset") {
		value, err := strconv.Atoi(query.Get("offset"))

		if err != nil || value < 0 {
			slog.Warn("Некорректный offset")

			http.Error(
				w,
				"offset должен быть целым неотрицательным числом",
				http.StatusBadRequest,
			)
			return
		}
		offset = value
	}

	notesMu.RLock()

	start := min(offset, len(notes))
	count := min(limit, len(notes)-start)
	end := start + count

	result := make([]models.Note, count)
	copy(result, notes[start:end])

	body, err := json.Marshal(result)

	notesMu.RUnlock()

	if err != nil {
		slog.Error("ошибка форматирование JSON", "error", err)

		http.Error(
			w,
			"ошибка сервера",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(body); err != nil {
		slog.Error("ошибка отправки ответа", "error", err)
	}
}

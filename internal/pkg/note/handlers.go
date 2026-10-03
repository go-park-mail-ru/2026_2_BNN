package note

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

var testUserID = uuid.NewV4()
var testCreatedAt = time.Now().UTC()

var notes = []models.Note{
	{
		ID:        uuid.NewV4(),
		Title:     "Go",
		CreatedBy: testUserID,
		Blocks: []models.Block{
			{
				ID:        uuid.NewV4(),
				Content:   "Go — язык программирования со статической типизацией.",
				CreatedAt: testCreatedAt,
				UpdatedAt: testCreatedAt,
			},
			{
				ID:        uuid.NewV4(),
				Content:   "Структуры позволяют объединять связанные данные.",
				CreatedAt: testCreatedAt,
				UpdatedAt: testCreatedAt,
			},
		},
		CreatedAt: testCreatedAt,
		UpdatedAt: testCreatedAt,
	},
	{
		ID:        uuid.NewV4(),
		Title:     "HTTP in Go",
		CreatedBy: testUserID,
		Blocks: []models.Block{
			{
				ID:        uuid.NewV4(),
				Content:   "HTTP-обработчик принимает запрос и формирует ответ.",
				CreatedAt: testCreatedAt,
				UpdatedAt: testCreatedAt,
			},
		},
		CreatedAt: testCreatedAt,
		UpdatedAt: testCreatedAt,
	},
}

var notesMu sync.RWMutex

func parsePagination(query url.Values) (int, int, error) {
	const (
		defaultLimit  = 10
		defaultOffset = 0
		minLimit      = 1
		maxLimit      = 100
	)

	limit := defaultLimit
	offset := defaultOffset

	if query.Has("limit") {
		value, err := strconv.Atoi(query.Get("limit"))
		if err != nil || value < minLimit || value > maxLimit {
			return 0, 0, errors.New("invalid limit")
		}

		limit = value
	}

	if query.Has("offset") {
		value, err := strconv.Atoi(query.Get("offset"))
		if err != nil || value < 0 {
			return 0, 0, errors.New("invalid offset")
		}

		offset = value
	}

	return limit, offset, nil
}

func ListNotes(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing notes list request")

	limit, offset, err := parsePagination(r.URL.Query())
	if err != nil {
		slog.Warn("Invalid pagination parameters", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
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
		slog.Error("Failed to encode notes response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write notes response", "error", err)
	}
}

func GetNote(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing get note request")

	_, ok := auth.GetUserID(r)
	if !ok {
		slog.Warn("User not authenticated")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	rawID := vars["id"]

	if rawID == "" {
		slog.Warn("Empty note id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	noteID, err := uuid.FromString(rawID)
	if err != nil {
		slog.Warn("Invalid note id", "id", rawID, "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	notesMu.RLock()
	var found *models.Note
	for i := range notes {
		if notes[i].ID == noteID {
			found = &notes[i]
			break
		}
	}
	notesMu.RUnlock()

	if found == nil {
		slog.Warn("Note not found", "id", rawID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	body, err := json.Marshal(found)
	if err != nil {
		slog.Error("Failed to marshal note", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write response", "error", err)
	}
}

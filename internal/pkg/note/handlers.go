package note

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

var testUserID = uuid.NewV4()

var notes = []models.Note{
	{
		ID:        uuid.NewV4(),
		Title:     "Изучаю Go",
		CreatedBy: testUserID,
		BlocksID:  []uuid.UUID{},
	},
	{
		ID:        uuid.NewV4(),
		Title:     "Изучаю HTTP в Go",
		CreatedBy: testUserID,
		BlocksID:  []uuid.UUID{},
	},
}

var notesMu sync.RWMutex

func ListNotes(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing notes list request")

	limit := 10
	offset := 0

	query := r.URL.Query()

	if query.Has("limit") {
		value, err := strconv.Atoi(query.Get("limit"))
		if err != nil || value < 1 || value > 100 {
			slog.Warn("Invalid limit")

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
			slog.Warn("Invalid offset")

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
		slog.Error("Failed to encode notes response", "error", err)

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
		slog.Error("Failed to write notes response", "error", err)
	}
}

func GetNote(w http.ResponseWriter, r *http.Request) {
	slog.Info("getting note")

	userID, ok := auth.GetUserID(r)
	if !ok {
		slog.Warn("user not authenticated")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	rawID := vars["id"]

	if rawID == "" {
		slog.Warn("empty note id")
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	noteID, err := uuid.Parse(rawID)
	if err != nil {
		slog.Warn("invalid note id", "id", rawID, "error", err)
		http.Error(w, "invalid note id", http.StatusBadRequest)
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
		slog.Warn("note not found", "id", rawID)
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}

	if found.CreatedBy.String() != userID {
		slog.Warn("access denied", "note_id", rawID, "user_id", userID)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	body, err := json.Marshal(found)
	if err != nil {
		slog.Error("failed to marshal note", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(body); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}
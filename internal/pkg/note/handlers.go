package note

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

var testUserID = uuid.NewV4()

var notes = []models.Note{
	{
		ID:        uuid.NewV4(),
		Title:     "Learning Go",
		CreatedBy: testUserID,
		BlocksID:  []uuid.UUID{},
	},
	{
		ID:        uuid.NewV4(),
		Title:     "Learning HTTP in Go",
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
			http.Error(w, "limit must be an integer between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = value
	}

	if query.Has("offset") {
		value, err := strconv.Atoi(query.Get("offset"))
		if err != nil || value < 0 {
			slog.Warn("Invalid offset")
			http.Error(w, "offset must be a non-negative integer", http.StatusBadRequest)
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
		http.Error(w, "internal server error", http.StatusInternalServerError)
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

	userID, ok := auth.GetUserID(r)
	if !ok {
		slog.Warn("User not authenticated")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	rawID := vars["id"]

	if rawID == "" {
		slog.Warn("Empty note id")
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	noteID, err := uuid.FromString(rawID)
	if err != nil {
		slog.Warn("Invalid note id", "id", rawID, "error", err)
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
		slog.Warn("Note not found", "id", rawID)
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}

	if found.CreatedBy.String() != userID {
		slog.Warn("Access denied", "note_id", rawID, "user_id", userID)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	body, err := json.Marshal(found)
	if err != nil {
		slog.Error("Failed to marshal note", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write response", "error", err)
	}
}

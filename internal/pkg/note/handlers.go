package note

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var notes = []models.Note{
	{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Title:     "Learning Go",
		CreatedBy: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		BlocksID:  []uuid.UUID{},
	},
	{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Title:     "Learning HTTP in Go",
		CreatedBy: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		BlocksID:  []uuid.UUID{},
	},
}

var notesMu sync.RWMutex

func ListNotes(w http.ResponseWriter, r *http.Request) {
	slog.Info("listing notes")

	limit := 10
	offset := 0

	query := r.URL.Query()

	if query.Has("limit") {
		value, err := strconv.Atoi(query.Get("limit"))
		if err != nil || value < 1 || value > 100 {
			slog.Warn("invalid limit")
			http.Error(w, "limit must be an integer between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = value
	}

	if query.Has("offset") {
		value, err := strconv.Atoi(query.Get("offset"))
		if err != nil || value < 0 {
			slog.Warn("invalid offset")
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
		slog.Error("failed to marshal notes", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(body); err != nil {
		slog.Error("failed to write response", "error", err)
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
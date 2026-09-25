package note

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetNotes(list []models.Note) {
	notesMu.Lock()
	defer notesMu.Unlock()
	notes = list
}

func TestGetNote(t *testing.T) {
	ownerID := uuid.NewV4()
	otherID := uuid.NewV4()
	noteID := uuid.NewV4()

	baseNote := models.Note{
		ID:        noteID,
		Title:     "Test note",
		CreatedBy: ownerID,
		BlocksID:  []uuid.UUID{}, // ← было []string{}
	}

	tests := []struct {
		name          string
		pathID        string
		authUserID    *uuid.UUID
		prepare       func()
		expectedCode  int
		expectedTitle string
	}{
		{
			name:          "OK: owner gets own note",
			pathID:        noteID.String(),
			authUserID:    &ownerID,
			prepare:       func() { resetNotes([]models.Note{baseNote}) },
			expectedCode:  http.StatusOK,
			expectedTitle: "Test note",
		},
		{
			name:         "Note not found",
			pathID:       uuid.NewV4().String(),
			authUserID:   &ownerID,
			prepare:      func() { resetNotes([]models.Note{baseNote}) },
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Forbidden: note belongs to another user",
			pathID:       noteID.String(),
			authUserID:   &otherID,
			prepare:      func() { resetNotes([]models.Note{baseNote}) },
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "Bad request: invalid UUID",
			pathID:       "not-a-uuid",
			authUserID:   &ownerID,
			prepare:      func() { resetNotes([]models.Note{baseNote}) },
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Unauthorized: no user in context",
			pathID:       noteID.String(),
			authUserID:   nil,
			prepare:      func() { resetNotes([]models.Note{baseNote}) },
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()

			req := httptest.NewRequest(http.MethodGet, "/api/notes/"+tt.pathID, nil)
			if tt.authUserID != nil {
				req = req.WithContext(
					context.WithValue(req.Context(), auth.UserIDKey, tt.authUserID.String()),
				)
			}

			router := mux.NewRouter()
			router.HandleFunc("/api/notes/{id}", GetNote)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectedCode == http.StatusOK {
				var got models.Note
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, noteID, got.ID)
				assert.Equal(t, tt.expectedTitle, got.Title)
				assert.Equal(t, ownerID, got.CreatedBy)
			}
		})
	}
}

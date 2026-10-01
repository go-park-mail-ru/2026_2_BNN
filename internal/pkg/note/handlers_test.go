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
		BlocksID:  []uuid.UUID{},
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

func TestListNotes(t *testing.T) {
	ownerID := uuid.NewV4()

	fixtures := make([]models.Note, 12)
	ids := make([]uuid.UUID, 12)

	for i := range fixtures {
		ids[i] = uuid.NewV4()

		fixtures[i] = models.Note{
			ID:        ids[i],
			Title:     "Test note",
			CreatedBy: ownerID,
			BlocksID:  []uuid.UUID{},
		}
	}

	tests := []struct {
		name         string
		query        string
		emptyStorage bool
		expectedCode int
		expectedIDs  []uuid.UUID
	}{
		{
			name:         "default pagination",
			expectedCode: http.StatusOK,
			expectedIDs:  ids[:10],
		},
		{
			name:         "limit and offset",
			query:        "?limit=5&offset=3",
			expectedCode: http.StatusOK,
			expectedIDs:  ids[3:8],
		},
		{
			name:         "minimum limit",
			query:        "?limit=1&offset=0",
			expectedCode: http.StatusOK,
			expectedIDs:  ids[:1],
		},
		{
			name:         "maximum limit",
			query:        "?limit=100",
			expectedCode: http.StatusOK,
			expectedIDs:  ids,
		},
		{
			name:         "last page",
			query:        "?limit=5&offset=10",
			expectedCode: http.StatusOK,
			expectedIDs:  ids[10:],
		},
		{
			name:         "offset at end",
			query:        "?offset=12",
			expectedCode: http.StatusOK,
			expectedIDs:  []uuid.UUID{},
		},
		{
			name:         "offset beyond end",
			query:        "?offset=100",
			expectedCode: http.StatusOK,
			expectedIDs:  []uuid.UUID{},
		},
		{
			name:         "empty storage",
			emptyStorage: true,
			expectedCode: http.StatusOK,
			expectedIDs:  []uuid.UUID{},
		},
		{
			name:         "zero limit",
			query:        "?limit=0",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "limit above maximum",
			query:        "?limit=101",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid limit",
			query:        "?limit=abc",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "empty limit",
			query:        "?limit=",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "negative offset",
			query:        "?offset=-1",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid offset",
			query:        "?offset=abc",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "empty offset",
			query:        "?offset=",
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notesMu.Lock()
			previousNotes := notes

			if tt.emptyStorage {
				notes = []models.Note{}
			} else {
				notes = append([]models.Note(nil), fixtures...)
			}

			notesMu.Unlock()

			t.Cleanup(func() {
				notesMu.Lock()
				notes = previousNotes
				notesMu.Unlock()
			})

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/notes/getall"+tt.query,
				nil,
			)

			ctx := context.WithValue(
				req.Context(),
				auth.UserIDKey,
				ownerID.String(),
			)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			ListNotes(rec, req)

			require.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectedCode != http.StatusOK {
				assert.Empty(t, rec.Body.String())
				return
			}

			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var result []models.Note
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

			// Пустой результат должен быть [], а не null.
			require.NotNil(t, result)

			resultIDs := make([]uuid.UUID, 0, len(result))
			for _, item := range result {
				resultIDs = append(resultIDs, item.ID)
			}

			assert.Equal(t, tt.expectedIDs, resultIDs)
		})
	}
}

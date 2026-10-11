package note

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/models"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authdelivery "bnn/internal/pkg/auth/delivery/http"
)

func TestGetNote(t *testing.T) {
	ownerID := uuid.NewV4()
	noteID := uuid.NewV4()

	baseNote := models.Note{
		ID:        noteID,
		Title:     "Test note",
		CreatedBy: ownerID,
		Blocks: []models.Block{
			{
				ID:      uuid.NewV4(),
				Content: "Текст тестовой заметки",
			},
		},
	}

	tests := []struct {
		name          string
		pathID        string
		authUserID    *uuid.UUID
		expectedCode  int
		expectedTitle string
	}{
		{
			name:          "OK: owner gets own note",
			pathID:        noteID.String(),
			authUserID:    &ownerID,
			expectedCode:  http.StatusOK,
			expectedTitle: "Test note",
		},
		{
			name:         "Note not found",
			pathID:       uuid.NewV4().String(),
			authUserID:   &ownerID,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Bad request: invalid UUID",
			pathID:       "not-a-uuid",
			authUserID:   &ownerID,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Unauthorized: no user in context",
			pathID:       noteID.String(),
			authUserID:   nil,
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler([]models.Note{baseNote})

			req := httptest.NewRequest(http.MethodGet, "/api/notes/"+tt.pathID, nil)
			if tt.authUserID != nil {
				req = req.WithContext(
					authdelivery.SetUserIDForTest(req.Context(), *tt.authUserID),
				)
			}

			router := mux.NewRouter()
			router.HandleFunc("/api/notes/{id}", handler.GetNote)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectedCode == http.StatusOK {
				var got models.Note
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, noteID, got.ID)
				assert.Equal(t, tt.expectedTitle, got.Title)
				assert.Equal(t, ownerID, got.CreatedBy)

				require.Len(t, got.Blocks, 1)
				assert.Equal(t, baseNote.Blocks[0].ID, got.Blocks[0].ID)
				assert.Equal(t, "Текст тестовой заметки", got.Blocks[0].Content)
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
			Blocks:    []models.Block{},
		}
	}

	tests := []struct {
		name         string
		query        string
		emptyStorage bool
		expectedCode int
		expectedIDs  []uuid.UUID
	}{
		// ... те же кейсы
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := fixtures
			if tt.emptyStorage {
				stored = []models.Note{}
			}

			handler := NewHandler(append([]models.Note(nil), stored...))

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/notes/getall"+tt.query,
				nil,
			)

			rec := httptest.NewRecorder()

			handler.ListNotes(rec, req)

			require.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectedCode != http.StatusOK {
				assert.Empty(t, rec.Body.String())
				return
			}

			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var result []models.Note
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

			require.NotNil(t, result)

			resultIDs := make([]uuid.UUID, 0, len(result))
			for _, item := range result {
				resultIDs = append(resultIDs, item.ID)
			}

			assert.Equal(t, tt.expectedIDs, resultIDs)
		})
	}
}

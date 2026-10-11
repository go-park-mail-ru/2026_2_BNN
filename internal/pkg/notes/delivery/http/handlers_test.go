package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/notes"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubUsecase — простая ручная заглушка для notes.Usecase.
// Реализует только те методы, которые дергают тесты.
type stubUsecase struct {
	getNotes func(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error)
	getNote  func(ctx context.Context, userID uuid.UUID, noteID uuid.UUID) (models.Note, error)
}

func (s *stubUsecase) GetNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error) {
	return s.getNotes(ctx, userID, limit, offset)
}

func (s *stubUsecase) GetNote(ctx context.Context, userID uuid.UUID, noteID uuid.UUID) (models.Note, error) {
	return s.getNote(ctx, userID, noteID)
}

// Заглушки остальных методов Usecase — чтобы интерфейс удовлетворялся.
func (s *stubUsecase) CreateNote(ctx context.Context, userID uuid.UUID, note models.Note) (models.Note, error) {
	return models.Note{}, nil
}
func (s *stubUsecase) UpdateNote(ctx context.Context, userID uuid.UUID, note models.Note) (models.Note, error) {
	return models.Note{}, nil
}
func (s *stubUsecase) DeleteNote(ctx context.Context, userID uuid.UUID, noteID uuid.UUID) error {
	return nil
}

// Проверка, что stubUsecase удовлетворяет интерфейсу.
var _ notes.Usecase = (*stubUsecase)(nil)

func TestGetNote(t *testing.T) {
	ownerID := uuid.NewV4()
	otherUserID := uuid.NewV4()
	noteID := uuid.NewV4()

	baseNote := models.Note{
		ID:        noteID,
		Title:     "Test note",
		CreatedBy: ownerID,
		Blocks: []models.Block{
			{
				ID:      uuid.NewV4(),
				Content: json.RawMessage(`"Текст тестовой заметки"`),
			},
		},
	}
	tests := []struct {
		name         string
		pathID       string
		authUserID   *uuid.UUID
		uc           *stubUsecase
		expectedCode int
	}{
		{
			name:       "OK: owner gets own note",
			pathID:     noteID.String(),
			authUserID: &ownerID,
			uc: &stubUsecase{
				getNote: func(ctx context.Context, userID uuid.UUID, id uuid.UUID) (models.Note, error) {
					return baseNote, nil
				},
			},
			expectedCode: http.StatusOK,
		},
		{
			name:       "Note not found",
			pathID:     uuid.NewV4().String(),
			authUserID: &ownerID,
			uc: &stubUsecase{
				getNote: func(ctx context.Context, userID uuid.UUID, id uuid.UUID) (models.Note, error) {
					return models.Note{}, models.ErrNoteNotFound
				},
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Bad request: invalid UUID",
			pathID:       "not-a-uuid",
			authUserID:   &ownerID,
			uc:           &stubUsecase{},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Unauthorized: no user in context",
			pathID:       noteID.String(),
			authUserID:   nil,
			uc:           &stubUsecase{},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:       "Forbidden: other user's note",
			pathID:     noteID.String(),
			authUserID: &otherUserID,
			uc: &stubUsecase{
				getNote: func(ctx context.Context, userID uuid.UUID, id uuid.UUID) (models.Note, error) {
					return models.Note{}, models.ErrNoteNotFound
				},
			},
			// Если usecase возвращает ErrNoteNotFound — handler вернёт 404.
			// Если ты хочешь именно 403 — usecase должен возвращать
			// отдельную ErrForbidden, которой в models нет.
			// Пока оставим 404 — так делает текущий usecase.
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.uc)

			req := httptest.NewRequest(http.MethodGet, "/api/notes/"+tt.pathID, nil)
			if tt.authUserID != nil {
				req = req.WithContext(
					ctxutil.WithUserID(req.Context(), *tt.authUserID),
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
				assert.Equal(t, "Test note", got.Title)
				assert.Equal(t, ownerID, got.CreatedBy)

				require.Len(t, got.Blocks, 1)
				assert.Equal(t, baseNote.Blocks[0].ID, got.Blocks[0].ID)
				assert.JSONEq(t, `"Текст тестовой заметки"`, string(got.Blocks[0].Content))
			}
		})
	}
}

func TestListNotes(t *testing.T) {
	userID := uuid.NewV4()

	fixtures := make([]models.Note, 12)
	ids := make([]uuid.UUID, 12)

	for i := range fixtures {
		ids[i] = uuid.NewV4()
		fixtures[i] = models.Note{
			ID:        ids[i],
			Title:     "Test note",
			CreatedBy: userID,
			Blocks:    []models.Block{},
		}
	}

	tests := []struct {
		name         string
		query        string
		empty        bool
		expectedCode int
		expectedIDs  []uuid.UUID
	}{
		{name: "default pagination", expectedCode: http.StatusOK, expectedIDs: ids[:10]},
		{name: "limit and offset", query: "?limit=5&offset=3", expectedCode: http.StatusOK, expectedIDs: ids[3:8]},
		{name: "minimum limit", query: "?limit=1&offset=0", expectedCode: http.StatusOK, expectedIDs: ids[:1]},
		{name: "maximum limit", query: "?limit=100", expectedCode: http.StatusOK, expectedIDs: ids},
		{name: "last page", query: "?limit=5&offset=10", expectedCode: http.StatusOK, expectedIDs: ids[10:]},
		{name: "offset at end", query: "?offset=12", expectedCode: http.StatusOK, expectedIDs: []uuid.UUID{}},
		{name: "offset beyond end", query: "?offset=100", expectedCode: http.StatusOK, expectedIDs: []uuid.UUID{}},
		{name: "empty storage", empty: true, expectedCode: http.StatusOK, expectedIDs: []uuid.UUID{}},
		{name: "zero limit", query: "?limit=0", expectedCode: http.StatusBadRequest},
		{name: "limit above maximum", query: "?limit=101", expectedCode: http.StatusBadRequest},
		{name: "invalid limit", query: "?limit=abc", expectedCode: http.StatusBadRequest},
		{name: "empty limit", query: "?limit=", expectedCode: http.StatusBadRequest},
		{name: "negative offset", query: "?offset=-1", expectedCode: http.StatusBadRequest},
		{name: "invalid offset", query: "?offset=abc", expectedCode: http.StatusBadRequest},
		{name: "empty offset", query: "?offset=", expectedCode: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := fixtures
			if tt.empty {
				stored = []models.Note{}
			}

			uc := &stubUsecase{
				getNotes: func(ctx context.Context, uid uuid.UUID, limit, offset int64) ([]models.Note, error) {
					// Эмулируем пагинацию по stored.
					start := int(offset)
					if start > len(stored) {
						start = len(stored)
					}
					end := start + int(limit)
					if end > len(stored) {
						end = len(stored)
					}
					result := make([]models.Note, end-start)
					copy(result, stored[start:end])
					return result, nil
				},
			}

			handler := NewHandler(uc)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/notes/getall"+tt.query,
				nil,
			)
			req = req.WithContext(
				ctxutil.WithUserID(req.Context(), userID),
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

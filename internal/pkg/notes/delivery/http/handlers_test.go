package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/notes"
	"bnn/internal/pkg/notes/delivery/http/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubUsecase struct {
	getNotes func(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error)
	getNote  func(ctx context.Context, userID uuid.UUID, noteID uuid.UUID) (models.Note, error)
}

func (s *stubUsecase) GetNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error) {
	if s.getNotes != nil {
		return s.getNotes(ctx, userID, limit, offset)
	}
	return nil, nil
}

func (s *stubUsecase) GetNote(ctx context.Context, userID uuid.UUID, noteID uuid.UUID) (models.Note, error) {
	if s.getNote != nil {
		return s.getNote(ctx, userID, noteID)
	}
	return models.Note{}, nil
}

func (s *stubUsecase) CreateNote(ctx context.Context, userID uuid.UUID, req notes.CreateNoteRequest) (models.Note, error) {
	return models.Note{}, nil
}
func (s *stubUsecase) UpdateNote(ctx context.Context, userID, noteID uuid.UUID, req notes.UpdateNoteRequest) (models.Note, error) {
	return models.Note{}, nil
}
func (s *stubUsecase) DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error {
	return nil
}

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
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.uc)

			ctx := context.Background()
			if tt.authUserID != nil {
				ctx = ctxutil.WithUserID(ctx, *tt.authUserID)
			}

			reqObj := api.GetNoteRequestObject{
				Id: openapi_types.UUID(uuid.FromStringOrNil(tt.pathID)),
			}

			resp, err := handler.GetNote(ctx, reqObj)
			require.NoError(t, err)

			if tt.expectedCode == http.StatusOK {
				okResp, ok := resp.(api.GetNote200JSONResponse)
				require.True(t, ok)
				assert.Equal(t, openapi_types.UUID(noteID), okResp.Id)
				assert.Equal(t, "Test note", okResp.Title)
				assert.Equal(t, openapi_types.UUID(ownerID), okResp.CreatedBy)
			} else if tt.expectedCode == http.StatusNotFound {
				_, ok := resp.(api.GetNote404Response)
				assert.True(t, ok)
			} else if tt.expectedCode == http.StatusUnauthorized {
				_, ok := resp.(api.GetNote401Response)
				assert.True(t, ok)
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
		limit        *int
		offset       *int
		empty        bool
		expectedCode int
		expectedIDs  []uuid.UUID
	}{
		{name: "default pagination", expectedCode: http.StatusOK, expectedIDs: ids[:10]},
		{name: "limit and offset", limit: intPtr(5), offset: intPtr(3), expectedCode: http.StatusOK, expectedIDs: ids[3:8]},
		{name: "minimum limit", limit: intPtr(1), offset: intPtr(0), expectedCode: http.StatusOK, expectedIDs: ids[:1]},
		{name: "offset at end", offset: intPtr(12), expectedCode: http.StatusOK, expectedIDs: []uuid.UUID{}},
		{name: "empty storage", empty: true, expectedCode: http.StatusOK, expectedIDs: []uuid.UUID{}},
		{name: "zero limit", limit: intPtr(0), expectedCode: http.StatusBadRequest},
		{name: "limit above maximum", limit: intPtr(101), expectedCode: http.StatusBadRequest},
		{name: "negative offset", offset: intPtr(-1), expectedCode: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := fixtures
			if tt.empty {
				stored = []models.Note{}
			}

			uc := &stubUsecase{
				getNotes: func(ctx context.Context, uid uuid.UUID, limit, offset int64) ([]models.Note, error) {
					if offset < 0 {
						offset = 0
					}

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
			ctx := ctxutil.WithUserID(context.Background(), userID)

			reqObj := api.ListNotesRequestObject{
				Params: api.ListNotesParams{
					Limit:  tt.limit,
					Offset: tt.offset,
				},
			}

			resp, err := handler.ListNotes(ctx, reqObj)
			require.NoError(t, err)

			if tt.expectedCode != http.StatusOK {
				_, ok := resp.(api.ListNotes400Response)
				assert.True(t, ok, "expected 400 response")
				return
			}

			okResp, ok := resp.(api.ListNotes200JSONResponse)
			require.True(t, ok)

			resultIDs := make([]uuid.UUID, 0, len(okResp))
			for _, item := range okResp {
				resultIDs = append(resultIDs, uuid.UUID(item.Id))
			}
			assert.Equal(t, tt.expectedIDs, resultIDs)
		})
	}
}

func intPtr(i int) *int {
	return &i
}

package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/notes"
	"bnn/internal/pkg/notes/delivery/http/api"
	"bnn/internal/pkg/notes/mocks"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListNotes(t *testing.T) {
	userID := uuid.NewV4()
	note := models.Note{ID: uuid.NewV4(), CreatedBy: userID, Title: "Заметка"}

	tests := []struct {
		name     string
		limit    *int
		offset   *int
		prepare  func(uc *mocks.MockUsecase)
		expected api.ListNotesResponseObject
	}{
		{
			name: "default pagination",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNotes(gomock.Any(), userID, int64(10), int64(0)).Return([]models.Note{note}, nil)
			},
			expected: api.ListNotes200JSONResponse{toAPINote(note)},
		},
		{
			name:   "limit and offset",
			limit:  intPtr(5),
			offset: intPtr(3),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNotes(gomock.Any(), userID, int64(5), int64(3)).Return([]models.Note{note}, nil)
			},
			expected: api.ListNotes200JSONResponse{toAPINote(note)},
		},
		{
			name:  "maximum limit",
			limit: intPtr(100),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNotes(gomock.Any(), userID, int64(100), int64(0)).Return([]models.Note{}, nil)
			},
			expected: api.ListNotes200JSONResponse{},
		},
		{
			name:     "zero limit",
			limit:    intPtr(0),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.ListNotes400Response{},
		},
		{
			name:     "limit above maximum",
			limit:    intPtr(101),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.ListNotes400Response{},
		},
		{
			name:     "negative offset",
			offset:   intPtr(-1),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.ListNotes400Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			req := api.ListNotesRequestObject{Params: api.ListNotesParams{Limit: tt.limit, Offset: tt.offset}}
			resp, err := NewHandler(uc).ListNotes(ctxutil.WithUserID(context.Background(), userID), req)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestGetNote(t *testing.T) {
	userID := uuid.NewV4()
	noteID := uuid.NewV4()
	parentID := uuid.NewV4()
	blockID := uuid.NewV4()
	tagID := uuid.NewV4()
	now := time.Now().UTC()

	note := models.Note{
		ID:           noteID,
		ParentNoteID: &parentID,
		CreatedBy:    userID,
		Title:        "Заметка",
		Header:       "Шапка",
		Icon:         "📒",
		Blocks:       []models.Block{{ID: blockID, Content: json.RawMessage(`{"text":"Привет"}`), CreatedAt: now, UpdatedAt: now}},
		Tags:         []models.Tag{{ID: tagID, Name: "учёба", Color: "#FF880080", CreatedAt: now, UpdatedAt: now}},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	apiParentID := openapi_types.UUID(parentID)
	header := "Шапка"
	icon := "📒"
	expected := api.GetNote200JSONResponse(api.Note{
		Id:           openapi_types.UUID(noteID),
		ParentNoteId: &apiParentID,
		CreatedBy:    openapi_types.UUID(userID),
		Title:        "Заметка",
		Header:       &header,
		Icon:         &icon,
		Blocks:       &[]api.Block{{Id: openapi_types.UUID(blockID), Content: map[string]interface{}{"text": "Привет"}, CreatedAt: now, UpdatedAt: now}},
		Tags:         &[]api.Tag{{Id: openapi_types.UUID(tagID), Name: "учёба", Color: "#FF880080", CreatedAt: now, UpdatedAt: now}},
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	tests := []struct {
		name     string
		ctx      context.Context
		prepare  func(uc *mocks.MockUsecase)
		expected api.GetNoteResponseObject
	}{
		{
			name: "ok",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNote(gomock.Any(), userID, noteID).Return(note, nil)
			},
			expected: expected,
		},
		{
			name: "not found",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetNote(gomock.Any(), userID, noteID).Return(models.Note{}, models.ErrNoteNotFound)
			},
			expected: api.GetNote404Response{},
		},
		{
			name:     "no user",
			ctx:      context.Background(),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.GetNote401Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			resp, err := NewHandler(uc).GetNote(tt.ctx, api.GetNoteRequestObject{Id: openapi_types.UUID(noteID)})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestCreateNote(t *testing.T) {
	userID := uuid.NewV4()
	parentID := uuid.NewV4()
	apiParentID := openapi_types.UUID(parentID)
	title := "Новая заметка"
	body := &api.CreateNoteJSONRequestBody{Title: &title, ParentNoteId: &apiParentID}

	tests := []struct {
		name     string
		ctx      context.Context
		prepare  func(uc *mocks.MockUsecase)
		expected api.CreateNoteResponseObject
	}{
		{
			name: "ok",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				req := notes.CreateNoteRequest{Title: title, ParentNoteID: &parentID}
				uc.EXPECT().CreateNote(gomock.Any(), userID, req).Return(models.Note{Title: title}, nil)
			},
			expected: api.CreateNote201JSONResponse(api.Note{Title: title}),
		},
		{
			name:     "no user",
			ctx:      context.Background(),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.CreateNote401Response{},
		},
		{
			name: "parent not found",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CreateNote(gomock.Any(), userID, gomock.Any()).Return(models.Note{}, models.ErrNoteNotFound)
			},
			expected: api.CreateNote404Response{},
		},
		{
			name: "validation",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CreateNote(gomock.Any(), userID, gomock.Any()).Return(models.Note{}, models.ErrValidation)
			},
			expected: api.CreateNote400Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			resp, err := NewHandler(uc).CreateNote(tt.ctx, api.CreateNoteRequestObject{Body: body})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestUpdateNote(t *testing.T) {
	userID := uuid.NewV4()
	noteID := uuid.NewV4()
	tagID := uuid.NewV4()

	tests := []struct {
		name     string
		ctx      context.Context
		tagIDs   *[]openapi_types.UUID
		prepare  func(uc *mocks.MockUsecase)
		expected api.UpdateNoteResponseObject
	}{
		{
			name:   "ok with tags",
			ctx:    ctxutil.WithUserID(context.Background(), userID),
			tagIDs: &[]openapi_types.UUID{openapi_types.UUID(tagID)},
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateNote(gomock.Any(), userID, noteID, gomock.Any()).
					DoAndReturn(func(ctx context.Context, userID, noteID uuid.UUID, req notes.UpdateNoteRequest) (models.Note, error) {
						assert.Equal(t, []uuid.UUID{tagID}, req.TagIDs)
						return models.Note{Title: "t", Tags: []models.Tag{{ID: tagID, Name: "учёба"}}}, nil
					})
			},
			expected: api.UpdateNote200JSONResponse(api.Note{
				Title: "t",
				Tags:  &[]api.Tag{{Id: openapi_types.UUID(tagID), Name: "учёба"}},
			}),
		},
		{
			name: "tag_ids not sent",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateNote(gomock.Any(), userID, noteID, gomock.Any()).
					DoAndReturn(func(ctx context.Context, userID, noteID uuid.UUID, req notes.UpdateNoteRequest) (models.Note, error) {
						assert.Nil(t, req.TagIDs)
						return models.Note{Title: "t"}, nil
					})
			},
			expected: api.UpdateNote200JSONResponse(api.Note{Title: "t"}),
		},
		{
			name:   "empty tag_ids",
			ctx:    ctxutil.WithUserID(context.Background(), userID),
			tagIDs: &[]openapi_types.UUID{},
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateNote(gomock.Any(), userID, noteID, gomock.Any()).
					DoAndReturn(func(ctx context.Context, userID, noteID uuid.UUID, req notes.UpdateNoteRequest) (models.Note, error) {
						assert.NotNil(t, req.TagIDs)
						assert.Empty(t, req.TagIDs)
						return models.Note{Title: "t", Tags: []models.Tag{}}, nil
					})
			},
			expected: api.UpdateNote200JSONResponse(api.Note{Title: "t", Tags: &[]api.Tag{}}),
		},
		{
			name:     "no user",
			ctx:      context.Background(),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.UpdateNote401Response{},
		},
		{
			name: "note not found",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateNote(gomock.Any(), userID, noteID, gomock.Any()).Return(models.Note{}, models.ErrNoteNotFound)
			},
			expected: api.UpdateNote404Response{},
		},
		{
			name: "foreign block",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateNote(gomock.Any(), userID, noteID, gomock.Any()).Return(models.Note{}, models.ErrBlockNotFound)
			},
			expected: api.UpdateNote400Response{},
		},
		{
			name: "foreign tag",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateNote(gomock.Any(), userID, noteID, gomock.Any()).Return(models.Note{}, models.ErrTagNotFound)
			},
			expected: api.UpdateNote400Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			req := api.UpdateNoteRequestObject{
				Id:   openapi_types.UUID(noteID),
				Body: &api.UpdateNoteJSONRequestBody{Title: "t", TagIds: tt.tagIDs},
			}
			resp, err := NewHandler(uc).UpdateNote(tt.ctx, req)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestUpdateNoteBlocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := mocks.NewMockUsecase(ctrl)
	userID := uuid.NewV4()
	blockID := uuid.NewV4()
	apiBlockID := openapi_types.UUID(blockID)

	uc.EXPECT().UpdateNote(gomock.Any(), userID, gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, userID, noteID uuid.UUID, req notes.UpdateNoteRequest) (models.Note, error) {
			require.Len(t, req.Blocks, 2)
			assert.Equal(t, &blockID, req.Blocks[0].ID)
			assert.Nil(t, req.Blocks[1].ID)
			assert.JSONEq(t, `{"text":"Привет"}`, string(req.Blocks[0].Content))
			return models.Note{}, nil
		})

	req := api.UpdateNoteRequestObject{
		Body: &api.UpdateNoteJSONRequestBody{
			Title: "t",
			Blocks: []api.BlockInput{
				{Id: &apiBlockID, Content: map[string]interface{}{"text": "Привет"}},
				{Content: map[string]interface{}{}},
			},
		},
	}
	_, err := NewHandler(uc).UpdateNote(ctxutil.WithUserID(context.Background(), userID), req)

	require.NoError(t, err)
}

func TestDeleteNote(t *testing.T) {
	userID := uuid.NewV4()
	noteID := uuid.NewV4()
	req := api.DeleteNoteRequestObject{Id: openapi_types.UUID(noteID)}

	tests := []struct {
		name     string
		ucErr    error
		expected api.DeleteNoteResponseObject
	}{
		{name: "ok", expected: api.DeleteNote204Response{}},
		{name: "not found", ucErr: models.ErrNoteNotFound, expected: api.DeleteNote404Response{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			uc.EXPECT().DeleteNote(gomock.Any(), userID, noteID).Return(tt.ucErr)

			resp, err := NewHandler(uc).DeleteNote(ctxutil.WithUserID(context.Background(), userID), req)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestHandlersNoUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewHandler(mocks.NewMockUsecase(ctrl))
	ctx := context.Background()

	listResp, err := h.ListNotes(ctx, api.ListNotesRequestObject{})
	require.NoError(t, err)
	assert.Equal(t, api.ListNotes401Response{}, listResp)

	deleteResp, err := h.DeleteNote(ctx, api.DeleteNoteRequestObject{})
	require.NoError(t, err)
	assert.Equal(t, api.DeleteNote401Response{}, deleteResp)
}

func TestHandlersInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := mocks.NewMockUsecase(ctrl)
	ctx := ctxutil.WithUserID(context.Background(), uuid.NewV4())
	dbErr := errors.New("db is down")

	uc.EXPECT().GetNotes(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr)
	uc.EXPECT().GetNote(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.Note{}, dbErr)
	uc.EXPECT().CreateNote(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.Note{}, dbErr)
	uc.EXPECT().UpdateNote(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(models.Note{}, dbErr)
	uc.EXPECT().DeleteNote(gomock.Any(), gomock.Any(), gomock.Any()).Return(dbErr)

	h := NewHandler(uc)

	_, err := h.ListNotes(ctx, api.ListNotesRequestObject{})
	assert.ErrorIs(t, err, dbErr)

	_, err = h.GetNote(ctx, api.GetNoteRequestObject{})
	assert.ErrorIs(t, err, dbErr)

	_, err = h.CreateNote(ctx, api.CreateNoteRequestObject{Body: &api.CreateNoteJSONRequestBody{}})
	assert.ErrorIs(t, err, dbErr)

	_, err = h.UpdateNote(ctx, api.UpdateNoteRequestObject{Body: &api.UpdateNoteJSONRequestBody{}})
	assert.ErrorIs(t, err, dbErr)

	_, err = h.DeleteNote(ctx, api.DeleteNoteRequestObject{})
	assert.ErrorIs(t, err, dbErr)
}

func intPtr(i int) *int {
	return &i
}

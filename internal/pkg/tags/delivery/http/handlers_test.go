package delivery

import (
	"context"
	"errors"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/tags/delivery/http/api"
	"bnn/internal/pkg/tags/mocks"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := mocks.NewMockUsecase(ctrl)
	userID := uuid.NewV4()
	tag := models.Tag{ID: uuid.NewV4(), Name: "учёба", Color: "#FF880080"}

	uc.EXPECT().ListTags(gomock.Any(), userID).Return([]models.Tag{tag}, nil)

	resp, err := NewHandler(uc).ListTags(ctxutil.WithUserID(context.Background(), userID), api.ListTagsRequestObject{})

	require.NoError(t, err)
	assert.Equal(t, api.ListTags200JSONResponse{toAPITag(tag)}, resp)
}

func TestCreateTag(t *testing.T) {
	userID := uuid.NewV4()
	body := &api.CreateTagJSONRequestBody{Name: "учёба", Color: "#FF880080"}

	tests := []struct {
		name     string
		ctx      context.Context
		prepare  func(uc *mocks.MockUsecase)
		expected api.CreateTagResponseObject
	}{
		{
			name: "ok",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CreateTag(gomock.Any(), userID, "учёба", "#FF880080").Return(models.Tag{Name: "учёба", Color: "#FF880080"}, nil)
			},
			expected: api.CreateTag201JSONResponse(api.Tag{Name: "учёба", Color: "#FF880080"}),
		},
		{
			name:     "no user",
			ctx:      context.Background(),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.CreateTag401Response{},
		},
		{
			name: "validation",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CreateTag(gomock.Any(), userID, gomock.Any(), gomock.Any()).Return(models.Tag{}, models.ErrValidation)
			},
			expected: api.CreateTag400Response{},
		},
		{
			name: "name taken",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CreateTag(gomock.Any(), userID, gomock.Any(), gomock.Any()).Return(models.Tag{}, models.ErrTagExists)
			},
			expected: api.CreateTag409Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			resp, err := NewHandler(uc).CreateTag(tt.ctx, api.CreateTagRequestObject{Body: body})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestUpdateTag(t *testing.T) {
	userID := uuid.NewV4()
	tagID := uuid.NewV4()
	req := api.UpdateTagRequestObject{
		Id:   openapi_types.UUID(tagID),
		Body: &api.UpdateTagJSONRequestBody{Name: "работа", Color: "#00000000"},
	}

	tests := []struct {
		name     string
		ucErr    error
		expected api.UpdateTagResponseObject
	}{
		{name: "ok", expected: api.UpdateTag200JSONResponse(api.Tag{Name: "работа", Color: "#00000000"})},
		{name: "validation", ucErr: models.ErrValidation, expected: api.UpdateTag400Response{}},
		{name: "not found", ucErr: models.ErrTagNotFound, expected: api.UpdateTag404Response{}},
		{name: "name taken", ucErr: models.ErrTagExists, expected: api.UpdateTag409Response{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)

			tag := models.Tag{}
			if tt.ucErr == nil {
				tag = models.Tag{Name: "работа", Color: "#00000000"}
			}
			uc.EXPECT().UpdateTag(gomock.Any(), userID, tagID, "работа", "#00000000").Return(tag, tt.ucErr)

			resp, err := NewHandler(uc).UpdateTag(ctxutil.WithUserID(context.Background(), userID), req)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestDeleteTag(t *testing.T) {
	userID := uuid.NewV4()
	tagID := uuid.NewV4()
	req := api.DeleteTagRequestObject{Id: openapi_types.UUID(tagID)}

	tests := []struct {
		name     string
		ucErr    error
		expected api.DeleteTagResponseObject
	}{
		{name: "ok", expected: api.DeleteTag204Response{}},
		{name: "not found", ucErr: models.ErrTagNotFound, expected: api.DeleteTag404Response{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			uc.EXPECT().DeleteTag(gomock.Any(), userID, tagID).Return(tt.ucErr)

			resp, err := NewHandler(uc).DeleteTag(ctxutil.WithUserID(context.Background(), userID), req)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestDeleteTagInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := mocks.NewMockUsecase(ctrl)
	userID := uuid.NewV4()
	dbErr := errors.New("db is down")

	uc.EXPECT().DeleteTag(gomock.Any(), userID, gomock.Any()).Return(dbErr)

	_, err := NewHandler(uc).DeleteTag(ctxutil.WithUserID(context.Background(), userID), api.DeleteTagRequestObject{})

	assert.ErrorIs(t, err, dbErr)
}

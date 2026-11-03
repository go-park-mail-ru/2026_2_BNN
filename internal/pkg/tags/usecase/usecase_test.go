package usecase

import (
	"context"
	"strings"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/tags/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepo(ctrl)
	userID := uuid.NewV4()

	expected := []models.Tag{{ID: uuid.NewV4(), CreatedBy: userID, Name: "учёба", Color: "#FF880080"}}
	repo.EXPECT().ReadTags(gomock.Any(), userID).Return(expected, nil)

	tags, err := NewUsecase(repo).ListTags(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, expected, tags)
}

func TestCreateTag(t *testing.T) {
	userID := uuid.NewV4()

	tests := []struct {
		name        string
		tagName     string
		color       string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name:    "ok",
			tagName: "  учёба  ",
			color:   "#FF880080",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().CreateTag(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:        "empty name",
			tagName:     "   ",
			color:       "#FF880080",
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrValidation,
		},
		{
			name:        "long name",
			tagName:     strings.Repeat("я", 31),
			color:       "#FF880080",
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrValidation,
		},
		{
			name:        "color without alpha",
			tagName:     "учёба",
			color:       "#FF8800",
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrValidation,
		},
		{
			name:    "name taken",
			tagName: "учёба",
			color:   "#FF880080",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().CreateTag(gomock.Any(), gomock.Any()).Return(models.ErrTagExists)
			},
			expectedErr: models.ErrTagExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			tag, err := NewUsecase(repo).CreateTag(context.Background(), userID, tt.tagName, tt.color)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "учёба", tag.Name)
			assert.Equal(t, userID, tag.CreatedBy)
		})
	}
}

func TestUpdateTag(t *testing.T) {
	userID := uuid.NewV4()
	tagID := uuid.NewV4()
	own := models.Tag{ID: tagID, CreatedBy: userID, Name: "учёба", Color: "#FF880080"}

	tests := []struct {
		name        string
		color       string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name:  "ok",
			color: "#00000000",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadTag(gomock.Any(), tagID).Return(own, nil)
				repo.EXPECT().UpdateTag(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:  "not found",
			color: "#00000000",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadTag(gomock.Any(), tagID).Return(models.Tag{}, models.ErrTagNotFound)
			},
			expectedErr: models.ErrTagNotFound,
		},
		{
			name:  "foreign tag",
			color: "#00000000",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadTag(gomock.Any(), tagID).Return(models.Tag{ID: tagID, CreatedBy: uuid.NewV4()}, nil)
			},
			expectedErr: models.ErrTagNotFound,
		},
		{
			name:        "bad color",
			color:       "red",
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			tag, err := NewUsecase(repo).UpdateTag(context.Background(), userID, tagID, "работа", tt.color)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "работа", tag.Name)
			assert.Equal(t, tt.color, tag.Color)
		})
	}
}

func TestDeleteTag(t *testing.T) {
	userID := uuid.NewV4()
	tagID := uuid.NewV4()

	tests := []struct {
		name        string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name: "ok",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadTag(gomock.Any(), tagID).Return(models.Tag{ID: tagID, CreatedBy: userID}, nil)
				repo.EXPECT().DeleteTag(gomock.Any(), tagID).Return(nil)
			},
		},
		{
			name: "foreign tag",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadTag(gomock.Any(), tagID).Return(models.Tag{ID: tagID, CreatedBy: uuid.NewV4()}, nil)
			},
			expectedErr: models.ErrTagNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			err := NewUsecase(repo).DeleteTag(context.Background(), userID, tagID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

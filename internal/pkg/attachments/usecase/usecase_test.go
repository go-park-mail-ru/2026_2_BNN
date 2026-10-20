package usecase

import (
	"context"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUploadAttachment(t *testing.T) {
	userID := uuid.NewV4()
	blockID := uuid.NewV4()
	data := []byte("file")

	tests := []struct {
		name        string
		contentType string
		prepare     func(repo *mocks.MockRepo, files *mocks.MockFileRepo)
		expectedErr error
	}{
		{
			name:        "ok",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(userID, nil)
				files.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(nil)
				repo.EXPECT().CreateAttachment(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:        "foreign block",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(uuid.NewV4(), nil)
			},
			expectedErr: models.ErrBlockNotFound,
		},
		{
			name:        "unsupported type",
			contentType: "text/plain; charset=utf-8",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(userID, nil)
			},
			expectedErr: models.ErrUnsupportedFileType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			files := mocks.NewMockFileRepo(ctrl)
			tt.prepare(repo, files)

			u := NewUsecase(repo, files)

			attachment, err := u.UploadAttachment(context.Background(), userID, blockID, data, tt.contentType)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, blockID, attachment.BlockID)
			assert.Equal(t, attachment.ID.String()+".png", attachment.Path)
		})
	}
}

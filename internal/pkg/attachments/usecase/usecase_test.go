package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errDB = errors.New("db is down")

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
			name:        "block not found",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(uuid.Nil, models.ErrBlockNotFound)
			},
			expectedErr: models.ErrBlockNotFound,
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
		{
			name:        "storage error",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(userID, nil)
				files.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(errDB)
			},
			expectedErr: errDB,
		},
		{
			name:        "db error removes file",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(userID, nil)
				files.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(nil)
				repo.EXPECT().CreateAttachment(gomock.Any(), gomock.Any()).Return(errDB)
				files.EXPECT().DeleteFile(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			files := mocks.NewMockFileRepo(ctrl)
			tt.prepare(repo, files)

			attachment, err := NewUsecase(repo, files).UploadAttachment(context.Background(), userID, blockID, data, tt.contentType)

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

func TestGetAttachment(t *testing.T) {
	userID := uuid.NewV4()
	blockID := uuid.NewV4()
	attachmentID := uuid.NewV4()
	attachment := models.Attachment{ID: attachmentID, BlockID: blockID, Path: "a.png"}
	file := models.File{Body: io.NopCloser(strings.NewReader("file")), ContentType: "image/png", Size: 4}

	tests := []struct {
		name        string
		prepare     func(repo *mocks.MockRepo, files *mocks.MockFileRepo)
		expectedErr error
	}{
		{
			name: "ok",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(userID, nil)
				files.EXPECT().GetFile(gomock.Any(), "a.png").Return(file, nil)
			},
		},
		{
			name: "not found",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(models.Attachment{}, models.ErrAttachmentNotFound)
			},
			expectedErr: models.ErrAttachmentNotFound,
		},
		{
			name: "foreign attachment",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(uuid.NewV4(), nil)
			},
			expectedErr: models.ErrAttachmentNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			files := mocks.NewMockFileRepo(ctrl)
			tt.prepare(repo, files)

			got, err := NewUsecase(repo, files).GetAttachment(context.Background(), userID, attachmentID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, file, got)
		})
	}
}

func TestDeleteAttachment(t *testing.T) {
	userID := uuid.NewV4()
	blockID := uuid.NewV4()
	attachmentID := uuid.NewV4()
	attachment := models.Attachment{ID: attachmentID, BlockID: blockID, Path: "a.png"}

	tests := []struct {
		name        string
		prepare     func(repo *mocks.MockRepo, files *mocks.MockFileRepo)
		expectedErr error
	}{
		{
			name: "ok",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(userID, nil)
				repo.EXPECT().DeleteAttachment(gomock.Any(), attachmentID).Return(nil)
				files.EXPECT().DeleteFile(gomock.Any(), "a.png").Return(nil)
			},
		},
		{
			name: "foreign attachment",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(uuid.NewV4(), nil)
			},
			expectedErr: models.ErrAttachmentNotFound,
		},
		{
			name: "db error keeps file",
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockOwner(gomock.Any(), blockID).Return(userID, nil)
				repo.EXPECT().DeleteAttachment(gomock.Any(), attachmentID).Return(errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			files := mocks.NewMockFileRepo(ctrl)
			tt.prepare(repo, files)

			err := NewUsecase(repo, files).DeleteAttachment(context.Background(), userID, attachmentID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

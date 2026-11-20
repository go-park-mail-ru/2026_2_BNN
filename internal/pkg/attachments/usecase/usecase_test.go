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

// Мок для NotesClient (проверка доступа к заметке)
type mockNotesClient struct {
	role string
	err  error
}

func (m *mockNotesClient) CheckAccess(ctx context.Context, userID, noteID uuid.UUID) (string, error) {
	return m.role, m.err
}

func TestUploadAttachment(t *testing.T) {
	userID := uuid.NewV4()
	blockID := uuid.NewV4()
	noteID := uuid.NewV4()
	data := []byte("file")

	tests := []struct {
		name        string
		contentType string
		notesMock   *mockNotesClient
		prepare     func(repo *mocks.MockRepo, files *mocks.MockFileRepo)
		expectedErr error
	}{
		{
			name:        "ok (owner)",
			contentType: "image/png",
			notesMock:   &mockNotesClient{role: "owner", err: nil},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				// Узнаем заметку по блоку и проверяем доступ
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
				// Ожидаем, что CheckAccess вызовется (его реализует мок)

				files.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(nil)
				repo.EXPECT().CreateAttachment(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:        "ok (editor)",
			contentType: "image/png",
			notesMock:   &mockNotesClient{role: "editor", err: nil},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
				files.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(nil)
				repo.EXPECT().CreateAttachment(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:        "block not found",
			contentType: "image/png",
			notesMock:   &mockNotesClient{role: "owner", err: nil},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(uuid.Nil, models.ErrBlockNotFound)
			},
			expectedErr: models.ErrBlockNotFound,
		},
		{
			name:        "no access (viewer)",
			contentType: "image/png",
			notesMock:   &mockNotesClient{role: "viewer", err: nil}, // viewer не может загружать
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
			},
			expectedErr: models.ErrForbidden, // Ожидаем ошибку доступа
		},
		{
			name:        "unsupported type",
			contentType: "text/plain; charset=utf-8",
			notesMock:   &mockNotesClient{role: "owner", err: nil},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
			},
			expectedErr: models.ErrUnsupportedFileType,
		},
		{
			name:        "storage error",
			contentType: "image/png",
			notesMock:   &mockNotesClient{role: "owner", err: nil},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
				files.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(errDB)
			},
			expectedErr: errDB,
		},
		{
			name:        "db error removes file",
			contentType: "image/png",
			notesMock:   &mockNotesClient{role: "owner", err: nil},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
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

			// Передаем 3 аргумента
			uc := NewUsecase(repo, files, tt.notesMock)
			attachment, err := uc.UploadAttachment(context.Background(), userID, blockID, data, tt.contentType)

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
	noteID := uuid.NewV4()
	attachmentID := uuid.NewV4()
	attachment := models.Attachment{ID: attachmentID, BlockID: blockID, Path: "a.png"}
	file := models.File{Body: io.NopCloser(strings.NewReader("file")), ContentType: "image/png", Size: 4}

	tests := []struct {
		name        string
		notesMock   *mockNotesClient
		prepare     func(repo *mocks.MockRepo, files *mocks.MockFileRepo)
		expectedErr error
	}{
		{
			name:      "ok",
			notesMock: &mockNotesClient{role: "owner", err: nil},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
				files.EXPECT().GetFile(gomock.Any(), "a.png").Return(file, nil)
			},
		},
		{
			name:      "not found",
			notesMock: &mockNotesClient{},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(models.Attachment{}, models.ErrAttachmentNotFound)
			},
			expectedErr: models.ErrAttachmentNotFound,
		},
		{
			name:      "no access",
			notesMock: &mockNotesClient{err: models.ErrNoteNotFound}, // Доступ к заметке пропал
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
			},
			expectedErr: models.ErrAttachmentNotFound, // Скрываем наличие файла
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			files := mocks.NewMockFileRepo(ctrl)
			tt.prepare(repo, files)

			uc := NewUsecase(repo, files, tt.notesMock)
			got, err := uc.GetAttachment(context.Background(), userID, attachmentID)

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
	noteID := uuid.NewV4()
	attachmentID := uuid.NewV4()
	attachment := models.Attachment{ID: attachmentID, BlockID: blockID, Path: "a.png"}

	tests := []struct {
		name        string
		notesMock   *mockNotesClient
		prepare     func(repo *mocks.MockRepo, files *mocks.MockFileRepo)
		expectedErr error
	}{
		{
			name:      "ok",
			notesMock: &mockNotesClient{role: "owner", err: nil},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
				repo.EXPECT().DeleteAttachment(gomock.Any(), attachmentID).Return(nil)
				files.EXPECT().DeleteFile(gomock.Any(), "a.png").Return(nil)
			},
		},
		{
			name:      "foreign attachment",
			notesMock: &mockNotesClient{err: models.ErrNoteNotFound},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
			},
			expectedErr: models.ErrAttachmentNotFound,
		},
		{
			name:      "db error keeps file",
			notesMock: &mockNotesClient{role: "owner", err: nil},
			prepare: func(repo *mocks.MockRepo, files *mocks.MockFileRepo) {
				repo.EXPECT().ReadAttachment(gomock.Any(), attachmentID).Return(attachment, nil)
				repo.EXPECT().ReadBlockNoteID(gomock.Any(), blockID).Return(noteID, nil)
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

			uc := NewUsecase(repo, files, tt.notesMock)
			err := uc.DeleteAttachment(context.Background(), userID, attachmentID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

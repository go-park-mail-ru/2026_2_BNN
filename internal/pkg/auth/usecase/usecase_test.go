package usecase

import (
	"context"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/auth/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestChangePassword(t *testing.T) {
	userID := uuid.NewV4()

	hash, err := hashPassword("oldpassword")
	require.NoError(t, err)

	user := models.User{ID: userID, PasswordHash: hash}

	tests := []struct {
		name        string
		oldPassword string
		newPassword string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name:        "ok",
			oldPassword: "oldpassword",
			newPassword: "newpassword",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(user, nil)
				repo.EXPECT().UpdatePassword(gomock.Any(), userID, user.PasswordHash, gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:        "wrong old password",
			oldPassword: "wrongpassword",
			newPassword: "newpassword",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(user, nil)
			},
			expectedErr: models.ErrWrongPassword,
		},
		{
			name:        "new password too short",
			oldPassword: "oldpassword",
			newPassword: "short",
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			u := NewUsecase(repo, nil, "secret")

			tokens, err := u.ChangePassword(context.Background(), userID, tt.oldPassword, tt.newPassword)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, tokens.AccessToken)
			assert.NotEmpty(t, tokens.RefreshToken)
		})
	}
}

func TestSignUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepo(ctrl)
	uc := NewUsecase(mockRepo, nil, "test_secret")

	t.Run("Success", func(t *testing.T) {
		login := "testuser"
		password := "password123"

		mockRepo.EXPECT().
			CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		user, tokens, err := uc.SignUp(context.Background(), login, password)

		require.NoError(t, err)
		assert.Equal(t, login, user.Login)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("Validation Error - Short Login", func(t *testing.T) {
		_, _, err := uc.SignUp(context.Background(), "ab", "password123")

		assert.ErrorIs(t, err, models.ErrValidation)
	})
}

func TestUpdateAvatar(t *testing.T) {
	userID := uuid.NewV4()
	data := []byte("image")

	tests := []struct {
		name        string
		contentType string
		prepare     func(repo *mocks.MockRepo, storage *mocks.MockStorage)
		expectedErr error
	}{
		{
			name:        "ok",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, storage *mocks.MockStorage) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).
					Return(models.User{ID: userID, Avatar: "/avatars/old.jpg"}, nil)
				storage.EXPECT().PublicPath(gomock.Any()).Return("/avatars/new.png", nil)
				storage.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(nil)
				repo.EXPECT().UpdateAvatar(gomock.Any(), userID, "/avatars/new.png").Return(nil)
				storage.EXPECT().DeleteFile(gomock.Any(), "old.jpg").Return(nil)
			},
		},
		{
			name:        "unsupported type",
			contentType: "text/plain; charset=utf-8",
			prepare:     func(repo *mocks.MockRepo, storage *mocks.MockStorage) {},
			expectedErr: models.ErrUnsupportedFileType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			storage := mocks.NewMockStorage(ctrl)
			tt.prepare(repo, storage)

			uc := NewUsecase(repo, storage, "test_secret")

			user, err := uc.UpdateAvatar(context.Background(), userID, data, tt.contentType)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "/avatars/new.png", user.Avatar)
		})
	}
}

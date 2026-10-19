package usecase_test

import (
	"context"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/auth/mocks"
	"bnn/internal/pkg/auth/usecase"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSignUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepo(ctrl)
	mockS3 := mocks.NewMockS3(ctrl)

	uc := usecase.NewUsecase(mockRepo, mockS3, "test_secret")

	t.Run("Success", func(t *testing.T) {
		login := "testuser"
		password := "password123"

		mockRepo.EXPECT().
			CreateUser(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		mockRepo.EXPECT().
			CreateRefreshToken(gomock.Any(), gomock.Any()).
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepo(ctrl)
	mockS3 := mocks.NewMockS3(ctrl)

	uc := usecase.NewUsecase(mockRepo, mockS3, "test_secret")
	userID := uuid.NewV4()

	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	t.Run("Success", func(t *testing.T) {
		mockRepo.EXPECT().
			ReadUserByID(gomock.Any(), userID).
			Return(models.User{ID: userID, Avatar: "/static/avatars/old.jpg"}, nil)

		mockS3.EXPECT().
			Upload(gomock.Any(), "avatars", userID.String()+".png", gomock.Any(), "image/png").
			Return("/static/avatars/new.png", nil)

		mockRepo.EXPECT().
			UpdateAvatar(gomock.Any(), gomock.Any()).
			Return(nil)

		mockS3.EXPECT().
			Delete(gomock.Any(), "avatars", "old.jpg").
			Return(nil)

		user, err := uc.UpdateAvatar(context.Background(), userID, pngHeader)

		require.NoError(t, err)
		assert.Equal(t, "/static/avatars/new.png", user.Avatar)
	})

	t.Run("Invalid File Type", func(t *testing.T) {
		txtBytes := []byte("this is a text file, not an image")

		_, err := uc.UpdateAvatar(context.Background(), userID, txtBytes)

		assert.ErrorIs(t, err, models.ErrUnsupportedMediaType)
	})
}

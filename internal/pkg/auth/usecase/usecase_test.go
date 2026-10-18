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
				repo.EXPECT().UpdatePassword(gomock.Any(), userID, gomock.Any()).Return(nil)
				repo.EXPECT().DeleteUserRefreshTokens(gomock.Any(), userID).Return(nil)
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(nil)
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

			u := NewUsecase(repo, "secret")

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

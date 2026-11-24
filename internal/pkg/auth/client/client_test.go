package client

import (
	"context"
	"testing"
	"time"

	"bnn/internal/models"
	authgrpc "bnn/internal/pkg/auth/delivery/grpc"
	"bnn/internal/pkg/auth/delivery/grpc/gen"
	"bnn/internal/pkg/auth/mocks"
	"bnn/internal/pkg/grpcutil/grpctest"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

func newClient(t *testing.T) (*Client, *mocks.MockUsecase) {
	uc := mocks.NewMockUsecase(gomock.NewController(t))
	conn := grpctest.Dial(t, func(server *grpc.Server) {
		gen.RegisterAuthServer(server, authgrpc.NewHandler(uc))
	})
	return NewClient(conn), uc
}

func TestClient(t *testing.T) {
	ctx := context.Background()
	client, uc := newClient(t)

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	userID := uuid.NewV4()
	user := models.User{ID: userID, Login: "seyran", CreatedAt: now, UpdatedAt: now}
	tokens := models.AuthTokens{AccessToken: "access", AccessExpiresAt: now, RefreshToken: "refresh", RefreshExpiresAt: now}

	uc.EXPECT().SignUp(gomock.Any(), "seyran", "password").Return(user, tokens, nil)
	gotUser, gotTokens, err := client.SignUp(ctx, "seyran", "password")
	require.NoError(t, err)
	assert.Equal(t, user, gotUser)
	assert.Equal(t, tokens, gotTokens)

	uc.EXPECT().SignIn(gomock.Any(), "seyran", "password").Return(user, tokens, nil)
	gotUser, _, err = client.SignIn(ctx, "seyran", "password")
	require.NoError(t, err)
	assert.Equal(t, user, gotUser)

	uc.EXPECT().VerifyTwoFactor(gomock.Any(), "token", "123456").Return(user, tokens, nil)
	gotUser, _, err = client.VerifyTwoFactor(ctx, "token", "123456")
	require.NoError(t, err)
	assert.Equal(t, user, gotUser)

	uc.EXPECT().Refresh(gomock.Any(), "refresh").Return(tokens, nil)
	gotTokens, err = client.Refresh(ctx, "refresh")
	require.NoError(t, err)
	assert.Equal(t, tokens, gotTokens)

	uc.EXPECT().Logout(gomock.Any(), "refresh").Return(nil)
	assert.NoError(t, client.Logout(ctx, "refresh"))

	uc.EXPECT().Authenticate(gomock.Any(), "access").Return(userID, nil)
	gotID, err := client.Authenticate(ctx, "access")
	require.NoError(t, err)
	assert.Equal(t, userID, gotID)

	uc.EXPECT().GetUser(gomock.Any(), userID).Return(user, nil)
	gotUser, err = client.GetUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, user, gotUser)

	uc.EXPECT().ChangePassword(gomock.Any(), userID, "old", "new").Return(tokens, nil)
	gotTokens, err = client.ChangePassword(ctx, userID, "old", "new")
	require.NoError(t, err)
	assert.Equal(t, tokens, gotTokens)

	uc.EXPECT().UpdateAvatar(gomock.Any(), userID, []byte("png"), "image/png").Return(user, nil)
	gotUser, err = client.UpdateAvatar(ctx, userID, []byte("png"), "image/png")
	require.NoError(t, err)
	assert.Equal(t, user, gotUser)

	uc.EXPECT().Generate2FA(gomock.Any(), userID).Return([]byte("qr"), nil)
	qr, err := client.Generate2FA(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, []byte("qr"), qr)

	uc.EXPECT().Enable2FA(gomock.Any(), userID, "123456").Return(nil)
	assert.NoError(t, client.Enable2FA(ctx, userID, "123456"))

	uc.EXPECT().Disable2FA(gomock.Any(), userID, "123456").Return(nil)
	assert.NoError(t, client.Disable2FA(ctx, userID, "123456"))

	uc.EXPECT().GetOAuthURL(gomock.Any(), "github", "state").Return("https://github.com/login", nil)
	url, err := client.GetOAuthURL(ctx, "github", "state")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/login", url)

	uc.EXPECT().HandleOAuthCallback(gomock.Any(), "github", "code").Return(user, tokens, nil)
	gotUser, _, err = client.HandleOAuthCallback(ctx, "github", "code")
	require.NoError(t, err)
	assert.Equal(t, user, gotUser)

	uc.EXPECT().GetUser(gomock.Any(), userID).Return(user, nil)
	exists, err := client.CheckUserExists(ctx, userID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestClientErrors(t *testing.T) {
	ctx := context.Background()
	client, uc := newClient(t)
	userID := uuid.NewV4()

	uc.EXPECT().SignUp(gomock.Any(), "seyran", "password").Return(models.User{}, models.AuthTokens{}, models.ErrUserExists)
	_, _, err := client.SignUp(ctx, "seyran", "password")
	assert.ErrorIs(t, err, models.ErrUserExists)

	uc.EXPECT().SignIn(gomock.Any(), "seyran", "password").Return(models.User{}, models.AuthTokens{}, models.ErrInvalidCredentials)
	_, _, err = client.SignIn(ctx, "seyran", "password")
	assert.ErrorIs(t, err, models.ErrInvalidCredentials)

	uc.EXPECT().Refresh(gomock.Any(), "refresh").Return(models.AuthTokens{}, models.ErrInvalidToken)
	_, err = client.Refresh(ctx, "refresh")
	assert.ErrorIs(t, err, models.ErrInvalidToken)

	uc.EXPECT().Authenticate(gomock.Any(), "access").Return(uuid.Nil, models.ErrInvalidToken)
	_, err = client.Authenticate(ctx, "access")
	assert.ErrorIs(t, err, models.ErrInvalidToken)

	uc.EXPECT().GetUser(gomock.Any(), userID).Return(models.User{}, models.ErrUserNotFound)
	_, err = client.GetUser(ctx, userID)
	assert.ErrorIs(t, err, models.ErrUserNotFound)

	uc.EXPECT().ChangePassword(gomock.Any(), userID, "old", "new").Return(models.AuthTokens{}, models.ErrWrongPassword)
	_, err = client.ChangePassword(ctx, userID, "old", "new")
	assert.ErrorIs(t, err, models.ErrWrongPassword)

	uc.EXPECT().Generate2FA(gomock.Any(), userID).Return(nil, models.ErrTwoFactorEnabled)
	_, err = client.Generate2FA(ctx, userID)
	assert.ErrorIs(t, err, models.ErrTwoFactorEnabled)

	uc.EXPECT().Enable2FA(gomock.Any(), userID, "000000").Return(models.ErrInvalidCode)
	assert.ErrorIs(t, client.Enable2FA(ctx, userID, "000000"), models.ErrInvalidCode)

	uc.EXPECT().GetUser(gomock.Any(), userID).Return(models.User{}, models.ErrUserNotFound)
	exists, err := client.CheckUserExists(ctx, userID)
	require.NoError(t, err)
	assert.False(t, exists)
}

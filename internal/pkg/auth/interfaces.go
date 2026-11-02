package auth

import (
	"context"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

//go:generate go run go.uber.org/mock/mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type Usecase interface {
	SignUp(ctx context.Context, login, password string) (models.User, models.AuthTokens, error)
	SignIn(ctx context.Context, login, password string) (models.User, models.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (models.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	Authenticate(ctx context.Context, accessToken string) (uuid.UUID, error)
	GetUser(ctx context.Context, userID uuid.UUID) (models.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (models.AuthTokens, error)
	UpdateAvatar(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (models.User, error)
	Generate2FA(ctx context.Context, userID uuid.UUID) ([]byte, error)
	Enable2FA(ctx context.Context, userID uuid.UUID, code string) error
	Disable2FA(ctx context.Context, userID uuid.UUID) error
	GetOAuthURL(provider string, state string) string
	HandleOAuthCallback(ctx context.Context, provider, code string) (models.User, models.AuthTokens, error)
}

type Repo interface {
	CreateUser(ctx context.Context, user models.User) error
	ReadUserByLogin(ctx context.Context, login string) (models.User, error)
	ReadUserByID(ctx context.Context, id uuid.UUID) (models.User, error)
	UpdateAvatar(ctx context.Context, user models.User) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, hash []byte) error

	CreateRefreshToken(ctx context.Context, token models.RefreshToken) error
	DeleteRefreshToken(ctx context.Context, tokenHash []byte) (models.RefreshToken, error)
	DeleteUserRefreshTokens(ctx context.Context, userID uuid.UUID) error

	Set2FAStatus(ctx context.Context, userID uuid.UUID, enabled bool) error
	UpdateTOTPSecret(ctx context.Context, userID uuid.UUID, secret string) error

	ReadUserByEmail(ctx context.Context, email string) (models.User, error)
}

type Storage interface {
	PutFile(ctx context.Context, key string, data []byte, contentType string) error
	DeleteFile(ctx context.Context, key string) error
	PublicPath(key string) (string, error)
}

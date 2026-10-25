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
}

type Repo interface {
	CreateUser(ctx context.Context, user models.User, session models.RefreshToken) error
	ReadUserByLogin(ctx context.Context, login string) (models.User, error)
	ReadUserByID(ctx context.Context, id uuid.UUID) (models.User, error)
	UpdateAvatar(ctx context.Context, userID uuid.UUID, avatar string) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, oldHash, newHash []byte, session models.RefreshToken) error

	CreateSession(ctx context.Context, session models.RefreshToken) error
	RotateSession(ctx context.Context, oldTokenHash []byte, session models.RefreshToken) (uuid.UUID, error)
	RevokeSession(ctx context.Context, tokenHash []byte) error
}

type Storage interface {
	PutFile(ctx context.Context, key string, data []byte, contentType string) error
	DeleteFile(ctx context.Context, key string) error
	PublicPath(key string) (string, error)
}

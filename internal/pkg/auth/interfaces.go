package auth

import (
	"context"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type Usecase interface {
	SignUp(ctx context.Context, login, password string) (models.User, models.AuthTokens, error)
	SignIn(ctx context.Context, login, password string) (models.User, models.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (models.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	Authenticate(ctx context.Context, accessToken string) (uuid.UUID, error)
	GetUser(ctx context.Context, userID uuid.UUID) (models.User, error)
}

type Repo interface {
	CreateUser(ctx context.Context, user models.User) error
	ReadUserByLogin(ctx context.Context, login string) (models.User, error)
	ReadUserByID(ctx context.Context, id uuid.UUID) (models.User, error)

	CreateRefreshToken(ctx context.Context, token models.RefreshToken) error
	DeleteRefreshToken(ctx context.Context, tokenHash []byte) (models.RefreshToken, error)
}

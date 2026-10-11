package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"

	uuid "github.com/satori/go.uuid"
)

const (
	minLoginLength    = 3
	maxLoginLength    = 20
	minPasswordLength = 8
	maxPasswordLength = 128

	defaultAvatar = "default_avatar.jpg"
)

type Usecase struct {
	repo   auth.Repo
	secret []byte
}

func NewUsecase(repo auth.Repo, secret string) *Usecase {
	return &Usecase{
		repo:   repo,
		secret: []byte(secret),
	}
}

func (u *Usecase) SignUp(ctx context.Context, login, password string) (models.User, models.AuthTokens, error) {
	if err := validateCredentials(login, password); err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	now := time.Now().UTC()

	user := models.User{
		ID:           uuid.NewV4(),
		Login:        login,
		PasswordHash: passwordHash,
		Avatar:       defaultAvatar,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := u.repo.CreateUser(ctx, user); err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	tokens, err := u.createTokens(ctx, user.ID)
	if err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	return user, tokens, nil
}

func (u *Usecase) SignIn(ctx context.Context, login, password string) (models.User, models.AuthTokens, error) {
	if err := validateCredentials(login, password); err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	user, err := u.repo.ReadUserByLogin(ctx, login)
	if errors.Is(err, models.ErrUserNotFound) {
		return models.User{}, models.AuthTokens{}, models.ErrInvalidCredentials
	}
	if err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	if !verifyPassword(password, user.PasswordHash) {
		return models.User{}, models.AuthTokens{}, models.ErrInvalidCredentials
	}

	tokens, err := u.createTokens(ctx, user.ID)
	if err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	return user, tokens, nil
}

func (u *Usecase) Refresh(ctx context.Context, refreshToken string) (models.AuthTokens, error) {
	token, err := u.repo.DeleteRefreshToken(ctx, hashRefreshToken(refreshToken))
	if errors.Is(err, models.ErrRefreshTokenNotFound) {
		return models.AuthTokens{}, models.ErrInvalidToken
	}
	if err != nil {
		return models.AuthTokens{}, err
	}

	if time.Now().After(token.ExpiresAt) {
		return models.AuthTokens{}, models.ErrInvalidToken
	}

	return u.createTokens(ctx, token.UserID)
}

func (u *Usecase) Logout(ctx context.Context, refreshToken string) error {
	_, err := u.repo.DeleteRefreshToken(ctx, hashRefreshToken(refreshToken))
	if errors.Is(err, models.ErrRefreshTokenNotFound) {
		return nil
	}

	return err
}

func (u *Usecase) Authenticate(_ context.Context, accessToken string) (uuid.UUID, error) {
	userID, err := u.parseAccessToken(accessToken)
	if err != nil {
		return uuid.Nil, models.ErrInvalidToken
	}

	return userID, nil
}

func (u *Usecase) GetUser(ctx context.Context, userID uuid.UUID) (models.User, error) {
	return u.repo.ReadUserByID(ctx, userID)
}

func (u *Usecase) createTokens(ctx context.Context, userID uuid.UUID) (models.AuthTokens, error) {
	accessToken, err := u.newAccessToken(userID)
	if err != nil {
		return models.AuthTokens{}, fmt.Errorf("create access token: %w", err)
	}

	refreshToken := rand.Text()
	now := time.Now().UTC()

	err = u.repo.CreateRefreshToken(ctx, models.RefreshToken{
		TokenHash: hashRefreshToken(refreshToken),
		UserID:    userID,
		ExpiresAt: now.Add(refreshTokenTTL),
		CreatedAt: now,
	})
	if err != nil {
		return models.AuthTokens{}, err
	}

	return models.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func validateCredentials(login, password string) error {
	loginLength := utf8.RuneCountInString(login)
	if loginLength < minLoginLength || loginLength > maxLoginLength {
		return fmt.Errorf("%w: invalid login length", models.ErrValidation)
	}

	passwordLength := utf8.RuneCountInString(password)
	if passwordLength < minPasswordLength || passwordLength > maxPasswordLength {
		return fmt.Errorf("%w: invalid password length", models.ErrValidation)
	}

	return nil
}

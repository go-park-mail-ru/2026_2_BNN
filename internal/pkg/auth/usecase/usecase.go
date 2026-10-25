package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"path"
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
)

type Usecase struct {
	repo    auth.Repo
	storage auth.Storage
	secret  []byte
}

var allowedAvatarTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func NewUsecase(repo auth.Repo, storage auth.Storage, secret string) *Usecase {
	return &Usecase{
		repo:    repo,
		storage: storage,
		secret:  []byte(secret),
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
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	refreshToken, session := newSession(user.ID)
	if err := u.repo.CreateUser(ctx, user, session); err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	tokens, err := u.authTokens(user.ID, refreshToken, session)
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

	refreshToken, session := newSession(user.ID)
	if err := u.repo.CreateSession(ctx, session); err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	tokens, err := u.authTokens(user.ID, refreshToken, session)
	if err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	return user, tokens, nil
}

func (u *Usecase) Refresh(ctx context.Context, oldRefreshToken string) (models.AuthTokens, error) {
	refreshToken, session := newSession(uuid.Nil)

	userID, err := u.repo.RotateSession(ctx, hashRefreshToken(oldRefreshToken), session)
	if errors.Is(err, models.ErrRefreshTokenNotFound) {
		return models.AuthTokens{}, models.ErrInvalidToken
	}
	if err != nil {
		return models.AuthTokens{}, err
	}

	return u.authTokens(userID, refreshToken, session)
}

func (u *Usecase) Logout(ctx context.Context, refreshToken string) error {
	return u.repo.RevokeSession(ctx, hashRefreshToken(refreshToken))
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

func (u *Usecase) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (models.AuthTokens, error) {
	if err := validatePassword(newPassword); err != nil {
		return models.AuthTokens{}, err
	}

	user, err := u.repo.ReadUserByID(ctx, userID)
	if err != nil {
		return models.AuthTokens{}, err
	}

	if !verifyPassword(oldPassword, user.PasswordHash) {
		return models.AuthTokens{}, models.ErrWrongPassword
	}

	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return models.AuthTokens{}, err
	}

	refreshToken, session := newSession(userID)
	if err := u.repo.UpdatePassword(ctx, userID, user.PasswordHash, passwordHash, session); err != nil {
		return models.AuthTokens{}, err
	}

	return u.authTokens(userID, refreshToken, session)
}

func newSession(userID uuid.UUID) (string, models.RefreshToken) {
	refreshToken := rand.Text()

	return refreshToken, models.RefreshToken{
		TokenHash: hashRefreshToken(refreshToken),
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(refreshTokenTTL),
	}
}

func (u *Usecase) authTokens(userID uuid.UUID, refreshToken string, session models.RefreshToken) (models.AuthTokens, error) {
	accessExpiresAt := time.Now().UTC().Add(accessTokenTTL)

	accessToken, err := u.newAccessToken(userID, accessExpiresAt)
	if err != nil {
		return models.AuthTokens{}, fmt.Errorf("create access token: %w", err)
	}

	return models.AuthTokens{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: session.ExpiresAt,
	}, nil
}

func validateCredentials(login, password string) error {
	loginLength := utf8.RuneCountInString(login)
	if loginLength < minLoginLength || loginLength > maxLoginLength {
		return fmt.Errorf("%w: invalid login length", models.ErrValidation)
	}

	return validatePassword(password)
}

func validatePassword(password string) error {
	passwordLength := utf8.RuneCountInString(password)
	if passwordLength < minPasswordLength || passwordLength > maxPasswordLength {
		return fmt.Errorf("%w: invalid password length", models.ErrValidation)
	}

	return nil
}

func (u *Usecase) UpdateAvatar(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (models.User, error) {
	ext, ok := allowedAvatarTypes[contentType]
	if !ok {
		return models.User{}, models.ErrUnsupportedFileType
	}

	user, err := u.repo.ReadUserByID(ctx, userID)
	if err != nil {
		return models.User{}, err
	}

	key := fmt.Sprintf("%s%s", uuid.NewV4(), ext)
	avatar, err := u.storage.PublicPath(key)
	if err != nil {
		return models.User{}, err
	}
	if err := u.storage.PutFile(ctx, key, data, contentType); err != nil {
		return models.User{}, err
	}

	oldAvatar := user.Avatar
	user.Avatar = avatar
	user.UpdatedAt = time.Now().UTC()
	if err := u.repo.UpdateAvatar(ctx, userID, avatar); err != nil {
		_ = u.storage.DeleteFile(ctx, key)
		return models.User{}, err
	}

	if oldAvatar != "" {
		_ = u.storage.DeleteFile(ctx, path.Base(oldAvatar))
	}
	return user, nil
}

package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	avatarBucket  = "avatars"
)

type Usecase struct {
	repo   auth.Repo
	s3     auth.Repo
	secret []byte
}

var allowedAvatarTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func (u *Usecase) UpdateAvatar(ctx context.Context, userID uuid.UUID, data []byte) (models.User, error) {
	if len(data) == 0 {
		return models.User{}, models.ErrValidation
	}

	contentType := http.DetectContentType(data)
	ext, ok := allowedAvatarTypes[contentType]
	if !ok {
		return models.User{}, models.ErrUnsupportedMediaType
	}

	user, err := u.repo.ReadUserByID(ctx, userID)
	if err != nil {
		return models.User{}, err
	}

	oldAvatar := user.Avatar

	objectName := userID.String() + ext
	newURL, err := u.s3.Upload(ctx, avatarBucket, objectName, data, contentType)
	if err != nil {
		return models.User{}, fmt.Errorf("upload avatar: %w", err)
	}

	user.Avatar = newURL
	user.UpdatedAt = time.Now().UTC()

	if err := u.repo.UpdateAvatar(ctx, user); err != nil {
		_ = u.s3.Delete(ctx, avatarBucket, objectName)
		return models.User{}, fmt.Errorf("update avatar: %w", err)
	}

	if oldAvatar != "" && oldAvatar != defaultAvatar && oldAvatar != newURL {
		oldKey := extractObjectName(oldAvatar, avatarBucket)
		if oldKey != "" {
			_ = u.s3.Delete(ctx, avatarBucket, oldKey)
		}
	}

	return user, nil
}

func extractObjectName(url, bucket string) string {
	marker := "/" + bucket + "/"
	idx := strings.LastIndex(url, marker)
	if idx == -1 {
		return ""
	}
	return url[idx+len(marker):]
}

func NewUsecase(repo auth.Repo, secret string) *Usecase {
	return &Usecase{
		repo:   repo,
		s3:     s3,
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
	now := time.Now().UTC()
	accessExpiresAt := now.Add(accessTokenTTL)
	refreshExpiresAt := now.Add(refreshTokenTTL)

	accessToken, err := u.newAccessToken(userID, accessExpiresAt)
	if err != nil {
		return models.AuthTokens{}, fmt.Errorf("create access token: %w", err)
	}

	refreshToken := rand.Text()

	err = u.repo.CreateRefreshToken(ctx, models.RefreshToken{
		TokenHash: hashRefreshToken(refreshToken),
		UserID:    userID,
		ExpiresAt: refreshExpiresAt,
		CreatedAt: now,
	})
	if err != nil {
		return models.AuthTokens{}, err
	}

	return models.AuthTokens{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
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

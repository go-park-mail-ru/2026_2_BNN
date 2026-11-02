package usecase

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"time"
	"unicode/utf8"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"

	uuid "github.com/satori/go.uuid"

	"encoding/base32"
	"net/url"

	"github.com/dgryski/dgoogauth"
	"github.com/skip2/go-qrcode"
)

const (
	minLoginLength    = 3
	maxLoginLength    = 20
	minPasswordLength = 8
	maxPasswordLength = 128
	issuerName        = "BNN_Notes"
)

type Usecase struct {
	repo    auth.Repo
	storage auth.Storage
	oauth   *auth.OAuthConfig
	secret  []byte
}

var allowedAvatarTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func NewUsecase(repo auth.Repo, storage auth.Storage, oauth *auth.OAuthConfig, secret string) *Usecase {
	return &Usecase{
		repo:    repo,
		storage: storage,
		oauth:   oauth,
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

	if user.IsExternal {
		return models.User{}, models.AuthTokens{}, errors.New("user registered via oauth, please use oauth login")
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

	if err := u.repo.UpdatePassword(ctx, userID, passwordHash); err != nil {
		return models.AuthTokens{}, err
	}

	if err := u.repo.DeleteUserRefreshTokens(ctx, userID); err != nil {
		return models.AuthTokens{}, err
	}

	return u.createTokens(ctx, userID)
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
	if err := u.repo.UpdateAvatar(ctx, user); err != nil {
		_ = u.storage.DeleteFile(ctx, key)
		return models.User{}, err
	}

	if oldAvatar != "" {
		_ = u.storage.DeleteFile(ctx, path.Base(oldAvatar))
	}
	return user, nil
}

func (u *Usecase) Generate2FA(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	user, err := u.repo.ReadUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)

	if err := u.repo.UpdateTOTPSecret(ctx, userID, secret); err != nil {
		return nil, err
	}

	u2, _ := url.Parse("otpauth://totp")
	u2.Path += fmt.Sprintf("/%s:%s", url.PathEscape(issuerName), url.PathEscape(user.Login))

	params := url.Values{}
	params.Add("secret", secret)
	params.Add("issuer", issuerName)
	u2.RawQuery = params.Encode()

	png, err := qrcode.Encode(u2.String(), qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("generate qr: %w", err)
	}

	return png, nil
}

func (u *Usecase) Enable2FA(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := u.repo.ReadUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.TOTPSecret == "" {
		return models.ErrValidation
	}

	otpConfig := &dgoogauth.OTPConfig{
		Secret:      user.TOTPSecret,
		WindowSize:  3,
		HotpCounter: 0,
	}

	valid, err := otpConfig.Authenticate(code)
	if err != nil || !valid {
		return models.ErrInvalidToken
	}

	return u.repo.Set2FAStatus(ctx, userID, true)
}

func (u *Usecase) Disable2FA(ctx context.Context, userID uuid.UUID) error {
	if err := u.repo.Set2FAStatus(ctx, userID, false); err != nil {
		return err
	}
	return u.repo.UpdateTOTPSecret(ctx, userID, "")
}

func (u *Usecase) GetOAuthURL(provider string, state string) string {
	if provider == "github" {
		return u.oauth.GitHub.AuthCodeURL(state)
	}
	return ""
}

func (u *Usecase) HandleOAuthCallback(ctx context.Context, provider, code string) (models.User, models.AuthTokens, error) {
	if provider != "github" {
		return models.User{}, models.AuthTokens{}, models.ErrValidation
	}
	return u.handleGitHubCallback(ctx, code)
}

func (u *Usecase) handleGitHubCallback(ctx context.Context, code string) (models.User, models.AuthTokens, error) {
	tok, err := u.oauth.GitHub.Exchange(ctx, code)
	if err != nil {
		return models.User{}, models.AuthTokens{}, fmt.Errorf("exchange github code: %w", err)
	}

	client := u.oauth.GitHub.Client(ctx, tok)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return models.User{}, models.AuthTokens{}, fmt.Errorf("get github user: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var ghUser struct {
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	json.Unmarshal(body, &ghUser)

	email := ghUser.Email
	if email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailResp.Body.Close()
			var emails []struct {
				Email    string `json:"email"`
				Primary  bool   `json:"primary"`
				Verified bool   `json:"verified"`
			}
			emailBody, _ := io.ReadAll(emailResp.Body)
			json.Unmarshal(emailBody, &emails)
			for _, e := range emails {
				if e.Primary {
					email = e.Email
					break
				}
			}
		}
	}

	if email == "" {
		return models.User{}, models.AuthTokens{}, errors.New("no email provided by github")
	}

	user, err := u.repo.ReadUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			now := time.Now().UTC()
			user = models.User{
				ID:         uuid.NewV4(),
				Login:      email,
				Email:      email,
				Avatar:     ghUser.AvatarURL,
				IsExternal: true,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			if err := u.repo.CreateUser(ctx, user); err != nil {
				return models.User{}, models.AuthTokens{}, err
			}
		} else {
			return models.User{}, models.AuthTokens{}, err
		}
	}

	tokens, err := u.createTokens(ctx, user.ID)
	if err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	return user, tokens, nil
}

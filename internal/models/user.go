package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash []byte
	Avatar       string
	TOTPSecret   string
	Is2FAEnabled bool
	IsExternal   bool
	Email        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RefreshToken struct {
	TokenHash []byte
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

type AuthTokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time

	TwoFactorToken     string
	TwoFactorExpiresAt time.Time
}

type OAuthProfile struct {
	Email     string
	AvatarURL string
}

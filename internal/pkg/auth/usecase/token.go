package usecase

import (
	"crypto/sha256"
	"time"

	"github.com/golang-jwt/jwt/v5"
	uuid "github.com/satori/go.uuid"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

// В ID (jti) лежит id сессии: по нему Authenticate проверяет, что сессия не отозвана.
func (u *Usecase) newAccessToken(userID, sessionID uuid.UUID, expiresAt time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		ID:        sessionID.String(),
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(u.secret)
}

func (u *Usecase) parseAccessToken(token string) (uuid.UUID, uuid.UUID, error) {
	var claims jwt.RegisteredClaims

	_, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) {
		return u.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	userID, err := uuid.FromString(claims.Subject)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	sessionID, err := uuid.FromString(claims.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return userID, sessionID, nil
}

func hashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

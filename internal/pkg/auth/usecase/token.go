package usecase

import (
	"crypto/sha256"
	"time"

	"github.com/golang-jwt/jwt/v5"
	uuid "github.com/satori/go.uuid"
)

const (
	accessTokenTTL    = 15 * time.Minute
	refreshTokenTTL   = 30 * 24 * time.Hour
	twoFactorTokenTTL = 5 * time.Minute

	accessAudience    = "access"
	twoFactorAudience = "2fa"
)

func (u *Usecase) newToken(userID uuid.UUID, audience string, expiresAt time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		Audience:  jwt.ClaimStrings{audience},
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(u.secret)
}

func (u *Usecase) parseToken(token, audience string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims

	_, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) {
		return u.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithAudience(audience))
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.FromString(claims.Subject)
}

func hashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

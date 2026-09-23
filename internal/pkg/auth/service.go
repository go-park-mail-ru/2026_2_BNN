package auth

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	CookieName = "bnn_jwt"
	tokenTTL   = 12 * time.Hour
)

// Claims — то, что кладём внутрь токена.
// Никаких паролей и прочего чувствительного — токен легко декодируется.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func jwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is not set")
	}
	return []byte(secret), nil
}

// GenerateToken выпускает подписанный JWT для пользователя.
func GenerateToken(userID string) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken проверяет подпись и срок действия, возвращает claims.
func ParseToken(tokenString string) (*Claims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// setAuthCookie ставит HttpOnly-куку с токеном.
func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		HttpOnly: true,
		Secure:   false, // true на проде с HTTPS
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(tokenTTL),
		Path:     "/",
	})
}
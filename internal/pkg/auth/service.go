package auth

import (
	"bnn/internal/models"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	CookieName = "bnn_jwt"
	tokenTTL   = 48 * time.Hour
)

type Claims struct {
	UserID  string `json:"user_id"`
	Version int    `json:"version"`
	jwt.RegisteredClaims
}
type Service struct {
	secret []byte

	mu        sync.RWMutex
	users     map[string]models.User
	usersByID map[string]models.User
}

func NewService(secret string) *Service {
	return &Service{
		secret:    []byte(secret),
		users:     make(map[string]models.User),
		usersByID: make(map[string]models.User),
	}
}

func (s *Service) UserByID(id string) (models.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.usersByID[id]

	return user, ok
}

func (s *Service) GenerateToken(userID string, version int) (string, error) {
	now := time.Now().UTC()

	claims := Claims{
		UserID:  userID,
		Version: version,
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.secret)
}

func (s *Service) ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(tokenTTL),
		Path:     "/",
	})
}

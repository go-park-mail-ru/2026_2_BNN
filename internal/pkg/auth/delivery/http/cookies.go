package delivery

import (
	"context"
	"net/http"

	"bnn/internal/models"
	"bnn/internal/pkg/api"
)

const (
	accessCookieName  = "bnn_access"
	refreshCookieName = "bnn_refresh"
	refreshCookiePath = "/api/auth"
)

func setAuthCookies(ctx context.Context, tokens models.AuthTokens) {
	api.SetCookie(ctx, api.Cookie{
		Name:     accessCookieName,
		Value:    tokens.AccessToken,
		Path:     "/",
		Expires:  tokens.AccessExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	api.SetCookie(ctx, api.Cookie{
		Name:     refreshCookieName,
		Value:    tokens.RefreshToken,
		Path:     refreshCookiePath,
		Expires:  tokens.RefreshExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAuthCookies(ctx context.Context) {
	api.SetCookie(ctx, api.Cookie{
		Name:     accessCookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	api.SetCookie(ctx, api.Cookie{
		Name:     refreshCookieName,
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

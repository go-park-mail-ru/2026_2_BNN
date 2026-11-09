package delivery

import (
	"context"
	"crypto/rand"
	"net/http"

	"bnn/internal/models"
	"bnn/internal/pkg/cookies"
)

const (
	accessCookieName     = "bnn_access"
	refreshCookieName    = "bnn_refresh"
	csrfCookieName       = "bnn_csrf"
	refreshCookiePath    = "/api/auth"
	twoFactorCookieName  = "bnn_2fa"
	twoFactorCookiePath  = "/api/auth/2fa"
	oauthStateCookieName = "oauth_state"
	oauthStateCookiePath = "/api/auth/github"
	oauthStateMaxAge     = 10 * 60
)

func setAuthCookies(ctx context.Context, tokens models.AuthTokens) {
	cookies.Set(ctx, &http.Cookie{
		Name:     accessCookieName,
		Value:    tokens.AccessToken,
		Path:     "/",
		Expires:  tokens.AccessExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	cookies.Set(ctx, &http.Cookie{
		Name:     refreshCookieName,
		Value:    tokens.RefreshToken,
		Path:     refreshCookiePath,
		Expires:  tokens.RefreshExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	cookies.Set(ctx, &http.Cookie{
		Name:     csrfCookieName,
		Value:    rand.Text(),
		Path:     "/",
		Expires:  tokens.RefreshExpiresAt,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAuthCookies(ctx context.Context) {
	cookies.Set(ctx, &http.Cookie{
		Name:     accessCookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	cookies.Set(ctx, &http.Cookie{
		Name:     refreshCookieName,
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	cookies.Set(ctx, &http.Cookie{
		Name:     csrfCookieName,
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func setTwoFactorCookie(ctx context.Context, tokens models.AuthTokens) {
	cookies.Set(ctx, &http.Cookie{
		Name:     twoFactorCookieName,
		Value:    tokens.TwoFactorToken,
		Path:     twoFactorCookiePath,
		Expires:  tokens.TwoFactorExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearTwoFactorCookie(ctx context.Context) {
	cookies.Set(ctx, &http.Cookie{
		Name:     twoFactorCookieName,
		Path:     twoFactorCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func setOAuthStateCookie(ctx context.Context, state string) {
	cookies.Set(ctx, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     oauthStateCookiePath,
		MaxAge:   oauthStateMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthStateCookie(ctx context.Context) {
	cookies.Set(ctx, &http.Cookie{
		Name:     oauthStateCookieName,
		Path:     oauthStateCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

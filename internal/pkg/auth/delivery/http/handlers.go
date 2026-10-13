package delivery

import (
	"log/slog"
	"net/http"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/httpjson"
)

const (
	maxRequestBodySize = 4096

	accessCookieName  = "bnn_access"
	refreshCookieName = "bnn_refresh"
	refreshCookiePath = "/api/auth"
)

type Handler struct {
	uc auth.Usecase
}

func NewHandler(uc auth.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req models.SignUpRequest
	if err := httpjson.Read(w, r, &req, maxRequestBodySize); err != nil {
		slog.Warn("Failed to read signup request", "error", err)
		w.WriteHeader(httpjson.ReadErrorStatus(err))
		return
	}

	user, tokens, err := h.uc.SignUp(r.Context(), req.Login, req.Password)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	setAuthCookies(w, tokens)
	httpjson.Write(w, http.StatusCreated, user)
}

func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req models.SignInRequest
	if err := httpjson.Read(w, r, &req, maxRequestBodySize); err != nil {
		slog.Warn("Failed to read sign in request", "error", err)
		w.WriteHeader(httpjson.ReadErrorStatus(err))
		return
	}

	user, tokens, err := h.uc.SignIn(r.Context(), req.Login, req.Password)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	setAuthCookies(w, tokens)
	httpjson.Write(w, http.StatusOK, user)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		slog.Warn("Refresh cookie missing")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokens, err := h.uc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	setAuthCookies(w, tokens)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		if err := h.uc.Logout(r.Context(), cookie.Value); err != nil {
			httpjson.WriteError(w, err)
			return
		}
	}

	clearAuthCookies(w)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxutil.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := h.uc.GetUser(r.Context(), userID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, user)
}

func setAuthCookies(w http.ResponseWriter, tokens models.AuthTokens) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    tokens.AccessToken,
		Path:     "/",
		Expires:  tokens.AccessExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    tokens.RefreshToken,
		Path:     refreshCookiePath,
		Expires:  tokens.RefreshExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

package delivery

import (
	"context"
	"errors"
	"log/slog"

	"bnn/internal/models"
	"bnn/internal/pkg/api"
	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type Handler struct {
	uc auth.Usecase
}

func NewHandler(uc auth.Usecase) *Handler {
	return &Handler{uc: uc}
}

// --- SignUp ---

func (h *Handler) SignUp(ctx context.Context, req api.SignUpRequestObject) (api.SignUpResponseObject, error) {
	user, tokens, err := h.uc.SignUp(ctx, req.Body.Login, req.Body.Password)
	if errors.Is(err, models.ErrValidation) {
		return api.SignUp400Response{}, nil
	}
	if errors.Is(err, models.ErrUserExists) {
		return api.SignUp409Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	setAuthCookies(ctx, tokens)

	return toSignUpResponse(user), nil
}

// --- SignIn ---

func (h *Handler) SignIn(ctx context.Context, req api.SignInRequestObject) (api.SignInResponseObject, error) {
	user, tokens, err := h.uc.SignIn(ctx, req.Body.Login, req.Body.Password)
	if errors.Is(err, models.ErrInvalidCredentials) {
		return api.SignIn401Response{}, nil
	}
	if errors.Is(err, models.ErrValidation) {
		return api.SignIn400Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	slog.Info("SignIn: setting cookies", "store_found", api.GetCookieStore(ctx) != nil)

	setAuthCookies(ctx, tokens)

	return toSignInResponse(user), nil
}

// --- Refresh ---

func (h *Handler) Refresh(ctx context.Context, req api.RefreshRequestObject) (api.RefreshResponseObject, error) {
	cookie, err := api.GetCookieValue(ctx, refreshCookieName)
	if err != nil {
		return api.Refresh401Response{}, nil
	}

	tokens, err := h.uc.Refresh(ctx, cookie)
	if errors.Is(err, models.ErrInvalidToken) {
		return api.Refresh401Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	setAuthCookies(ctx, tokens)

	return api.Refresh200Response{}, nil
}

// --- Logout ---

func (h *Handler) Logout(ctx context.Context, req api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	if cookie, err := api.GetCookieValue(ctx, refreshCookieName); err == nil {
		if err := h.uc.Logout(ctx, cookie); err != nil {
			return nil, err
		}
	}

	clearAuthCookies(ctx)

	return api.Logout200Response{}, nil
}

// --- GetCurrentUser ---

func (h *Handler) GetCurrentUser(ctx context.Context, req api.GetCurrentUserRequestObject) (api.GetCurrentUserResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.GetCurrentUser401Response{}, nil
	}

	user, err := h.uc.GetUser(ctx, userID)
	if errors.Is(err, models.ErrUserNotFound) {
		return api.GetCurrentUser404Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return toGetCurrentUserResponse(user), nil
}

// --- UpdateAvatar (заглушка, реализуем позже) ---

func (h *Handler) UpdateAvatar(ctx context.Context, req api.UpdateAvatarRequestObject) (api.UpdateAvatarResponseObject, error) {
	return api.UpdateAvatar400Response{}, nil
}

func toSignUpResponse(u models.User) api.SignUp201JSONResponse {
	return api.SignUp201JSONResponse{
		Body: toAPIUser(u),
	}
}

func toSignInResponse(u models.User) api.SignIn200JSONResponse {
	return api.SignIn200JSONResponse{
		Body: toAPIUser(u),
	}
}

func toGetCurrentUserResponse(u models.User) api.GetCurrentUser200JSONResponse {
	return api.GetCurrentUser200JSONResponse(toAPIUser(u))
}

func toAPIUser(u models.User) api.User {
	return api.User{
		Id:        openapi_types.UUID(u.ID),
		Login:     u.Login,
		Avatar:    u.Avatar,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

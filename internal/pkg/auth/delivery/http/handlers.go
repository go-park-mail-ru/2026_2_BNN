package delivery

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/auth/delivery/http/api"
	"bnn/internal/pkg/cookies"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/upload"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	avatarField      = "avatar"
	maxAvatarSize5MB = 5 << 20
)

type Handler struct {
	uc auth.Usecase
}

func NewHandler(uc auth.Usecase) *Handler {
	return &Handler{uc: uc}
}

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

	setAuthCookies(ctx, tokens)

	return toSignInResponse(user), nil
}

func (h *Handler) Refresh(ctx context.Context, req api.RefreshRequestObject) (api.RefreshResponseObject, error) {
	cookie, err := cookies.Get(ctx, refreshCookieName)
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

func (h *Handler) Logout(ctx context.Context, req api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	if cookie, err := cookies.Get(ctx, refreshCookieName); err == nil {
		if err := h.uc.Logout(ctx, cookie); err != nil {
			return nil, err
		}
	}

	clearAuthCookies(ctx)

	return api.Logout200Response{}, nil
}

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

func (h *Handler) ChangePassword(ctx context.Context, req api.ChangePasswordRequestObject) (api.ChangePasswordResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.ChangePassword401Response{}, nil
	}

	tokens, err := h.uc.ChangePassword(ctx, userID, req.Body.OldPassword, req.Body.NewPassword)
	if errors.Is(err, models.ErrValidation) {
		return api.ChangePassword400Response{}, nil
	}
	if errors.Is(err, models.ErrWrongPassword) {
		return api.ChangePassword403Response{}, nil
	}
	if errors.Is(err, models.ErrUserNotFound) {
		return api.ChangePassword401Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	setAuthCookies(ctx, tokens)

	return api.ChangePassword204Response{}, nil
}

func (h *Handler) UpdateAvatar(ctx context.Context, req api.UpdateAvatarRequestObject) (api.UpdateAvatarResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.UpdateAvatar401Response{}, nil
	}

	data, err := upload.ReadFile(req.Body, avatarField, maxAvatarSize5MB)
	if errors.Is(err, upload.ErrFileTooLarge) || errors.As(err, new(*http.MaxBytesError)) {
		return api.UpdateAvatar413Response{}, nil
	}
	if err != nil {
		return api.UpdateAvatar400Response{}, nil
	}

	user, err := h.uc.UpdateAvatar(ctx, userID, data, http.DetectContentType(data))
	if errors.Is(err, models.ErrUnsupportedFileType) {
		return api.UpdateAvatar415Response{}, nil
	}
	if err != nil {
		return nil, err
	}
	return toUpdateAvatarResponse(user), nil
}

func toUpdateAvatarResponse(u models.User) api.UpdateAvatar200JSONResponse {
	return api.UpdateAvatar200JSONResponse(toAPIUser(u))
}

func toSignUpResponse(u models.User) api.SignUp201JSONResponse {
	return api.SignUp201JSONResponse(toAPIUser(u))
}

func toSignInResponse(u models.User) api.SignIn200JSONResponse {
	return api.SignIn200JSONResponse(toAPIUser(u))
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

func (h *Handler) Generate2FA(ctx context.Context, req api.Generate2FARequestObject) (api.Generate2FAResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.Generate2FA401Response{}, nil
	}

	png, err := h.uc.Generate2FA(ctx, userID)
	if err != nil {
		return nil, err
	}

	return api.Generate2FA200ImagepngResponse{
		Body:          bytes.NewReader(png),
		ContentLength: int64(len(png)),
	}, nil
}

func (h *Handler) Enable2FA(ctx context.Context, req api.Enable2FARequestObject) (api.Enable2FAResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.Enable2FA401Response{}, nil
	}

	err := h.uc.Enable2FA(ctx, userID, req.Body.Code)
	if errors.Is(err, models.ErrInvalidToken) || errors.Is(err, models.ErrValidation) {
		return api.Enable2FA401Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.Enable2FA200Response{}, nil
}

func (h *Handler) Disable2FA(ctx context.Context, req api.Disable2FARequestObject) (api.Disable2FAResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.Disable2FA401Response{}, nil
	}

	if err := h.uc.Disable2FA(ctx, userID); err != nil {
		return nil, err
	}

	return api.Disable2FA200Response{}, nil
}

func (h *Handler) GitHubLogin(ctx context.Context, req api.GitHubLoginRequestObject) (api.GitHubLoginResponseObject, error) {
	return nil, errors.New("use /api/auth/github/login router directly")
}

func (h *Handler) GitHubCallback(ctx context.Context, req api.GitHubCallbackRequestObject) (api.GitHubCallbackResponseObject, error) {
	return nil, errors.New("use /api/auth/github/callback router directly")
}

func (h *Handler) GetOAuthURL(provider string, state string) string {
	return h.uc.GetOAuthURL(provider, state)
}

func (h *Handler) HandleOAuthCallback(ctx context.Context, provider, code string) (models.User, models.AuthTokens, error) {
	return h.uc.HandleOAuthCallback(ctx, provider, code)
}

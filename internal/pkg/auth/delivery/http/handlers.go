package delivery

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/auth/delivery/http/api"
	"bnn/internal/pkg/cookies"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/upload"

	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	avatarField      = "avatar"
	maxAvatarSize5MB = 5 << 20
	githubProvider   = "github"
	profilePage      = "profile"
	twoFactorPage    = "login/2fa"
)

type Handler struct {
	uc          auth.Usecase
	frontendURL string
}

func NewHandler(uc auth.Usecase, frontendURL string) *Handler {
	return &Handler{uc: uc, frontendURL: frontendURL}
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
	if errors.Is(err, models.ErrOAuthAccount) {
		return api.SignIn403Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	if tokens.TwoFactorToken != "" {
		setTwoFactorCookie(ctx, tokens)
		return api.SignIn202Response{}, nil
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
	if errors.Is(err, models.ErrValidation) || errors.Is(err, models.ErrOAuthAccount) {
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
		Id:            openapi_types.UUID(u.ID),
		Login:         u.Login,
		Avatar:        u.Avatar,
		Is2faEnabled:  u.Is2FAEnabled,
		OauthProvider: lo.Ternary(u.IsExternal, lo.ToPtr(api.Github), nil),
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}
func (h *Handler) Generate2FA(ctx context.Context, req api.Generate2FARequestObject) (api.Generate2FAResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.Generate2FA401Response{}, nil
	}

	png, err := h.uc.Generate2FA(ctx, userID)
	if errors.Is(err, models.ErrTwoFactorEnabled) {
		return api.Generate2FA409Response{}, nil
	}
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
	if errors.Is(err, models.ErrValidation) {
		return api.Enable2FA400Response{}, nil
	}
	if errors.Is(err, models.ErrInvalidCode) {
		return api.Enable2FA403Response{}, nil
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

	err := h.uc.Disable2FA(ctx, userID, req.Body.Code)
	if errors.Is(err, models.ErrInvalidCode) {
		return api.Disable2FA403Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.Disable2FA200Response{}, nil
}

func (h *Handler) Verify2FA(ctx context.Context, req api.Verify2FARequestObject) (api.Verify2FAResponseObject, error) {
	token, err := cookies.Get(ctx, twoFactorCookieName)
	if err != nil {
		return api.Verify2FA401Response{}, nil
	}

	user, tokens, err := h.uc.VerifyTwoFactor(ctx, token, req.Body.Code)
	if errors.Is(err, models.ErrInvalidToken) {
		return api.Verify2FA401Response{}, nil
	}
	if errors.Is(err, models.ErrInvalidCode) {
		return api.Verify2FA403Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	clearTwoFactorCookie(ctx)
	setAuthCookies(ctx, tokens)

	return api.Verify2FA200JSONResponse(toAPIUser(user)), nil
}

func (h *Handler) GitHubLogin(ctx context.Context, req api.GitHubLoginRequestObject) (api.GitHubLoginResponseObject, error) {
	state := uuid.NewV4().String()
	setOAuthStateCookie(ctx, state)

	location := h.uc.GetOAuthURL(githubProvider, state)
	return api.GitHubLogin307Response{Headers: api.GitHubLogin307ResponseHeaders{Location: location}}, nil
}

func (h *Handler) GitHubCallback(ctx context.Context, req api.GitHubCallbackRequestObject) (api.GitHubCallbackResponseObject, error) {
	state, err := cookies.Get(ctx, oauthStateCookieName)
	if err != nil || state != req.Params.State {
		return api.GitHubCallback400Response{}, nil
	}

	_, tokens, err := h.uc.HandleOAuthCallback(ctx, githubProvider, req.Params.Code)
	if err != nil {
		return nil, err
	}

	clearOAuthStateCookie(ctx)

	page := profilePage
	if tokens.TwoFactorToken != "" {
		setTwoFactorCookie(ctx, tokens)
		page = twoFactorPage
	} else {
		setAuthCookies(ctx, tokens)
	}

	location, err := url.JoinPath(h.frontendURL, page)
	if err != nil {
		return nil, err
	}

	return api.GitHubCallback307Response{Headers: api.GitHubCallback307ResponseHeaders{Location: location}}, nil
}

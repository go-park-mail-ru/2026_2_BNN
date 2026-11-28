package grpc

import (
	"context"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/auth/delivery/grpc/gen"
	"bnn/internal/pkg/auth/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandler(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewV4()
	user := models.User{ID: userID, Login: "seyran"}
	tokens := models.AuthTokens{AccessToken: "access", RefreshToken: "refresh"}
	creds := &gen.Credentials{Login: "seyran", Password: "password"}
	userReq := &gen.UserRequest{UserId: userID.String()}
	badUserReq := &gen.UserRequest{UserId: "bad"}
	codeReq := &gen.TwoFactorCodeRequest{UserId: userID.String(), Code: "123456"}

	tests := []struct {
		name    string
		prepare func(uc *mocks.MockUsecase)
		call    func(h *Handler) error
		code    codes.Code
	}{
		{
			name: "sign up ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().SignUp(gomock.Any(), "seyran", "password").Return(user, tokens, nil)
			},
			call: func(h *Handler) error {
				_, err := h.SignUp(ctx, creds)
				return err
			},
		},
		{
			name: "sign up exists",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().SignUp(gomock.Any(), "seyran", "password").Return(models.User{}, models.AuthTokens{}, models.ErrUserExists)
			},
			call: func(h *Handler) error {
				_, err := h.SignUp(ctx, creds)
				return err
			},
			code: codes.AlreadyExists,
		},
		{
			name: "sign in ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().SignIn(gomock.Any(), "seyran", "password").Return(user, tokens, nil)
			},
			call: func(h *Handler) error {
				_, err := h.SignIn(ctx, creds)
				return err
			},
		},
		{
			name: "sign in wrong credentials",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().SignIn(gomock.Any(), "seyran", "password").Return(models.User{}, models.AuthTokens{}, models.ErrInvalidCredentials)
			},
			call: func(h *Handler) error {
				_, err := h.SignIn(ctx, creds)
				return err
			},
			code: codes.Unauthenticated,
		},
		{
			name: "verify two factor ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().VerifyTwoFactor(gomock.Any(), "token", "123456").Return(user, tokens, nil)
			},
			call: func(h *Handler) error {
				_, err := h.VerifyTwoFactor(ctx, &gen.VerifyTwoFactorRequest{TwoFactorToken: "token", Code: "123456"})
				return err
			},
		},
		{
			name: "verify two factor wrong code",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().VerifyTwoFactor(gomock.Any(), "token", "000000").Return(models.User{}, models.AuthTokens{}, models.ErrInvalidCode)
			},
			call: func(h *Handler) error {
				_, err := h.VerifyTwoFactor(ctx, &gen.VerifyTwoFactorRequest{TwoFactorToken: "token", Code: "000000"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "refresh ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Refresh(gomock.Any(), "refresh").Return(tokens, nil)
			},
			call: func(h *Handler) error {
				_, err := h.Refresh(ctx, &gen.RefreshRequest{RefreshToken: "refresh"})
				return err
			},
		},
		{
			name: "refresh invalid",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Refresh(gomock.Any(), "refresh").Return(models.AuthTokens{}, models.ErrInvalidToken)
			},
			call: func(h *Handler) error {
				_, err := h.Refresh(ctx, &gen.RefreshRequest{RefreshToken: "refresh"})
				return err
			},
			code: codes.Unauthenticated,
		},
		{
			name: "logout ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Logout(gomock.Any(), "refresh").Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.Logout(ctx, &gen.RefreshRequest{RefreshToken: "refresh"})
				return err
			},
		},
		{
			name: "logout internal",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Logout(gomock.Any(), "refresh").Return(assert.AnError)
			},
			call: func(h *Handler) error {
				_, err := h.Logout(ctx, &gen.RefreshRequest{RefreshToken: "refresh"})
				return err
			},
			code: codes.Internal,
		},
		{
			name: "authenticate ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Authenticate(gomock.Any(), "access").Return(userID, nil)
			},
			call: func(h *Handler) error {
				_, err := h.Authenticate(ctx, &gen.AuthenticateRequest{AccessToken: "access"})
				return err
			},
		},
		{
			name: "authenticate invalid",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Authenticate(gomock.Any(), "access").Return(uuid.Nil, models.ErrInvalidToken)
			},
			call: func(h *Handler) error {
				_, err := h.Authenticate(ctx, &gen.AuthenticateRequest{AccessToken: "access"})
				return err
			},
			code: codes.Unauthenticated,
		},
		{
			name: "get user ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetUser(gomock.Any(), userID).Return(user, nil)
			},
			call: func(h *Handler) error {
				_, err := h.GetUser(ctx, userReq)
				return err
			},
		},
		{
			name:    "get user bad id",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.GetUser(ctx, badUserReq)
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "get user not found",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetUser(gomock.Any(), userID).Return(models.User{}, models.ErrUserNotFound)
			},
			call: func(h *Handler) error {
				_, err := h.GetUser(ctx, userReq)
				return err
			},
			code: codes.NotFound,
		},
		{
			name: "change password ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().ChangePassword(gomock.Any(), userID, "old", "new").Return(tokens, nil)
			},
			call: func(h *Handler) error {
				_, err := h.ChangePassword(ctx, &gen.ChangePasswordRequest{UserId: userID.String(), OldPassword: "old", NewPassword: "new"})
				return err
			},
		},
		{
			name:    "change password bad id",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.ChangePassword(ctx, &gen.ChangePasswordRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "change password wrong",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().ChangePassword(gomock.Any(), userID, "old", "new").Return(models.AuthTokens{}, models.ErrWrongPassword)
			},
			call: func(h *Handler) error {
				_, err := h.ChangePassword(ctx, &gen.ChangePasswordRequest{UserId: userID.String(), OldPassword: "old", NewPassword: "new"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "update avatar ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateAvatar(gomock.Any(), userID, []byte("png"), "image/png").Return(user, nil)
			},
			call: func(h *Handler) error {
				_, err := h.UpdateAvatar(ctx, &gen.UpdateAvatarRequest{UserId: userID.String(), Data: []byte("png"), ContentType: "image/png"})
				return err
			},
		},
		{
			name:    "update avatar bad id",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.UpdateAvatar(ctx, &gen.UpdateAvatarRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "update avatar unsupported",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateAvatar(gomock.Any(), userID, []byte("gif"), "image/gif").Return(models.User{}, models.ErrUnsupportedFileType)
			},
			call: func(h *Handler) error {
				_, err := h.UpdateAvatar(ctx, &gen.UpdateAvatarRequest{UserId: userID.String(), Data: []byte("gif"), ContentType: "image/gif"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "generate 2fa ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Generate2FA(gomock.Any(), userID).Return([]byte("qr"), nil)
			},
			call: func(h *Handler) error {
				_, err := h.Generate2FA(ctx, userReq)
				return err
			},
		},
		{
			name:    "generate 2fa bad id",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.Generate2FA(ctx, badUserReq)
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "generate 2fa enabled",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Generate2FA(gomock.Any(), userID).Return(nil, models.ErrTwoFactorEnabled)
			},
			call: func(h *Handler) error {
				_, err := h.Generate2FA(ctx, userReq)
				return err
			},
			code: codes.FailedPrecondition,
		},
		{
			name: "enable 2fa ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Enable2FA(gomock.Any(), userID, "123456").Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.Enable2FA(ctx, codeReq)
				return err
			},
		},
		{
			name:    "enable 2fa bad id",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.Enable2FA(ctx, &gen.TwoFactorCodeRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "enable 2fa wrong code",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Enable2FA(gomock.Any(), userID, "123456").Return(models.ErrInvalidCode)
			},
			call: func(h *Handler) error {
				_, err := h.Enable2FA(ctx, codeReq)
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "disable 2fa ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Disable2FA(gomock.Any(), userID, "123456").Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.Disable2FA(ctx, codeReq)
				return err
			},
		},
		{
			name:    "disable 2fa bad id",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.Disable2FA(ctx, &gen.TwoFactorCodeRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "disable 2fa wrong code",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Disable2FA(gomock.Any(), userID, "123456").Return(models.ErrInvalidCode)
			},
			call: func(h *Handler) error {
				_, err := h.Disable2FA(ctx, codeReq)
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "oauth url ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetOAuthURL(gomock.Any(), "github", "state").Return("https://github.com/login", nil)
			},
			call: func(h *Handler) error {
				_, err := h.GetOAuthURL(ctx, &gen.OAuthURLRequest{Provider: "github", State: "state"})
				return err
			},
		},
		{
			name: "oauth url bad provider",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetOAuthURL(gomock.Any(), "gitlab", "state").Return("", models.ErrValidation)
			},
			call: func(h *Handler) error {
				_, err := h.GetOAuthURL(ctx, &gen.OAuthURLRequest{Provider: "gitlab", State: "state"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "oauth callback ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().HandleOAuthCallback(gomock.Any(), "github", "code").Return(user, tokens, nil)
			},
			call: func(h *Handler) error {
				_, err := h.HandleOAuthCallback(ctx, &gen.OAuthCallbackRequest{Provider: "github", Code: "code"})
				return err
			},
		},
		{
			name: "oauth callback internal",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().HandleOAuthCallback(gomock.Any(), "github", "code").Return(models.User{}, models.AuthTokens{}, assert.AnError)
			},
			call: func(h *Handler) error {
				_, err := h.HandleOAuthCallback(ctx, &gen.OAuthCallbackRequest{Provider: "github", Code: "code"})
				return err
			},
			code: codes.Internal,
		},
		{
			name: "user exists",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetUser(gomock.Any(), userID).Return(user, nil)
			},
			call: func(h *Handler) error {
				_, err := h.CheckUserExists(ctx, userReq)
				return err
			},
		},
		{
			name: "user does not exist",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetUser(gomock.Any(), userID).Return(models.User{}, models.ErrUserNotFound)
			},
			call: func(h *Handler) error {
				resp, err := h.CheckUserExists(ctx, userReq)
				assert.False(t, resp.GetExists())
				return err
			},
		},
		{
			name:    "user exists bad id",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.CheckUserExists(ctx, badUserReq)
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "set admin ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().SetAdmin(gomock.Any(), userID, true).Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.SetAdmin(ctx, &gen.SetAdminRequest{UserId: userID.String(), IsAdmin: true})
				return err
			},
		},
		{
			name: "set admin user not found",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().SetAdmin(gomock.Any(), userID, false).Return(models.ErrUserNotFound)
			},
			call: func(h *Handler) error {
				_, err := h.SetAdmin(ctx, &gen.SetAdminRequest{UserId: userID.String(), IsAdmin: false})
				return err
			},
			code: codes.NotFound,
		},
		{
			name:    "set admin bad id",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.SetAdmin(ctx, &gen.SetAdminRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "user exists internal",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetUser(gomock.Any(), userID).Return(models.User{}, assert.AnError)
			},
			call: func(h *Handler) error {
				_, err := h.CheckUserExists(ctx, userReq)
				return err
			},
			code: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			err := tt.call(NewHandler(uc))
			assert.Equal(t, tt.code, status.Code(err))
		})
	}
}

package grpc

import (
	"context"

	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/auth/delivery/grpc/gen"
	"bnn/internal/pkg/grpcconv"
	"bnn/internal/pkg/grpcerr"

	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	gen.UnimplementedAuthServer
	uc auth.Usecase
}

func NewHandler(uc auth.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) SignUp(ctx context.Context, in *gen.Credentials) (*gen.Session, error) {
	user, tokens, err := h.uc.SignUp(ctx, in.GetLogin(), in.GetPassword())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return SessionToProto(user, tokens), nil
}

func (h *Handler) SignIn(ctx context.Context, in *gen.Credentials) (*gen.Session, error) {
	user, tokens, err := h.uc.SignIn(ctx, in.GetLogin(), in.GetPassword())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return SessionToProto(user, tokens), nil
}

func (h *Handler) VerifyTwoFactor(ctx context.Context, in *gen.VerifyTwoFactorRequest) (*gen.Session, error) {
	user, tokens, err := h.uc.VerifyTwoFactor(ctx, in.GetTwoFactorToken(), in.GetCode())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return SessionToProto(user, tokens), nil
}

func (h *Handler) Refresh(ctx context.Context, in *gen.RefreshRequest) (*gen.Tokens, error) {
	tokens, err := h.uc.Refresh(ctx, in.GetRefreshToken())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return TokensToProto(tokens), nil
}

func (h *Handler) Logout(ctx context.Context, in *gen.RefreshRequest) (*emptypb.Empty, error) {
	if err := h.uc.Logout(ctx, in.GetRefreshToken()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) Authenticate(ctx context.Context, in *gen.AuthenticateRequest) (*gen.AuthenticateResponse, error) {
	userID, err := h.uc.Authenticate(ctx, in.GetAccessToken())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &gen.AuthenticateResponse{UserId: userID.String()}, nil
}

func (h *Handler) GetUser(ctx context.Context, in *gen.UserRequest) (*gen.User, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	user, err := h.uc.GetUser(ctx, userID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return UserToProto(user), nil
}

func (h *Handler) ChangePassword(ctx context.Context, in *gen.ChangePasswordRequest) (*gen.Tokens, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	tokens, err := h.uc.ChangePassword(ctx, userID, in.GetOldPassword(), in.GetNewPassword())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return TokensToProto(tokens), nil
}

func (h *Handler) UpdateAvatar(ctx context.Context, in *gen.UpdateAvatarRequest) (*gen.User, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	user, err := h.uc.UpdateAvatar(ctx, userID, in.GetData(), in.GetContentType())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return UserToProto(user), nil
}

func (h *Handler) Generate2FA(ctx context.Context, in *gen.UserRequest) (*gen.Generate2FAResponse, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	png, err := h.uc.Generate2FA(ctx, userID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &gen.Generate2FAResponse{QrPng: png}, nil
}

func (h *Handler) Enable2FA(ctx context.Context, in *gen.TwoFactorCodeRequest) (*emptypb.Empty, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	if err := h.uc.Enable2FA(ctx, userID, in.GetCode()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) Disable2FA(ctx context.Context, in *gen.TwoFactorCodeRequest) (*emptypb.Empty, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	if err := h.uc.Disable2FA(ctx, userID, in.GetCode()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) GetOAuthURL(ctx context.Context, in *gen.OAuthURLRequest) (*gen.OAuthURLResponse, error) {
	url, err := h.uc.GetOAuthURL(ctx, in.GetProvider(), in.GetState())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &gen.OAuthURLResponse{Url: url}, nil
}

func (h *Handler) HandleOAuthCallback(ctx context.Context, in *gen.OAuthCallbackRequest) (*gen.Session, error) {
	user, tokens, err := h.uc.HandleOAuthCallback(ctx, in.GetProvider(), in.GetCode())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return SessionToProto(user, tokens), nil
}

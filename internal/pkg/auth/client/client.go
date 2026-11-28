package client

import (
	"context"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/auth/delivery/grpc/convert"
	"bnn/internal/pkg/auth/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"
	"bnn/internal/pkg/utils/grpcerr"

	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ auth.Usecase = (*Client)(nil)

type Client struct {
	client gen.AuthClient
}

func NewClient(conn grpc.ClientConnInterface) *Client {
	return &Client{client: gen.NewAuthClient(conn)}
}

func (c *Client) SignUp(ctx context.Context, login, password string) (models.User, models.AuthTokens, error) {
	resp, err := c.client.SignUp(ctx, &gen.Credentials{Login: login, Password: password})
	if err != nil {
		return models.User{}, models.AuthTokens{}, grpcerr.FromStatus(err)
	}

	return convert.SessionFromProto(resp)
}

func (c *Client) SignIn(ctx context.Context, login, password string) (models.User, models.AuthTokens, error) {
	resp, err := c.client.SignIn(ctx, &gen.Credentials{Login: login, Password: password})
	if err != nil {
		return models.User{}, models.AuthTokens{}, grpcerr.FromStatus(err)
	}

	return convert.SessionFromProto(resp)
}

func (c *Client) VerifyTwoFactor(ctx context.Context, twoFactorToken, code string) (models.User, models.AuthTokens, error) {
	resp, err := c.client.VerifyTwoFactor(ctx, &gen.VerifyTwoFactorRequest{TwoFactorToken: twoFactorToken, Code: code})
	if err != nil {
		return models.User{}, models.AuthTokens{}, grpcerr.FromStatus(err)
	}

	return convert.SessionFromProto(resp)
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (models.AuthTokens, error) {
	resp, err := c.client.Refresh(ctx, &gen.RefreshRequest{RefreshToken: refreshToken})
	if err != nil {
		return models.AuthTokens{}, grpcerr.FromStatus(err)
	}

	return convert.TokensFromProto(resp), nil
}

func (c *Client) Logout(ctx context.Context, refreshToken string) error {
	_, err := c.client.Logout(ctx, &gen.RefreshRequest{RefreshToken: refreshToken})
	return grpcerr.FromStatus(err)
}

func (c *Client) Authenticate(ctx context.Context, accessToken string) (uuid.UUID, error) {
	resp, err := c.client.Authenticate(ctx, &gen.AuthenticateRequest{AccessToken: accessToken})
	if err != nil {
		return uuid.Nil, grpcerr.FromStatus(err)
	}

	return grpcconv.ParseUUID(resp.GetUserId())
}

func (c *Client) GetUser(ctx context.Context, userID uuid.UUID) (models.User, error) {
	resp, err := c.client.GetUser(ctx, &gen.UserRequest{UserId: userID.String()})
	if err != nil {
		return models.User{}, grpcerr.FromStatus(err)
	}

	return convert.UserFromProto(resp)
}

func (c *Client) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (models.AuthTokens, error) {
	resp, err := c.client.ChangePassword(ctx, &gen.ChangePasswordRequest{
		UserId:      userID.String(),
		OldPassword: oldPassword,
		NewPassword: newPassword,
	})
	if err != nil {
		return models.AuthTokens{}, grpcerr.FromStatus(err)
	}

	return convert.TokensFromProto(resp), nil
}

func (c *Client) UpdateAvatar(ctx context.Context, userID uuid.UUID, data []byte, contentType string) (models.User, error) {
	resp, err := c.client.UpdateAvatar(ctx, &gen.UpdateAvatarRequest{
		UserId:      userID.String(),
		Data:        data,
		ContentType: contentType,
	})
	if err != nil {
		return models.User{}, grpcerr.FromStatus(err)
	}

	return convert.UserFromProto(resp)
}

func (c *Client) Generate2FA(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	resp, err := c.client.Generate2FA(ctx, &gen.UserRequest{UserId: userID.String()})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}

	return resp.GetQrPng(), nil
}

func (c *Client) Enable2FA(ctx context.Context, userID uuid.UUID, code string) error {
	_, err := c.client.Enable2FA(ctx, &gen.TwoFactorCodeRequest{UserId: userID.String(), Code: code})
	return grpcerr.FromStatus(err)
}

func (c *Client) Disable2FA(ctx context.Context, userID uuid.UUID, code string) error {
	_, err := c.client.Disable2FA(ctx, &gen.TwoFactorCodeRequest{UserId: userID.String(), Code: code})
	return grpcerr.FromStatus(err)
}

func (c *Client) GetOAuthURL(ctx context.Context, provider, state string) (string, error) {
	resp, err := c.client.GetOAuthURL(ctx, &gen.OAuthURLRequest{Provider: provider, State: state})
	if err != nil {
		return "", grpcerr.FromStatus(err)
	}

	return resp.GetUrl(), nil
}

func (c *Client) HandleOAuthCallback(ctx context.Context, provider, code string) (models.User, models.AuthTokens, error) {
	resp, err := c.client.HandleOAuthCallback(ctx, &gen.OAuthCallbackRequest{Provider: provider, Code: code})
	if err != nil {
		return models.User{}, models.AuthTokens{}, grpcerr.FromStatus(err)
	}

	return convert.SessionFromProto(resp)
}

func (c *Client) CheckUserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	resp, err := c.client.CheckUserExists(ctx, &gen.UserRequest{
		UserId: userID.String(),
	})
	if err != nil {
		return false, grpcerr.FromStatus(err)
	}
	return resp.Exists, nil
}

func (c *Client) GetUserByLogin(ctx context.Context, login string) (uuid.UUID, error) {
	resp, err := c.client.GetUserByLogin(ctx, &gen.GetUserByLoginRequest{
		Login: login,
	})
	if err != nil {
		return uuid.Nil, grpcerr.FromStatus(err)
	}

	userID, err := uuid.FromString(resp.Id)
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (c *Client) ListAdmins(ctx context.Context) ([]models.User, error) {
	resp, err := c.client.ListAdmins(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}

	return convert.UsersFromProto(resp.GetUsers())
}

func (c *Client) SearchUsers(ctx context.Context, query string) ([]models.User, error) {
	resp, err := c.client.SearchUsers(ctx, &gen.SearchUsersRequest{Query: query})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}

	return convert.UsersFromProto(resp.GetUsers())
}

func (c *Client) SetAdmin(ctx context.Context, userID uuid.UUID, isAdmin bool) error {
	_, err := c.client.SetAdmin(ctx, &gen.SetAdminRequest{UserId: userID.String(), IsAdmin: isAdmin})
	if err != nil {
		return grpcerr.FromStatus(err)
	}
	return nil
}

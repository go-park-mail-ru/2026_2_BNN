package convert

import (
	"bnn/internal/models"
	"bnn/internal/pkg/auth/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"
)

func UserToProto(u models.User) *gen.User {
	return &gen.User{
		Id:            u.ID.String(),
		Login:         u.Login,
		Avatar:        u.Avatar,
		Email:         u.Email,
		IsExternal:    u.IsExternal,
		Is_2FaEnabled: u.Is2FAEnabled,
		CreatedAt:     grpcconv.TimeToProto(u.CreatedAt),
		UpdatedAt:     grpcconv.TimeToProto(u.UpdatedAt),
	}
}

func UserFromProto(u *gen.User) (models.User, error) {
	id, err := grpcconv.ParseUUID(u.GetId())
	if err != nil {
		return models.User{}, err
	}

	return models.User{
		ID:           id,
		Login:        u.GetLogin(),
		Avatar:       u.GetAvatar(),
		Email:        u.GetEmail(),
		IsExternal:   u.GetIsExternal(),
		Is2FAEnabled: u.GetIs_2FaEnabled(),
		CreatedAt:    grpcconv.TimeFromProto(u.GetCreatedAt()),
		UpdatedAt:    grpcconv.TimeFromProto(u.GetUpdatedAt()),
	}, nil
}

func TokensToProto(t models.AuthTokens) *gen.Tokens {
	return &gen.Tokens{
		AccessToken:        t.AccessToken,
		AccessExpiresAt:    grpcconv.TimeToProto(t.AccessExpiresAt),
		RefreshToken:       t.RefreshToken,
		RefreshExpiresAt:   grpcconv.TimeToProto(t.RefreshExpiresAt),
		TwoFactorToken:     t.TwoFactorToken,
		TwoFactorExpiresAt: grpcconv.TimeToProto(t.TwoFactorExpiresAt),
	}
}

func TokensFromProto(t *gen.Tokens) models.AuthTokens {
	return models.AuthTokens{
		AccessToken:        t.GetAccessToken(),
		AccessExpiresAt:    grpcconv.TimeFromProto(t.GetAccessExpiresAt()),
		RefreshToken:       t.GetRefreshToken(),
		RefreshExpiresAt:   grpcconv.TimeFromProto(t.GetRefreshExpiresAt()),
		TwoFactorToken:     t.GetTwoFactorToken(),
		TwoFactorExpiresAt: grpcconv.TimeFromProto(t.GetTwoFactorExpiresAt()),
	}
}

func SessionToProto(u models.User, t models.AuthTokens) *gen.Session {
	return &gen.Session{
		User:   UserToProto(u),
		Tokens: TokensToProto(t),
	}
}

func SessionFromProto(s *gen.Session) (models.User, models.AuthTokens, error) {
	user, err := UserFromProto(s.GetUser())
	if err != nil {
		return models.User{}, models.AuthTokens{}, err
	}

	return user, TokensFromProto(s.GetTokens()), nil
}

func UsersFromProto(list []*gen.User) ([]models.User, error) {
	result := make([]models.User, 0, len(list))
	for _, u := range list {
		user, err := UserFromProto(u)
		if err != nil {
			return nil, err
		}
		result = append(result, user)
	}

	return result, nil
}

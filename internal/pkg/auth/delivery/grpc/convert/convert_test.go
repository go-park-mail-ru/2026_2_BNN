package convert

import (
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/auth/delivery/grpc/gen"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionConvert(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	user := models.User{
		ID:           uuid.NewV4(),
		Login:        "seyran",
		Avatar:       "avatars/1.png",
		Email:        "seyran@example.com",
		IsExternal:   true,
		Is2FAEnabled: true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	tokens := models.AuthTokens{
		AccessToken:      "access",
		AccessExpiresAt:  now,
		RefreshToken:     "refresh",
		RefreshExpiresAt: now,
	}

	gotUser, gotTokens, err := SessionFromProto(SessionToProto(user, tokens))
	require.NoError(t, err)
	assert.Equal(t, user, gotUser)
	assert.Equal(t, tokens, gotTokens)

	_, _, err = SessionFromProto(&gen.Session{User: &gen.User{Id: "bad"}})
	assert.ErrorIs(t, err, models.ErrValidation)
}

package client

import (
	"context"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestClientListHistory(t *testing.T) {
	client, uc := newClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	actor := models.OnlineConfUser{
		ID: uuid.NewV4(), Login: "liza",
	}
	now := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)

	oldKey := models.OnlineConfKey{
		ID:        uuid.NewV4(),
		Scope:     onlineconf.ScopeBackend,
		Key:       onlineconf.KeyAttachmentsUploadEnabled,
		Kind:      onlineconf.KindFlag,
		Version:   1,
		UpdatedBy: actor,
		UpdatedAt: now,
		Flag: &models.OnlineConfFlag{
			Enabled:        true,
			RolloutPercent: 10,
			IncludeUsers: []models.OnlineConfUser{
				{ID: uuid.NewV4(), Login: "included"},
			},
			ExcludeUsers: []models.OnlineConfUser{
				{ID: uuid.NewV4(), Login: "excluded"},
			},
		},
	}

	newKey := oldKey
	newKey.Version = 2
	newKey.UpdatedAt = now.Add(time.Minute)

	flag := *oldKey.Flag
	flag.RolloutPercent = 50
	newKey.Flag = &flag

	history := []models.OnlineConfHistory{
		{
			ID:        uuid.NewV4(),
			KeyID:     oldKey.ID,
			Action:    "delete",
			OldValue:  &newKey,
			User:      actor,
			CreatedAt: now.Add(2 * time.Minute),
		},
		{
			ID:        uuid.NewV4(),
			KeyID:     oldKey.ID,
			Action:    "update",
			OldValue:  &oldKey,
			NewValue:  &newKey,
			User:      actor,
			CreatedAt: now.Add(time.Minute),
		},
		{
			ID:        uuid.NewV4(),
			KeyID:     oldKey.ID,
			Action:    "create",
			NewValue:  &oldKey,
			User:      actor,
			CreatedAt: now,
		},
	}

	uc.EXPECT().
		ListHistory(gomock.Any(), oldKey.ID).
		Return(history, nil)

	got, err := client.ListHistory(ctx, oldKey.ID)
	require.NoError(t, err)
	assert.Equal(t, history, got)
}

func TestClientListHistoryInvalidID(t *testing.T) {
	client, _ := newClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.client.ListHistory(ctx, &gen.ListHistoryRequest{
		KeyId: "invalid",
	})

	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

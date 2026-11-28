package client

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	onlineconfgrpc "bnn/internal/pkg/onlineconf/delivery/grpc"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"
	"bnn/internal/pkg/onlineconf/mocks"
	"bnn/internal/pkg/utils/grpcutil/grpctest"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newClient(t *testing.T) (*Client, *mocks.MockUsecase) {
	t.Helper()

	uc := mocks.NewMockUsecase(gomock.NewController(t))
	conn := grpctest.Dial(t, func(server *grpc.Server) {
		gen.RegisterOnlineConfServer(
			server, onlineconfgrpc.NewHandler(uc),
		)
	})

	return NewClient(conn), uc
}

func testInput() onlineconf.KeyInput {
	return onlineconf.KeyInput{
		Scope: "backend", Key: "test.value", Kind: "config",
		IsPublic: true,
		Config: &models.OnlineConfConfig{
			ValueType: "number",
			Value:     json.RawMessage("10485760"),
		},
	}
}

func TestClientCRUD(t *testing.T) {
	client, uc := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	actor := uuid.NewV4()
	input := testInput()
	key := models.OnlineConfKey{
		ID:       uuid.NewV4(),
		Scope:    input.Scope,
		Key:      input.Key,
		Kind:     input.Kind,
		IsPublic: input.IsPublic,
		Config:   input.Config,
		Version:  1,
		UpdatedBy: models.OnlineConfUser{
			ID: actor, Login: "liza",
		},
		UpdatedAt: time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC),
	}

	uc.EXPECT().ListKeys(
		gomock.Any(), "test", "backend", "config",
	).Return([]models.OnlineConfKey{key}, nil)

	keys, err := client.ListKeys(ctx, "test", "backend", "config")
	require.NoError(t, err)
	assert.Equal(t, []models.OnlineConfKey{key}, keys)

	uc.EXPECT().GetKey(gomock.Any(), key.ID).Return(key, nil)

	found, err := client.GetKey(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, key, found)

	uc.EXPECT().CreateKey(
		gomock.Any(), actor, input,
	).Return(key, nil)

	created, err := client.CreateKey(ctx, actor, input)
	require.NoError(t, err)
	assert.Equal(t, key, created)

	uc.EXPECT().UpdateKey(
		gomock.Any(), actor, key.ID, int32(1), input,
	).Return(key, nil)

	updated, err := client.UpdateKey(ctx, actor, key.ID, 1, input)
	require.NoError(t, err)
	assert.Equal(t, key, updated)

	uc.EXPECT().DeleteKey(
		gomock.Any(), actor, key.ID, int32(1),
	).Return(nil)

	assert.NoError(t, client.DeleteKey(ctx, actor, key.ID, 1))
}

func TestClientSnapshot(t *testing.T) {
	client, uc := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	empty := models.Snapshot{
		Revision: 0,
		Keys:     []models.OnlineConfKey{},
	}

	uc.EXPECT().GetSnapshot(
		gomock.Any(), gomock.Nil(),
	).Return(empty, true, nil)

	snapshot, changed, err := client.GetSnapshot(ctx, nil)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, empty, snapshot)

	known := int64(0)
	uc.EXPECT().GetSnapshot(
		gomock.Any(), &known,
	).Return(models.Snapshot{}, false, nil)

	snapshot, changed, err = client.GetSnapshot(ctx, &known)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Empty(t, snapshot)

	uc.EXPECT().GetSnapshot(
		gomock.Any(), gomock.Nil(),
	).Return(
		models.Snapshot{}, false, models.ErrOnlineConfSnapshotNotReady,
	)

	_, _, err = client.GetSnapshot(ctx, nil)
	assert.ErrorIs(t, err, models.ErrOnlineConfSnapshotNotReady)
}

func TestClientDomainErrors(t *testing.T) {
	client, uc := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	actor, id := uuid.NewV4(), uuid.NewV4()
	input := testInput()

	uc.EXPECT().GetKey(gomock.Any(), id).
		Return(models.OnlineConfKey{}, models.ErrOnlineConfKeyNotFound)

	_, err := client.GetKey(ctx, id)
	assert.ErrorIs(t, err, models.ErrOnlineConfKeyNotFound)

	uc.EXPECT().CreateKey(gomock.Any(), actor, input).
		Return(models.OnlineConfKey{}, models.ErrOnlineConfKeyExists)

	_, err = client.CreateKey(ctx, actor, input)
	assert.ErrorIs(t, err, models.ErrOnlineConfKeyExists)

	uc.EXPECT().UpdateKey(gomock.Any(), actor, id, int32(1), input).
		Return(models.OnlineConfKey{}, models.ErrOnlineConfVersionConflict)

	_, err = client.UpdateKey(ctx, actor, id, 1, input)
	assert.ErrorIs(t, err, models.ErrOnlineConfVersionConflict)

	uc.EXPECT().GetKey(gomock.Any(), id).
		Return(models.OnlineConfKey{}, errors.New("private database error"))

	_, err = client.GetKey(ctx, id)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, "internal error", status.Convert(err).Message())
}

func TestHandlerRejectsInvalidRequests(t *testing.T) {
	client, _ := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.client.GetKey(ctx, &gen.GetKeyRequest{
		KeyId: "broken",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = client.client.CreateKey(ctx, &gen.CreateKeyRequest{
		UserId: uuid.NewV4().String(),
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = client.client.CreateKey(ctx, &gen.CreateKeyRequest{
		UserId: uuid.NewV4().String(),
		Key: &gen.KeyInput{
			Scope: "backend",
			Key:   "test.flag",
			Kind:  "flag",
			Settings: &gen.KeyInput_Flag{
				Flag: &gen.FlagInput{
					IncludeUserIds: []string{"broken"},
				},
			},
		},
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

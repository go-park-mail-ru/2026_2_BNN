package client

import (
	"context"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/grpc/convert"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcerr"

	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
)

var _ onlineconf.CoreUsecase = (*Client)(nil)

type Client struct {
	client gen.OnlineConfClient
}

func NewClient(conn grpc.ClientConnInterface) *Client {
	return &Client{client: gen.NewOnlineConfClient(conn)}
}

func (c *Client) GetSnapshot(
	ctx context.Context,
	revision *int64,
) (models.Snapshot, bool, error) {
	response, err := c.client.GetSnapshot(
		ctx, &gen.GetSnapshotRequest{Revision: revision},
	)
	if err != nil {
		return models.Snapshot{}, false, grpcerr.FromStatus(err)
	}

	if !response.GetChanged() {
		return models.Snapshot{}, false, nil
	}

	snapshot, err := convert.SnapshotFromProto(response.GetSnapshot())
	if err != nil {
		return models.Snapshot{}, false, err
	}

	return snapshot, true, nil
}

func (c *Client) ListScopes() []string {
	return []string{
		onlineconf.ScopeFrontend,
		onlineconf.ScopeBackend,
		onlineconf.ScopeAuth,
		onlineconf.ScopeNotes,
	}
}

func (c *Client) ListKeys(
	ctx context.Context,
	query, scope, kind string,
) ([]models.OnlineConfKey, error) {
	response, err := c.client.ListKeys(ctx, &gen.ListKeysRequest{
		Query: query,
		Scope: scope,
		Kind:  kind,
	})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}

	return convert.KeysFromProto(response.GetKeys())
}

func (c *Client) GetKey(
	ctx context.Context,
	id uuid.UUID,
) (models.OnlineConfKey, error) {
	response, err := c.client.GetKey(ctx, &gen.GetKeyRequest{
		KeyId: id.String(),
	})
	if err != nil {
		return models.OnlineConfKey{}, grpcerr.FromStatus(err)
	}

	return convert.KeyFromProto(response)
}

func (c *Client) CreateKey(
	ctx context.Context,
	userID uuid.UUID,
	input onlineconf.KeyInput,
) (models.OnlineConfKey, error) {
	payload, err := convert.InputToProto(input)
	if err != nil {
		return models.OnlineConfKey{}, err
	}

	response, err := c.client.CreateKey(ctx, &gen.CreateKeyRequest{
		UserId: userID.String(),
		Key:    payload,
	})
	if err != nil {
		return models.OnlineConfKey{}, grpcerr.FromStatus(err)
	}

	return convert.KeyFromProto(response)
}

func (c *Client) UpdateKey(
	ctx context.Context,
	userID, id uuid.UUID,
	version int32,
	input onlineconf.KeyInput,
) (models.OnlineConfKey, error) {
	payload, err := convert.InputToProto(input)
	if err != nil {
		return models.OnlineConfKey{}, err
	}

	response, err := c.client.UpdateKey(ctx, &gen.UpdateKeyRequest{
		UserId:  userID.String(),
		KeyId:   id.String(),
		Version: version,
		Key:     payload,
	})
	if err != nil {
		return models.OnlineConfKey{}, grpcerr.FromStatus(err)
	}

	return convert.KeyFromProto(response)
}

func (c *Client) DeleteKey(
	ctx context.Context,
	userID, id uuid.UUID,
	version int32,
) error {
	_, err := c.client.DeleteKey(ctx, &gen.DeleteKeyRequest{
		UserId:  userID.String(),
		KeyId:   id.String(),
		Version: version,
	})

	return grpcerr.FromStatus(err)
}

package client

import (
	"context"

	"bnn/internal/models"
	"bnn/internal/pkg/tags"
	"bnn/internal/pkg/tags/delivery/grpc/convert"
	"bnn/internal/pkg/tags/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcerr"

	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
)

var _ tags.Usecase = (*Client)(nil)

type Client struct {
	client gen.TagsClient
}

func NewClient(conn grpc.ClientConnInterface) *Client {
	return &Client{client: gen.NewTagsClient(conn)}
}

func (c *Client) ListTags(ctx context.Context, userID uuid.UUID) ([]models.Tag, error) {
	resp, err := c.client.ListTags(ctx, &gen.ListTagsRequest{UserId: userID.String()})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}

	return convert.TagsFromProto(resp.GetTags())
}

func (c *Client) CreateTag(ctx context.Context, userID uuid.UUID, name, color string) (models.Tag, error) {
	resp, err := c.client.CreateTag(ctx, &gen.CreateTagRequest{
		UserId: userID.String(),
		Name:   name,
		Color:  color,
	})
	if err != nil {
		return models.Tag{}, grpcerr.FromStatus(err)
	}

	return convert.TagFromProto(resp)
}

func (c *Client) UpdateTag(ctx context.Context, userID, tagID uuid.UUID, name, color string) (models.Tag, error) {
	resp, err := c.client.UpdateTag(ctx, &gen.UpdateTagRequest{
		UserId: userID.String(),
		TagId:  tagID.String(),
		Name:   name,
		Color:  color,
	})
	if err != nil {
		return models.Tag{}, grpcerr.FromStatus(err)
	}

	return convert.TagFromProto(resp)
}

func (c *Client) DeleteTag(ctx context.Context, userID, tagID uuid.UUID) error {
	_, err := c.client.DeleteTag(ctx, &gen.DeleteTagRequest{
		UserId: userID.String(),
		TagId:  tagID.String(),
	})

	return grpcerr.FromStatus(err)
}

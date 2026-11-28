package client

import (
	"bytes"
	"context"
	"io"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments"
	"bnn/internal/pkg/attachments/delivery/grpc/convert"
	"bnn/internal/pkg/attachments/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcerr"

	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
)

var _ attachments.Usecase = (*Client)(nil)

type Client struct {
	client gen.AttachmentsClient
}

func NewClient(conn grpc.ClientConnInterface) *Client {
	return &Client{client: gen.NewAttachmentsClient(conn)}
}

func (c *Client) UploadAttachment(ctx context.Context, userID, blockID uuid.UUID, data []byte, contentType string) (models.Attachment, error) {
	resp, err := c.client.UploadAttachment(ctx, &gen.UploadAttachmentRequest{
		UserId:      userID.String(),
		BlockId:     blockID.String(),
		Data:        data,
		ContentType: contentType,
	})
	if err != nil {
		return models.Attachment{}, grpcerr.FromStatus(err)
	}

	return convert.AttachmentFromProto(resp)
}

func (c *Client) GetAttachment(ctx context.Context, userID, attachmentID uuid.UUID) (models.File, error) {
	resp, err := c.client.GetAttachment(ctx, &gen.AttachmentRequest{
		UserId:       userID.String(),
		AttachmentId: attachmentID.String(),
	})
	if err != nil {
		return models.File{}, grpcerr.FromStatus(err)
	}

	return models.File{
		Body:        io.NopCloser(bytes.NewReader(resp.GetData())),
		ContentType: resp.GetContentType(),
		Size:        resp.GetSize(),
	}, nil
}

func (c *Client) DeleteAttachment(ctx context.Context, userID, attachmentID uuid.UUID) error {
	_, err := c.client.DeleteAttachment(ctx, &gen.AttachmentRequest{
		UserId:       userID.String(),
		AttachmentId: attachmentID.String(),
	})

	return grpcerr.FromStatus(err)
}

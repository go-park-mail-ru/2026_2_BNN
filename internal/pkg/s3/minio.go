package s3

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

type Client struct {
	client     *minio.Client
	publicBase string
}

func NewClient(client *minio.Client, publicBase string) *Client {
	return &Client{
		client:     client,
		publicBase: publicBase,
	}
}

func (c *Client) Upload(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error) {
	_, err := c.client.PutObject(
		ctx,
		bucket,
		objectName,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", c.publicBase, bucket, objectName), nil
}

func (c *Client) Delete(ctx context.Context, bucket, objectName string) error {
	err := c.client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}

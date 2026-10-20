package s3

import (
	"bytes"
	"context"
	"net/url"

	"bnn/internal/pkg/s3"

	"github.com/minio/minio-go/v7"
)

type Repo struct {
	client *minio.Client
	bucket string
}

func NewRepo(ctx context.Context, client *minio.Client, bucket string) (*Repo, error) {
	if err := s3.EnsureBucket(ctx, client, bucket); err != nil {
		return nil, err
	}
	if err := s3.AllowPublicRead(ctx, client, bucket); err != nil {
		return nil, err
	}
	return &Repo{client: client, bucket: bucket}, nil
}

func (r *Repo) PutFile(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := r.client.PutObject(ctx, r.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (r *Repo) DeleteFile(ctx context.Context, key string) error {
	return r.client.RemoveObject(ctx, r.bucket, key, minio.RemoveObjectOptions{})
}

func (r *Repo) PublicPath(key string) (string, error) {
	return url.JoinPath("/", r.bucket, key)
}

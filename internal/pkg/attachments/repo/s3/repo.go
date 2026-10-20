package s3

import (
	"bytes"
	"context"
	"fmt"

	"bnn/internal/models"
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

	return &Repo{
		client: client,
		bucket: bucket,
	}, nil
}

func (r *Repo) PutFile(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := r.client.PutObject(ctx, r.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put file: %w", err)
	}

	return nil
}

func (r *Repo) GetFile(ctx context.Context, key string) (models.File, error) {
	object, err := r.client.GetObject(ctx, r.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return models.File{}, fmt.Errorf("get file: %w", err)
	}

	info, err := object.Stat()
	if err != nil {
		object.Close()

		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return models.File{}, models.ErrAttachmentNotFound
		}

		return models.File{}, fmt.Errorf("stat file: %w", err)
	}

	return models.File{
		Body:        object,
		ContentType: info.ContentType,
		Size:        info.Size,
	}, nil
}

func (r *Repo) DeleteFile(ctx context.Context, key string) error {
	err := r.client.RemoveObject(ctx, r.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

package attachments

import (
	"context"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type Usecase interface {
	UploadAttachment(ctx context.Context, userID, blockID uuid.UUID, data []byte, contentType string) (models.Attachment, error)
	GetAttachment(ctx context.Context, userID, attachmentID uuid.UUID) (models.File, error)
	DeleteAttachment(ctx context.Context, userID, attachmentID uuid.UUID) error
}

type Repo interface {
	ReadBlockOwner(ctx context.Context, blockID uuid.UUID) (uuid.UUID, error)
	CreateAttachment(ctx context.Context, attachment models.Attachment) error
	ReadAttachment(ctx context.Context, id uuid.UUID) (models.Attachment, error)
	DeleteAttachment(ctx context.Context, id uuid.UUID) error
}

type FileRepo interface {
	PutFile(ctx context.Context, key string, data []byte, contentType string) error
	GetFile(ctx context.Context, key string) (models.File, error)
	DeleteFile(ctx context.Context, key string) error
}

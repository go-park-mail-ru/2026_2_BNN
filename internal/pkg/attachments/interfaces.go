package attachments

import (
	"context"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

//go:generate go run go.uber.org/mock/mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks
type Usecase interface {
	UploadAttachment(ctx context.Context, userID, blockID uuid.UUID, data []byte, contentType string) (models.Attachment, error)
	GetAttachment(ctx context.Context, userID, attachmentID uuid.UUID) (models.File, error)
	DeleteAttachment(ctx context.Context, userID, attachmentID uuid.UUID) error
}

type Repo interface {
	CanEditBlock(ctx context.Context, userID, blockID uuid.UUID) (bool, error)
	CreateAttachment(ctx context.Context, userID uuid.UUID, attachment models.Attachment) error
	ReadAttachmentPath(ctx context.Context, userID, attachmentID uuid.UUID) (string, error)
	DeleteAttachment(ctx context.Context, userID, attachmentID uuid.UUID) (string, error)
}

type FileRepo interface {
	PutFile(ctx context.Context, key string, data []byte, contentType string) error
	GetFile(ctx context.Context, key string) (models.File, error)
	DeleteFile(ctx context.Context, key string) error
}

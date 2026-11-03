package tags

import (
	"context"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

//go:generate go run go.uber.org/mock/mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks
type Usecase interface {
	ListTags(ctx context.Context, userID uuid.UUID) ([]models.Tag, error)
	CreateTag(ctx context.Context, userID uuid.UUID, name, color string) (models.Tag, error)
	UpdateTag(ctx context.Context, userID, tagID uuid.UUID, name, color string) (models.Tag, error)
	DeleteTag(ctx context.Context, userID, tagID uuid.UUID) error
}

type Repo interface {
	ReadTags(ctx context.Context, userID uuid.UUID) ([]models.Tag, error)
	ReadTag(ctx context.Context, id uuid.UUID) (models.Tag, error)
	CreateTag(ctx context.Context, tag models.Tag) error
	UpdateTag(ctx context.Context, tag models.Tag) error
	DeleteTag(ctx context.Context, id uuid.UUID) error
}

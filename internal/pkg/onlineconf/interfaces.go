package onlineconf

import (
	"context"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

type Repo interface {
	ReadKeys(ctx context.Context, query, scope, kind string) ([]models.OnlineConfKey, error)
	ReadKey(ctx context.Context, id uuid.UUID) (models.OnlineConfKey, error)
	CreateKey(ctx context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error)
	UpdateKey(ctx context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error)
	DeleteKey(ctx context.Context, id uuid.UUID, version int32, userID uuid.UUID) error
	ReadHistory(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfHistory, error)
	ReadSnapshot(ctx context.Context) (models.Snapshot, error)
}

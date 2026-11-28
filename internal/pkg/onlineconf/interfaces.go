package onlineconf

import (
	"context"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

//go:generate go run go.uber.org/mock/mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type CoreUsecase interface {
	GetSnapshot(ctx context.Context, revision *int64) (models.Snapshot, bool, error)
	ListScopes() []string
	ListKeys(ctx context.Context, query, scope, kind string) ([]models.OnlineConfKey, error)
	GetKey(ctx context.Context, id uuid.UUID) (models.OnlineConfKey, error)
	CreateKey(ctx context.Context, userID uuid.UUID, input KeyInput) (models.OnlineConfKey, error)
	UpdateKey(ctx context.Context, userID, id uuid.UUID, version int32, input KeyInput) (models.OnlineConfKey, error)
	DeleteKey(ctx context.Context, userID, id uuid.UUID, version int32) error
}

type Usecase interface {
	CoreUsecase
	ListHistory(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfHistory, error)
}

type Repo interface {
	ReadKeys(ctx context.Context, query, scope, kind string) ([]models.OnlineConfKey, error)
	ReadKey(ctx context.Context, id uuid.UUID) (models.OnlineConfKey, error)
	CreateKey(ctx context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error)
	UpdateKey(ctx context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error)
	DeleteKey(ctx context.Context, id uuid.UUID, version int32, userID uuid.UUID) error
	ReadHistory(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfHistory, error)
	ReadSnapshot(ctx context.Context) (models.Snapshot, error)
}

type AuthClient interface {
	GetUser(ctx context.Context, userID uuid.UUID) (models.User, error)
}

type KeyInput struct {
	Scope       string
	Key         string
	Kind        string
	Description string
	IsPublic    bool
	Config      *models.OnlineConfConfig
	Flag        *FlagInput
}

type FlagInput struct {
	Enabled        bool
	RolloutPercent int32
	IncludeUserIDs []uuid.UUID
	ExcludeUserIDs []uuid.UUID
}

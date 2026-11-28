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
	RolloutPlanUsecase
	RolloutUsecase
	ListHistory(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfHistory, error)
}

type RolloutUsecase interface {
	SetRolloutPlan(ctx context.Context, actor, keyID uuid.UUID, planID *uuid.UUID, version int32) (models.OnlineConfKey, error)
	GetRollout(ctx context.Context, keyID uuid.UUID) (models.OnlineConfRollout, error)
	AdvanceRollout(ctx context.Context, actor, keyID uuid.UUID, version int32, rollback bool) (models.OnlineConfKey, error)
	ListSchedules(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfSchedule, error)
	CreateSchedule(ctx context.Context, actor, keyID uuid.UUID, input ScheduleInput) (models.OnlineConfSchedule, error)
	DeleteSchedule(ctx context.Context, keyID, scheduleID uuid.UUID) error
}

type RolloutRepo interface {
	SetRolloutPlan(ctx context.Context, actor, keyID uuid.UUID, planID *uuid.UUID, version int32) (models.OnlineConfKey, error)
	ReadRollout(ctx context.Context, keyID uuid.UUID) (models.OnlineConfRollout, error)
	AdvanceRollout(ctx context.Context, actor, keyID uuid.UUID, version int32, rollback bool) (models.OnlineConfKey, error)
	ReadSchedules(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfSchedule, error)
	CreateSchedule(ctx context.Context, schedule models.OnlineConfSchedule) (models.OnlineConfSchedule, error)
	DeleteSchedule(ctx context.Context, keyID, scheduleID uuid.UUID) error
	ProcessRollouts(ctx context.Context) error
}

type RolloutPlanUsecase interface {
	ListRolloutPlans(ctx context.Context) ([]models.OnlineConfRolloutPlan, error)
	GetRolloutPlan(ctx context.Context, id uuid.UUID) (models.OnlineConfRolloutPlan, error)
	CreateRolloutPlan(ctx context.Context, input RolloutPlanInput) (models.OnlineConfRolloutPlan, error)
	UpdateRolloutPlan(ctx context.Context, id uuid.UUID, input RolloutPlanInput) (models.OnlineConfRolloutPlan, error)
	DeleteRolloutPlan(ctx context.Context, id uuid.UUID) error
}

type RolloutPlanRepo interface {
	ReadRolloutPlans(ctx context.Context) ([]models.OnlineConfRolloutPlan, error)
	ReadRolloutPlan(ctx context.Context, id uuid.UUID) (models.OnlineConfRolloutPlan, error)
	CreateRolloutPlan(ctx context.Context, plan models.OnlineConfRolloutPlan) (models.OnlineConfRolloutPlan, error)
	UpdateRolloutPlan(ctx context.Context, plan models.OnlineConfRolloutPlan) (models.OnlineConfRolloutPlan, error)
	DeleteRolloutPlan(ctx context.Context, id uuid.UUID) error
}

type Repo interface {
	RolloutPlanRepo
	RolloutRepo
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

type AdminSource interface {
	ListAdmins(ctx context.Context) ([]models.User, error)
}

type SnapshotSource interface {
	GetSnapshot(ctx context.Context, revision *int64) (models.Snapshot, bool, error)
}

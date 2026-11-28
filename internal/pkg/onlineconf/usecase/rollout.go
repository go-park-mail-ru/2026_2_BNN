package usecase

import (
	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"context"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"log/slog"
	"time"
)

const rolloutWorkerInterval = 10 * time.Second
const rolloutWorkerTimeout = 10 * time.Second

func (u *Usecase) SetRolloutPlan(ctx context.Context, actor, keyID uuid.UUID, planID *uuid.UUID, version int32) (models.OnlineConfKey, error) {
	if err := validateWrite(actor, keyID, version); err != nil {
		return models.OnlineConfKey{}, err
	}
	if planID != nil && *planID == uuid.Nil {
		return models.OnlineConfKey{}, fmt.Errorf("%w: invalid plan id", models.ErrValidation)
	}
	key, err := u.repo.SetRolloutPlan(ctx, actor, keyID, planID, version)
	if err == nil {
		u.refreshSnapshot(ctx)
	}
	return key, err
}
func (u *Usecase) GetRollout(ctx context.Context, keyID uuid.UUID) (models.OnlineConfRollout, error) {
	if keyID == uuid.Nil {
		return models.OnlineConfRollout{}, models.ErrValidation
	}
	return u.repo.ReadRollout(ctx, keyID)
}
func (u *Usecase) AdvanceRollout(ctx context.Context, actor, keyID uuid.UUID, version int32, rollback bool) (models.OnlineConfKey, error) {
	if err := validateWrite(actor, keyID, version); err != nil {
		return models.OnlineConfKey{}, err
	}
	key, err := u.repo.AdvanceRollout(ctx, actor, keyID, version, rollback)
	if err == nil {
		u.refreshSnapshot(ctx)
	}
	return key, err
}
func (u *Usecase) ListSchedules(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfSchedule, error) {
	if keyID == uuid.Nil {
		return nil, models.ErrValidation
	}
	return u.repo.ReadSchedules(ctx, keyID)
}
func (u *Usecase) CreateSchedule(ctx context.Context, actor, keyID uuid.UUID, input onlineconf.ScheduleInput) (models.OnlineConfSchedule, error) {
	if actor == uuid.Nil || keyID == uuid.Nil {
		return models.OnlineConfSchedule{}, models.ErrValidation
	}
	if err := validateScheduleInput(input, time.Now()); err != nil {
		return models.OnlineConfSchedule{}, err
	}
	return u.repo.CreateSchedule(ctx, models.OnlineConfSchedule{
		ID: uuid.NewV4(), KeyID: keyID, RunAt: input.RunAt.UTC(), Enabled: clonePointer(input.Enabled), RolloutPercent: clonePointer(input.RolloutPercent), CreatedBy: actor,
	})
}
func (u *Usecase) DeleteSchedule(ctx context.Context, keyID, scheduleID uuid.UUID) error {
	if keyID == uuid.Nil || scheduleID == uuid.Nil {
		return models.ErrValidation
	}
	return u.repo.DeleteSchedule(ctx, keyID, scheduleID)
}
func validateScheduleInput(input onlineconf.ScheduleInput, now time.Time) error {
	if input.RunAt.IsZero() || !input.RunAt.After(now) {
		return fmt.Errorf("%w: run_at must be in the future", models.ErrValidation)
	}
	if input.Enabled == nil && input.RolloutPercent == nil {
		return fmt.Errorf("%w: enabled or rollout_percent is required", models.ErrValidation)
	}
	if input.RolloutPercent != nil && (*input.RolloutPercent < onlineconf.RolloutPercentMin || *input.RolloutPercent > onlineconf.RolloutPercentMax) {
		return fmt.Errorf("%w: invalid rollout percent", models.ErrValidation)
	}
	return nil
}
func (u *Usecase) RunRollouts(ctx context.Context) {
	ticker := time.NewTicker(rolloutWorkerInterval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		workCtx, cancel := context.WithTimeout(ctx, rolloutWorkerTimeout)
		err := u.repo.ProcessRollouts(workCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "OnlineConf rollout worker failed", "error", err)
		}
		u.refreshSnapshot(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

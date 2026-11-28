package postgres

import (
	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	uuid "github.com/satori/go.uuid"
	"time"
)

const scheduleBatchSize = 100

//go:embed queries/read_rollout_state.sql
var readRolloutStateQuery string

//go:embed queries/read_rollout_plan_for_share.sql
var readRolloutPlanForShareQuery string

//go:embed queries/save_rollout.sql
var saveRolloutQuery string

//go:embed queries/rollout_can_approve.sql
var rolloutCanApproveQuery string

//go:embed queries/mark_rollouts_ready.sql
var markRolloutsReadyQuery string

//go:embed queries/read_schedules.sql
var readSchedulesQuery string

//go:embed queries/create_schedule.sql
var createScheduleQuery string

//go:embed queries/lock_schedule.sql
var lockScheduleQuery string

//go:embed queries/delete_schedule.sql
var deleteScheduleQuery string

//go:embed queries/read_due_schedules.sql
var readDueSchedulesQuery string

//go:embed queries/lock_due_schedule.sql
var lockDueScheduleQuery string

//go:embed queries/apply_schedule.sql
var applyScheduleQuery string

//go:embed queries/finish_schedule.sql
var finishScheduleQuery string

//go:embed queries/fail_schedule.sql
var failScheduleQuery string

//go:embed queries/database_time.sql
var databaseTimeQuery string

func lockCurrentKey(ctx context.Context, tx pgx.Tx, id uuid.UUID) (models.OnlineConfKey, error) {
	var version int32
	if err := tx.QueryRow(ctx, lockKeyQuery, id).Scan(&version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.OnlineConfKey{}, models.ErrOnlineConfKeyNotFound
		}
		return models.OnlineConfKey{}, fmt.Errorf("lock key: %w", err)
	}
	return readKey(ctx, tx, id)
}

func (r *Repo) SetRolloutPlan(ctx context.Context, actor, keyID uuid.UUID, planID *uuid.UUID, version int32) (models.OnlineConfKey, error) {
	var saved models.OnlineConfKey
	err := r.withTx(ctx, func(tx pgx.Tx) error {
		old, err := lockAndReadKey(ctx, tx, keyID, version)
		if err != nil {
			return err
		}
		if old.Kind != onlineconf.KindFlag {
			return models.ErrValidation
		}
		percent := old.Flag.RolloutPercent
		var step *int32
		var interval int32
		next := false
		if planID != nil {
			plan, err := scanRolloutPlan(tx.QueryRow(ctx, readRolloutPlanForShareQuery, *planID))
			if err != nil {
				return rolloutPlanError("read rollout plan", err)
			}
			if len(plan.Steps) < onlineconf.RolloutPlanMinSteps {
				return fmt.Errorf("invalid stored rollout plan")
			}
			first := int32(0)
			step = &first
			percent = plan.Steps[0]
			interval = plan.IntervalMin
			next = true
		}
		saved, err = saveRollout(ctx, tx, old, actor, planID, step, percent, interval, next)
		return err
	})
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	return saved, nil
}

func saveRollout(ctx context.Context, tx pgx.Tx, old models.OnlineConfKey, actor uuid.UUID, planID *uuid.UUID, step *int32, percent, interval int32, next bool) (models.OnlineConfKey, error) {
	if _, err := tx.Exec(ctx, saveRolloutQuery, old.ID, planID, step, percent, next, interval, actor); err != nil {
		return models.OnlineConfKey{}, err
	}
	saved, err := readKey(ctx, tx, old.ID)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	if err := recordChange(ctx, tx, old.ID, "update", &old, &saved, actor); err != nil {
		return models.OnlineConfKey{}, err
	}
	return saved, nil
}

func (r *Repo) AdvanceRollout(ctx context.Context, actor, keyID uuid.UUID, version int32, rollback bool) (models.OnlineConfKey, error) {
	var saved models.OnlineConfKey
	err := r.withTx(ctx, func(tx pgx.Tx) error {
		old, err := lockAndReadKey(ctx, tx, keyID, version)
		if err != nil {
			return err
		}
		if old.Kind != onlineconf.KindFlag || old.RolloutPlanID == nil || old.RolloutStep == nil {
			return models.ErrOnlineConfRolloutStepNotAllowed
		}
		plan, err := scanRolloutPlan(tx.QueryRow(ctx, readRolloutPlanForShareQuery, *old.RolloutPlanID))
		if err != nil {
			return rolloutPlanError("read rollout plan", err)
		}
		step := *old.RolloutStep
		if step < 0 || int(step) >= len(plan.Steps) {
			return fmt.Errorf("invalid stored rollout step")
		}
		if rollback {
			if step == 0 {
				return models.ErrOnlineConfRolloutStepNotAllowed
			}
			step--
		} else {
			if int(step)+1 >= len(plan.Steps) {
				return models.ErrOnlineConfRolloutStepNotAllowed
			}
			var ready bool
			if err := tx.QueryRow(ctx, rolloutCanApproveQuery, keyID).Scan(&ready); err != nil {
				return err
			}
			if !ready {
				return models.ErrOnlineConfRolloutNotReady
			}
			step++
		}
		saved, err = saveRollout(ctx, tx, old, actor, old.RolloutPlanID, &step, plan.Steps[step], plan.IntervalMin, int(step)+1 < len(plan.Steps))
		return err
	})
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	return saved, nil
}

func (r *Repo) ReadRollout(ctx context.Context, keyID uuid.UUID) (models.OnlineConfRollout, error) {
	var state models.OnlineConfRollout
	var kind string
	var planID *uuid.UUID
	var name *string
	var steps []int32
	var interval *int32
	var pending bool
	err := r.db.QueryRow(ctx, readRolloutStateQuery, keyID).Scan(
		&state.KeyID, &kind, &state.CurrentPercent, &state.Version, &planID, &state.Step, &state.StepReadyAt, &pending, &name, &steps, &interval,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, models.ErrOnlineConfKeyNotFound
	}
	if err != nil {
		return state, err
	}
	if kind != onlineconf.KindFlag {
		return state, models.ErrValidation
	}
	state.Status = onlineconf.RolloutUnassigned
	if planID == nil {
		return state, nil
	}
	if name == nil || interval == nil || state.Step == nil || *state.Step < 0 || int(*state.Step) >= len(steps) {
		return state, fmt.Errorf("invalid stored rollout state")
	}
	state.Plan = &models.OnlineConfRolloutPlan{ID: *planID, Name: *name, Steps: steps, IntervalMin: *interval}
	if int(*state.Step)+1 == len(steps) {
		state.Status = onlineconf.RolloutDone
		return state, nil
	}
	next := steps[*state.Step+1]
	state.NextPercent = &next
	state.Status = onlineconf.RolloutWaiting
	if pending {
		state.Status = onlineconf.RolloutPendingApproval
	}
	return state, nil
}

func scanSchedule(row pgx.Row) (models.OnlineConfSchedule, error) {
	var s models.OnlineConfSchedule
	err := row.Scan(&s.ID, &s.KeyID, &s.RunAt, &s.Enabled, &s.RolloutPercent, &s.Done, &s.Failed, &s.Error, &s.CreatedBy, &s.CreatedAt)
	return s, err
}
func (r *Repo) ReadSchedules(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfSchedule, error) {
	key, err := r.ReadKey(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key.Kind != onlineconf.KindFlag {
		return nil, models.ErrValidation
	}
	rows, err := r.db.Query(ctx, readSchedulesQuery, keyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]models.OnlineConfSchedule, 0)
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
func (r *Repo) CreateSchedule(ctx context.Context, s models.OnlineConfSchedule) (models.OnlineConfSchedule, error) {
	var saved models.OnlineConfSchedule
	err := r.withTx(ctx, func(tx pgx.Tx) error {
		key, err := lockCurrentKey(ctx, tx, s.KeyID)
		if err != nil {
			return err
		}
		if key.Kind != onlineconf.KindFlag {
			return models.ErrValidation
		}
		if err := validateScheduledPercent(key, s.RolloutPercent); err != nil {
			return err
		}
		var now time.Time
		if err := tx.QueryRow(ctx, databaseTimeQuery).Scan(&now); err != nil {
			return err
		}
		if !s.RunAt.After(now) {
			return fmt.Errorf("%w: run_at must be in the future", models.ErrValidation)
		}
		saved, err = scanSchedule(tx.QueryRow(ctx, createScheduleQuery, s.ID, s.KeyID, s.RunAt, s.Enabled, s.RolloutPercent, s.CreatedBy))
		return err
	})
	if err != nil {
		return models.OnlineConfSchedule{}, err
	}
	return saved, nil
}
func (r *Repo) DeleteSchedule(ctx context.Context, keyID, scheduleID uuid.UUID) error {
	return r.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := lockCurrentKey(ctx, tx, keyID); err != nil {
			return err
		}
		s, err := scanSchedule(tx.QueryRow(ctx, lockScheduleQuery, keyID, scheduleID))
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrOnlineConfScheduleNotFound
		}
		if err != nil {
			return err
		}
		if s.Done || s.Failed {
			return models.ErrOnlineConfScheduleFinished
		}
		_, err = tx.Exec(ctx, deleteScheduleQuery, keyID, scheduleID)
		return err
	})
}
func validateScheduledPercent(key models.OnlineConfKey, percent *int32) error {
	if percent != nil && key.RolloutPlanID != nil && *percent != key.Flag.RolloutPercent {
		return models.ErrOnlineConfRolloutStepNotAllowed
	}
	return nil
}
func (r *Repo) ProcessRollouts(ctx context.Context) error {
	if _, err := r.db.Exec(ctx, markRolloutsReadyQuery); err != nil {
		return fmt.Errorf("mark ready rollouts: %w", err)
	}
	rows, err := r.db.Query(ctx, readDueSchedulesQuery, scheduleBatchSize)
	if err != nil {
		return err
	}
	type due struct{ id, keyID uuid.UUID }
	jobs := make([]due, 0)
	for rows.Next() {
		var j due
		if err := rows.Scan(&j.id, &j.keyID); err != nil {
			rows.Close()
			return err
		}
		jobs = append(jobs, j)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var result error
	for _, j := range jobs {
		if err := ctx.Err(); err != nil {
			return errors.Join(result, err)
		}
		if err := r.applyDueSchedule(ctx, j.keyID, j.id); err != nil {
			result = errors.Join(result, fmt.Errorf("schedule %s: %w", j.id, err))
		}
	}
	return result
}
func (r *Repo) applyDueSchedule(ctx context.Context, keyID, id uuid.UUID) error {
	return r.withTx(ctx, func(tx pgx.Tx) error {
		old, err := lockCurrentKey(ctx, tx, keyID)
		if errors.Is(err, models.ErrOnlineConfKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		s, err := scanSchedule(tx.QueryRow(ctx, lockDueScheduleQuery, keyID, id))
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		policyErr := validateScheduledPercent(old, s.RolloutPercent)
		if old.Kind != onlineconf.KindFlag {
			policyErr = models.ErrValidation
		}
		if policyErr != nil {
			_, err := tx.Exec(ctx, failScheduleQuery, s.ID, policyErr.Error())
			return err
		}
		if _, err := tx.Exec(ctx, applyScheduleQuery, keyID, s.Enabled, s.RolloutPercent, s.CreatedBy); err != nil {
			return err
		}
		saved, err := readKey(ctx, tx, keyID)
		if err != nil {
			return err
		}
		if err := recordChange(ctx, tx, keyID, "update", &old, &saved, s.CreatedBy); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, finishScheduleQuery, s.ID)
		return err
	})
}

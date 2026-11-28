package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"bnn/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	uuid "github.com/satori/go.uuid"
)

const (
	rolloutPlanNameConstraint = "onlineconf_rollout_plans_name_key"
	rolloutPlanFKConstraint   = "onlineconf_keys_rollout_plan_id_fkey"
)

//go:embed queries/read_rollout_plans.sql
var readRolloutPlansQuery string

//go:embed queries/read_rollout_plan.sql
var readRolloutPlanQuery string

//go:embed queries/create_rollout_plan.sql
var createRolloutPlanQuery string

//go:embed queries/update_rollout_plan.sql
var updateRolloutPlanQuery string

//go:embed queries/delete_rollout_plan.sql
var deleteRolloutPlanQuery string

//go:embed queries/lock_rollout_plan.sql
var lockRolloutPlanQuery string

//go:embed queries/rollout_plan_in_use.sql
var rolloutPlanInUseQuery string

func scanRolloutPlan(row pgx.Row) (models.OnlineConfRolloutPlan, error) {
	var plan models.OnlineConfRolloutPlan
	if err := row.Scan(&plan.ID, &plan.Name, &plan.Steps, &plan.IntervalMin); err != nil {
		return models.OnlineConfRolloutPlan{}, err
	}
	return plan, nil
}

func (r *Repo) ReadRolloutPlans(ctx context.Context) ([]models.OnlineConfRolloutPlan, error) {
	rows, err := r.db.Query(ctx, readRolloutPlansQuery)
	if err != nil {
		return nil, fmt.Errorf("read rollout plans: %w", err)
	}
	defer rows.Close()

	plans := make([]models.OnlineConfRolloutPlan, 0)
	for rows.Next() {
		plan, err := scanRolloutPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan rollout plan: %w", err)
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read rollout plan rows: %w", err)
	}
	return plans, nil
}

func (r *Repo) ReadRolloutPlan(ctx context.Context, id uuid.UUID) (models.OnlineConfRolloutPlan, error) {
	plan, err := scanRolloutPlan(r.db.QueryRow(ctx, readRolloutPlanQuery, id))
	if err != nil {
		return models.OnlineConfRolloutPlan{}, rolloutPlanError("read rollout plan", err)
	}
	return plan, nil
}

func (r *Repo) CreateRolloutPlan(ctx context.Context, plan models.OnlineConfRolloutPlan) (models.OnlineConfRolloutPlan, error) {
	saved, err := scanRolloutPlan(r.db.QueryRow(ctx, createRolloutPlanQuery,
		plan.ID, plan.Name, plan.Steps, plan.IntervalMin,
	))
	if err != nil {
		return models.OnlineConfRolloutPlan{}, rolloutPlanError("create rollout plan", err)
	}
	return saved, nil
}

func (r *Repo) UpdateRolloutPlan(ctx context.Context, plan models.OnlineConfRolloutPlan) (models.OnlineConfRolloutPlan, error) {
	var saved models.OnlineConfRolloutPlan
	err := r.withTx(ctx, func(tx pgx.Tx) error {
		if err := lockUnusedRolloutPlan(ctx, tx, plan.ID); err != nil {
			return err
		}
		var err error
		saved, err = scanRolloutPlan(tx.QueryRow(ctx, updateRolloutPlanQuery,
			plan.ID, plan.Name, plan.Steps, plan.IntervalMin,
		))
		if err != nil {
			return rolloutPlanError("update rollout plan", err)
		}
		return nil
	})
	if err != nil {
		return models.OnlineConfRolloutPlan{}, err
	}
	return saved, nil
}

func (r *Repo) DeleteRolloutPlan(ctx context.Context, id uuid.UUID) error {
	return r.withTx(ctx, func(tx pgx.Tx) error {
		if err := lockUnusedRolloutPlan(ctx, tx, id); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, deleteRolloutPlanQuery, id)
		if err != nil {
			return rolloutPlanError("delete rollout plan", err)
		}
		if tag.RowsAffected() != 1 {
			return models.ErrOnlineConfRolloutPlanNotFound
		}
		return nil
	})
}

func lockUnusedRolloutPlan(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	var lockedID uuid.UUID
	if err := tx.QueryRow(ctx, lockRolloutPlanQuery, id).Scan(&lockedID); err != nil {
		return rolloutPlanError("lock rollout plan", err)
	}
	var inUse bool
	if err := tx.QueryRow(ctx, rolloutPlanInUseQuery, id).Scan(&inUse); err != nil {
		return fmt.Errorf("check rollout plan usage: %w", err)
	}
	if inUse {
		return models.ErrOnlineConfRolloutPlanInUse
	}
	return nil
}

func rolloutPlanError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrOnlineConfRolloutPlanNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && pgErr.ConstraintName == rolloutPlanNameConstraint:
			return models.ErrOnlineConfRolloutPlanExists
		case pgErr.Code == "23503" && pgErr.ConstraintName == rolloutPlanFKConstraint:
			return models.ErrOnlineConfRolloutPlanInUse
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

package postgres

import (
	"context"
	"testing"

	"bnn/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rolloutPlanColumns = []string{"id", "name", "steps", "interval_min"}

func rolloutPlanRows(plan models.OnlineConfRolloutPlan) *pgxmock.Rows {
	return pgxmock.NewRows(rolloutPlanColumns).AddRow(
		plan.ID.String(), plan.Name, plan.Steps, plan.IntervalMin,
	)
}

func expectRolloutPlanUsage(mock pgxmock.PgxPoolIface, id uuid.UUID, inUse bool) {
	mock.ExpectQuery(lockRolloutPlanQuery).WithArgs(id).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(id.String()))
	mock.ExpectQuery(rolloutPlanInUseQuery).WithArgs(id).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(inUse))
}

func TestRolloutPlanRepoCRUD(t *testing.T) {
	mock, repo := newMock(t)
	ctx := context.Background()
	plan := models.OnlineConfRolloutPlan{
		ID: uuid.NewV4(), Name: "Basic", Steps: []int32{5, 25, 100}, IntervalMin: 30,
	}

	mock.ExpectQuery(createRolloutPlanQuery).
		WithArgs(plan.ID, plan.Name, plan.Steps, plan.IntervalMin).
		WillReturnRows(rolloutPlanRows(plan))
	created, err := repo.CreateRolloutPlan(ctx, plan)
	require.NoError(t, err)
	assert.Equal(t, plan, created)

	mock.ExpectQuery(readRolloutPlansQuery).WillReturnRows(rolloutPlanRows(plan))
	plans, err := repo.ReadRolloutPlans(ctx)
	require.NoError(t, err)
	assert.Equal(t, []models.OnlineConfRolloutPlan{plan}, plans)

	mock.ExpectQuery(readRolloutPlanQuery).WithArgs(plan.ID).
		WillReturnRows(rolloutPlanRows(plan))
	found, err := repo.ReadRolloutPlan(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, plan, found)

	updated := plan
	updated.Name = "Updated"
	updated.Steps = []int32{10, 50, 100}
	mock.ExpectBegin()
	expectRolloutPlanUsage(mock, plan.ID, false)
	mock.ExpectQuery(updateRolloutPlanQuery).
		WithArgs(updated.ID, updated.Name, updated.Steps, updated.IntervalMin).
		WillReturnRows(rolloutPlanRows(updated))
	mock.ExpectCommit()
	saved, err := repo.UpdateRolloutPlan(ctx, updated)
	require.NoError(t, err)
	assert.Equal(t, updated, saved)

	mock.ExpectBegin()
	expectRolloutPlanUsage(mock, plan.ID, false)
	mock.ExpectExec(deleteRolloutPlanQuery).WithArgs(plan.ID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectCommit()
	require.NoError(t, repo.DeleteRolloutPlan(ctx, plan.ID))
}

func TestRolloutPlanRepoInUse(t *testing.T) {
	plan := models.OnlineConfRolloutPlan{
		ID: uuid.NewV4(), Name: "Basic", Steps: []int32{5, 100}, IntervalMin: 30,
	}
	for _, operation := range []string{"update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			mock, repo := newMock(t)
			mock.ExpectBegin()
			expectRolloutPlanUsage(mock, plan.ID, true)
			mock.ExpectRollback()

			var err error
			if operation == "update" {
				_, err = repo.UpdateRolloutPlan(context.Background(), plan)
			} else {
				err = repo.DeleteRolloutPlan(context.Background(), plan.ID)
			}
			assert.ErrorIs(t, err, models.ErrOnlineConfRolloutPlanInUse)
		})
	}
}

func TestRolloutPlanRepoNotFound(t *testing.T) {
	mock, repo := newMock(t)
	id := uuid.NewV4()
	mock.ExpectQuery(readRolloutPlanQuery).WithArgs(id).WillReturnError(pgx.ErrNoRows)
	_, err := repo.ReadRolloutPlan(context.Background(), id)
	assert.ErrorIs(t, err, models.ErrOnlineConfRolloutPlanNotFound)
}

func TestRolloutPlanRepoUpdateRollback(t *testing.T) {
	mock, repo := newMock(t)
	plan := models.OnlineConfRolloutPlan{
		ID: uuid.NewV4(), Name: "Existing name", Steps: []int32{5, 100}, IntervalMin: 30,
	}
	mock.ExpectBegin()
	expectRolloutPlanUsage(mock, plan.ID, false)
	mock.ExpectQuery(updateRolloutPlanQuery).
		WithArgs(plan.ID, plan.Name, plan.Steps, plan.IntervalMin).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: rolloutPlanNameConstraint})
	mock.ExpectRollback()
	_, err := repo.UpdateRolloutPlan(context.Background(), plan)
	assert.ErrorIs(t, err, models.ErrOnlineConfRolloutPlanExists)
}

func TestRolloutPlanRepoEmptyList(t *testing.T) {
	mock, repo := newMock(t)
	mock.ExpectQuery(readRolloutPlansQuery).
		WillReturnRows(pgxmock.NewRows(rolloutPlanColumns))
	plans, err := repo.ReadRolloutPlans(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, plans)
	assert.Empty(t, plans)
}

func TestRolloutPlanRepoCreateErrors(t *testing.T) {
	duplicateName := &pgconn.PgError{Code: "23505", ConstraintName: rolloutPlanNameConstraint}
	duplicateID := &pgconn.PgError{Code: "23505", ConstraintName: "onlineconf_rollout_plans_pkey"}
	for _, tt := range []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{"duplicate name", duplicateName, models.ErrOnlineConfRolloutPlanExists},
		{"other unique violation", duplicateID, duplicateID},
		{"database unavailable", errDB, errDB},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := newMock(t)
			plan := models.OnlineConfRolloutPlan{
				ID: uuid.NewV4(), Name: "Basic", Steps: []int32{5, 100}, IntervalMin: 30,
			}
			mock.ExpectQuery(createRolloutPlanQuery).
				WithArgs(plan.ID, plan.Name, plan.Steps, plan.IntervalMin).
				WillReturnError(tt.dbErr)
			_, err := repo.CreateRolloutPlan(context.Background(), plan)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

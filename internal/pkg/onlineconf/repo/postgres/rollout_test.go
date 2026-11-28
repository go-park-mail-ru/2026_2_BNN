package postgres

import (
	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func boundRolloutFlag() (models.OnlineConfKey, models.OnlineConfRolloutPlan) {
	key := newFlag()
	plan := models.OnlineConfRolloutPlan{ID: uuid.NewV4(), Name: "Basic", Steps: []int32{25, 50, 100}, IntervalMin: 30}
	step := int32(0)
	ready := key.UpdatedAt.Add(time.Minute)
	key.RolloutPlanID = &plan.ID
	key.RolloutStep = &step
	key.StepReadyAt = &ready
	return key, plan
}
func TestUpdateKeyCannotBypassRollout(t *testing.T) {
	mock, repo := newMock(t)
	old, _ := boundRolloutFlag()
	updated := old
	flag := *old.Flag
	flag.RolloutPercent = 100
	updated.Flag = &flag
	mock.ExpectBegin()
	expectLock(mock, old)
	expectRead(mock, old)
	mock.ExpectRollback()
	_, err := repo.UpdateKey(context.Background(), updated)
	assert.ErrorIs(t, err, models.ErrOnlineConfRolloutStepNotAllowed)
}
func TestAdvanceRolloutRequiresApprovalState(t *testing.T) {
	mock, repo := newMock(t)
	old, plan := boundRolloutFlag()
	mock.ExpectBegin()
	expectLock(mock, old)
	expectRead(mock, old)
	mock.ExpectQuery(readRolloutPlanForShareQuery).WithArgs(plan.ID).WillReturnRows(rolloutPlanRows(plan))
	mock.ExpectQuery(rolloutCanApproveQuery).WithArgs(old.ID).WillReturnRows(pgxmock.NewRows([]string{"ready"}).AddRow(false))
	mock.ExpectRollback()
	_, err := repo.AdvanceRollout(context.Background(), uuid.NewV4(), old.ID, old.Version, false)
	assert.ErrorIs(t, err, models.ErrOnlineConfRolloutNotReady)
}
func TestAdvanceRolloutRecordsHistoryAndRevision(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		name := "approve"
		if rollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			mock, repo := newMock(t)
			old, plan := boundRolloutFlag()
			actor := uuid.NewV4()
			target := int32(1)
			if rollback {
				current := int32(1)
				old.RolloutStep = &current
				old.Flag.RolloutPercent = 50
				target = 0
			}
			saved := old
			saved.Version++
			saved.UpdatedBy = models.OnlineConfUser{ID: actor, Login: "admin"}
			saved.RolloutStep = &target
			flag := *old.Flag
			flag.RolloutPercent = plan.Steps[target]
			saved.Flag = &flag
			ready := old.UpdatedAt.Add(time.Hour)
			saved.StepReadyAt = &ready

			mock.ExpectBegin()
			expectLock(mock, old)
			expectRead(mock, old)
			mock.ExpectQuery(readRolloutPlanForShareQuery).WithArgs(plan.ID).WillReturnRows(rolloutPlanRows(plan))
			if !rollback {
				mock.ExpectQuery(rolloutCanApproveQuery).WithArgs(old.ID).WillReturnRows(pgxmock.NewRows([]string{"ready"}).AddRow(true))
			}
			mock.ExpectExec(saveRolloutQuery).
				WithArgs(old.ID, old.RolloutPlanID, &target, plan.Steps[target], true, plan.IntervalMin, actor).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			expectRead(mock, saved)
			expectHistory(mock, old.ID, "update", &old, &saved, actor).WillReturnResult(pgxmock.NewResult("INSERT", 1))
			expectRevision(mock)
			mock.ExpectCommit()
			got, err := repo.AdvanceRollout(context.Background(), actor, old.ID, old.Version, rollback)
			require.NoError(t, err)
			assert.Equal(t, saved, got)
		})
	}
}

var scheduleColumns = []string{"id", "key_id", "run_at", "enabled", "rollout_percent", "done", "failed", "error", "created_by", "created_at"}

func scheduleRows(s models.OnlineConfSchedule) *pgxmock.Rows {
	return pgxmock.NewRows(scheduleColumns).AddRow(s.ID.String(), s.KeyID.String(), s.RunAt, s.Enabled, s.RolloutPercent, s.Done, s.Failed, s.Error, s.CreatedBy.String(), s.CreatedAt)
}
func TestDueScheduleAlreadyProcessed(t *testing.T) {
	mock, repo := newMock(t)
	key := newFlag()
	id := uuid.NewV4()
	mock.ExpectBegin()
	expectLock(mock, key)
	expectRead(mock, key)
	mock.ExpectQuery(lockDueScheduleQuery).WithArgs(key.ID, id).WillReturnError(pgx.ErrNoRows)
	mock.ExpectCommit()
	require.NoError(t, repo.applyDueSchedule(context.Background(), key.ID, id))
}
func TestDueScheduleCannotBypassNewPlan(t *testing.T) {
	mock, repo := newMock(t)
	key, _ := boundRolloutFlag()
	percent := int32(100)
	s := models.OnlineConfSchedule{ID: uuid.NewV4(), KeyID: key.ID, RunAt: key.UpdatedAt, RolloutPercent: &percent, CreatedBy: uuid.NewV4(), CreatedAt: key.UpdatedAt}
	mock.ExpectBegin()
	expectLock(mock, key)
	expectRead(mock, key)
	mock.ExpectQuery(lockDueScheduleQuery).WithArgs(key.ID, s.ID).WillReturnRows(scheduleRows(s))
	mock.ExpectExec(failScheduleQuery).WithArgs(s.ID, models.ErrOnlineConfRolloutStepNotAllowed.Error()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()
	require.NoError(t, repo.applyDueSchedule(context.Background(), key.ID, s.ID))
}
func TestDueScheduleAtomicApplication(t *testing.T) {
	for _, historyFails := range []bool{false, true} {
		name := "success"
		if historyFails {
			name = "history failure"
		}
		t.Run(name, func(t *testing.T) {
			mock, repo := newMock(t)
			old := newFlag()
			percent := int32(100)
			s := models.OnlineConfSchedule{ID: uuid.NewV4(), KeyID: old.ID, RunAt: old.UpdatedAt, RolloutPercent: &percent, CreatedBy: uuid.NewV4(), CreatedAt: old.UpdatedAt}
			saved := old
			saved.Version++
			saved.UpdatedBy = models.OnlineConfUser{ID: s.CreatedBy, Login: "creator"}
			flag := *old.Flag
			flag.RolloutPercent = percent
			saved.Flag = &flag
			mock.ExpectBegin()
			expectLock(mock, old)
			expectRead(mock, old)
			mock.ExpectQuery(lockDueScheduleQuery).WithArgs(old.ID, s.ID).WillReturnRows(scheduleRows(s))
			mock.ExpectExec(applyScheduleQuery).WithArgs(old.ID, s.Enabled, s.RolloutPercent, s.CreatedBy).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			expectRead(mock, saved)
			history := expectHistory(mock, old.ID, "update", &old, &saved, s.CreatedBy)
			if historyFails {
				history.WillReturnError(errDB)
				mock.ExpectRollback()
			} else {
				history.WillReturnResult(pgxmock.NewResult("INSERT", 1))
				expectRevision(mock)
				mock.ExpectExec(finishScheduleQuery).WithArgs(s.ID).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectCommit()
			}
			err := repo.applyDueSchedule(context.Background(), old.ID, s.ID)
			if historyFails {
				assert.ErrorIs(t, err, errDB)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
func TestSetRolloutPlanStartsAtFirstStep(t *testing.T) {
	mock, repo := newMock(t)
	old := newFlag()
	actor := uuid.NewV4()
	planID := uuid.NewV4()
	plan := models.OnlineConfRolloutPlan{ID: planID, Name: "Basic", Steps: []int32{5, 25, 100}, IntervalMin: 30}
	step := int32(0)
	saved := old
	saved.Version++
	saved.RolloutPlanID = &planID
	saved.RolloutStep = &step
	ready := old.UpdatedAt.Add(30 * time.Minute)
	saved.StepReadyAt = &ready
	saved.UpdatedBy = models.OnlineConfUser{ID: actor, Login: "admin"}
	flag := *old.Flag
	flag.RolloutPercent = 5
	saved.Flag = &flag
	mock.ExpectBegin()
	expectLock(mock, old)
	expectRead(mock, old)
	mock.ExpectQuery(readRolloutPlanForShareQuery).WithArgs(planID).WillReturnRows(rolloutPlanRows(plan))
	mock.ExpectExec(saveRolloutQuery).WithArgs(old.ID, &planID, &step, int32(5), true, int32(30), actor).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	expectRead(mock, saved)
	expectHistory(mock, old.ID, "update", &old, &saved, actor).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	expectRevision(mock)
	mock.ExpectCommit()
	got, err := repo.SetRolloutPlan(context.Background(), actor, old.ID, &planID, old.Version)
	require.NoError(t, err)
	assert.Equal(t, saved, got)
}
func TestRolloutStatus(t *testing.T) {
	for _, status := range []string{onlineconf.RolloutWaiting, onlineconf.RolloutPendingApproval, onlineconf.RolloutDone, onlineconf.RolloutUnassigned} {
		t.Run(status, func(t *testing.T) {
			mock, repo := newMock(t)
			key, plan := boundRolloutFlag()
			stepValue := int32(0)
			var planID, name, steps, interval, step, ready any = &plan.ID, &plan.Name, plan.Steps, &plan.IntervalMin, &stepValue, key.StepReadyAt
			pending := status == onlineconf.RolloutPendingApproval
			if status == onlineconf.RolloutDone {
				stepValue = 2
				key.Flag.RolloutPercent = 100
				ready = nil
			}
			if status == onlineconf.RolloutUnassigned {
				planID = nil
				name = nil
				steps = nil
				interval = nil
				step = nil
				ready = nil
			}
			mock.ExpectQuery(readRolloutStateQuery).WithArgs(key.ID).WillReturnRows(
				pgxmock.NewRows([]string{"id", "kind", "percent", "version", "plan_id", "step", "ready", "pending", "name", "steps", "interval"}).
					AddRow(key.ID.String(), onlineconf.KindFlag, key.Flag.RolloutPercent, key.Version, planID, step, ready, pending, name, steps, interval))
			state, err := repo.ReadRollout(context.Background(), key.ID)
			require.NoError(t, err)
			assert.Equal(t, status, state.Status)
		})
	}
}

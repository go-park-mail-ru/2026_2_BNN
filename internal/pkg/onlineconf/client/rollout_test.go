package client

import (
	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"context"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func TestClientRolloutPlans(t *testing.T) {
	client, uc := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	input := onlineconf.RolloutPlanInput{Name: "Basic", Steps: []int32{5, 100}, IntervalMin: 30}
	plan := models.OnlineConfRolloutPlan{ID: uuid.NewV4(), Name: input.Name, Steps: input.Steps, IntervalMin: input.IntervalMin}
	uc.EXPECT().CreateRolloutPlan(gomock.Any(), input).Return(plan, nil)
	created, err := client.CreateRolloutPlan(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, plan, created)
	uc.EXPECT().ListRolloutPlans(gomock.Any()).Return([]models.OnlineConfRolloutPlan{plan}, nil)
	plans, err := client.ListRolloutPlans(ctx)
	require.NoError(t, err)
	assert.Equal(t, []models.OnlineConfRolloutPlan{plan}, plans)
	uc.EXPECT().UpdateRolloutPlan(gomock.Any(), plan.ID, input).Return(plan, nil)
	updated, err := client.UpdateRolloutPlan(ctx, plan.ID, input)
	require.NoError(t, err)
	assert.Equal(t, plan, updated)
	uc.EXPECT().DeleteRolloutPlan(gomock.Any(), plan.ID).Return(models.ErrOnlineConfRolloutPlanInUse)
	assert.ErrorIs(t, client.DeleteRolloutPlan(ctx, plan.ID), models.ErrOnlineConfRolloutPlanInUse)
}
func TestClientRolloutStateAndErrors(t *testing.T) {
	client, uc := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	keyID, actor := uuid.NewV4(), uuid.NewV4()
	step := int32(0)
	next := int32(25)
	ready := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	state := models.OnlineConfRollout{KeyID: keyID, Plan: &models.OnlineConfRolloutPlan{ID: uuid.NewV4(), Name: "Basic", Steps: []int32{5, 25, 100}, IntervalMin: 30}, Step: &step, CurrentPercent: 5, NextPercent: &next, StepReadyAt: &ready, Status: onlineconf.RolloutPendingApproval, Version: 2}
	uc.EXPECT().GetRollout(gomock.Any(), keyID).Return(state, nil)
	got, err := client.GetRollout(ctx, keyID)
	require.NoError(t, err)
	assert.Equal(t, state, got)
	uc.EXPECT().AdvanceRollout(gomock.Any(), actor, keyID, int32(2), false).Return(models.OnlineConfKey{}, models.ErrOnlineConfRolloutNotReady)
	_, err = client.AdvanceRollout(ctx, actor, keyID, 2, false)
	assert.ErrorIs(t, err, models.ErrOnlineConfRolloutNotReady)
}
func TestClientScheduleOptionalValues(t *testing.T) {
	client, uc := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	actor, keyID := uuid.NewV4(), uuid.NewV4()
	disabled := false
	zero := int32(0)
	runAt := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	input := onlineconf.ScheduleInput{RunAt: runAt, Enabled: &disabled, RolloutPercent: &zero}
	expected := models.OnlineConfSchedule{ID: uuid.NewV4(), KeyID: keyID, RunAt: runAt, Enabled: &disabled, RolloutPercent: &zero, CreatedBy: actor, CreatedAt: time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)}
	uc.EXPECT().CreateSchedule(gomock.Any(), actor, keyID, input).Return(expected, nil)
	got, err := client.CreateSchedule(ctx, actor, keyID, input)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
	uc.EXPECT().ListSchedules(gomock.Any(), keyID).Return([]models.OnlineConfSchedule{expected}, nil)
	schedules, err := client.ListSchedules(ctx, keyID)
	require.NoError(t, err)
	assert.Equal(t, []models.OnlineConfSchedule{expected}, schedules)
}

package usecase

import (
	"context"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRolloutPlanUsecaseCRUD(t *testing.T) {
	repo, _, uc := newUsecase(t)
	ctx := context.Background()
	input := onlineconf.RolloutPlanInput{
		Name: "  Basic  ", Steps: []int32{5, 25, 100}, IntervalMin: 30,
	}
	repo.EXPECT().CreateRolloutPlan(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, plan models.OnlineConfRolloutPlan) (models.OnlineConfRolloutPlan, error) {
			assert.NotEqual(t, uuid.Nil, plan.ID)
			assert.Equal(t, "Basic", plan.Name)
			assert.Equal(t, input.Steps, plan.Steps)
			assert.Equal(t, input.IntervalMin, plan.IntervalMin)
			return plan, nil
		},
	)
	created, err := uc.CreateRolloutPlan(ctx, input)
	require.NoError(t, err)

	repo.EXPECT().ReadRolloutPlans(gomock.Any()).
		Return([]models.OnlineConfRolloutPlan{created}, nil)
	plans, err := uc.ListRolloutPlans(ctx)
	require.NoError(t, err)
	assert.Equal(t, []models.OnlineConfRolloutPlan{created}, plans)

	repo.EXPECT().ReadRolloutPlan(gomock.Any(), created.ID).Return(created, nil)
	found, err := uc.GetRolloutPlan(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created, found)

	input.Name = "  Updated  "
	expected := created
	expected.Name = "Updated"
	repo.EXPECT().UpdateRolloutPlan(gomock.Any(), expected).Return(expected, nil)
	updated, err := uc.UpdateRolloutPlan(ctx, created.ID, input)
	require.NoError(t, err)
	assert.Equal(t, expected, updated)

	repo.EXPECT().DeleteRolloutPlan(gomock.Any(), created.ID).Return(nil)
	require.NoError(t, uc.DeleteRolloutPlan(ctx, created.ID))
}

func TestRolloutPlanUsecaseInvalidID(t *testing.T) {
	_, _, uc := newUsecase(t)
	ctx := context.Background()
	input := onlineconf.RolloutPlanInput{
		Name: "Basic", Steps: []int32{5, 100}, IntervalMin: 30,
	}
	_, err := uc.GetRolloutPlan(ctx, uuid.Nil)
	assert.ErrorIs(t, err, models.ErrValidation)
	_, err = uc.UpdateRolloutPlan(ctx, uuid.Nil, input)
	assert.ErrorIs(t, err, models.ErrValidation)
	err = uc.DeleteRolloutPlan(ctx, uuid.Nil)
	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestRolloutPlanUsecaseInvalidInput(t *testing.T) {
	_, _, uc := newUsecase(t)
	ctx := context.Background()
	input := onlineconf.RolloutPlanInput{
		Name: "Immediate", Steps: []int32{100}, IntervalMin: 30,
	}
	_, err := uc.CreateRolloutPlan(ctx, input)
	assert.ErrorIs(t, err, models.ErrValidation)
	_, err = uc.UpdateRolloutPlan(ctx, uuid.NewV4(), input)
	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestRolloutPlanUsecaseInUse(t *testing.T) {
	repo, _, uc := newUsecase(t)
	ctx := context.Background()
	id := uuid.NewV4()
	input := onlineconf.RolloutPlanInput{
		Name: "Basic", Steps: []int32{5, 100}, IntervalMin: 30,
	}
	repo.EXPECT().UpdateRolloutPlan(gomock.Any(), gomock.Any()).
		Return(models.OnlineConfRolloutPlan{}, models.ErrOnlineConfRolloutPlanInUse)
	_, err := uc.UpdateRolloutPlan(ctx, id, input)
	assert.ErrorIs(t, err, models.ErrOnlineConfRolloutPlanInUse)

	repo.EXPECT().DeleteRolloutPlan(gomock.Any(), id).
		Return(models.ErrOnlineConfRolloutPlanInUse)
	err = uc.DeleteRolloutPlan(ctx, id)
	assert.ErrorIs(t, err, models.ErrOnlineConfRolloutPlanInUse)
}

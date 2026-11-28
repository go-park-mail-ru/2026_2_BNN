package usecase

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

func TestScheduleInputValidation(t *testing.T) {
	now := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	falseValue := false
	for _, tt := range []struct {
		name  string
		input onlineconf.ScheduleInput
		valid bool
	}{
		{"missing change", onlineconf.ScheduleInput{RunAt: now.Add(time.Minute)}, false},
		{"past", onlineconf.ScheduleInput{RunAt: now.Add(-time.Minute), Enabled: &falseValue}, false},
		{"now", onlineconf.ScheduleInput{RunAt: now, Enabled: &falseValue}, false},
		{"disable", onlineconf.ScheduleInput{RunAt: now.Add(time.Minute), Enabled: &falseValue}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScheduleInput(tt.input, now)
			if tt.valid {
				require.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, models.ErrValidation)
			}
		})
	}
	for _, percent := range []int32{-1, 0, 100, 101} {
		err := validateScheduleInput(onlineconf.ScheduleInput{RunAt: now.Add(time.Minute), RolloutPercent: &percent}, now)
		if percent >= 0 && percent <= 100 {
			require.NoError(t, err)
		} else {
			assert.ErrorIs(t, err, models.ErrValidation)
		}
	}
}
func TestCreateSchedulePreservesFalseAndUTC(t *testing.T) {
	repo, _, uc := newUsecase(t)
	actor, keyID := uuid.NewV4(), uuid.NewV4()
	disabled := false
	runAt := time.Now().Add(time.Hour).In(time.FixedZone("MSK", 3*60*60))
	repo.EXPECT().CreateSchedule(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, s models.OnlineConfSchedule) (models.OnlineConfSchedule, error) {
		assert.NotEqual(t, uuid.Nil, s.ID)
		assert.Equal(t, actor, s.CreatedBy)
		assert.Equal(t, keyID, s.KeyID)
		require.NotNil(t, s.Enabled)
		assert.False(t, *s.Enabled)
		assert.True(t, s.RunAt.Equal(runAt))
		assert.Equal(t, time.UTC, s.RunAt.Location())
		return s, nil
	})
	_, err := uc.CreateSchedule(context.Background(), actor, keyID, onlineconf.ScheduleInput{RunAt: runAt, Enabled: &disabled})
	require.NoError(t, err)
}
func TestAdvanceRolloutPublishesSnapshot(t *testing.T) {
	repo, _, uc := newUsecase(t)
	actor, keyID := uuid.NewV4(), uuid.NewV4()
	key := models.OnlineConfKey{ID: keyID, Version: 2}
	repo.EXPECT().AdvanceRollout(gomock.Any(), actor, keyID, int32(1), false).Return(key, nil)
	repo.EXPECT().ReadSnapshot(gomock.Any()).Return(models.Snapshot{Revision: 7, Keys: []models.OnlineConfKey{key}}, nil)
	_, err := uc.AdvanceRollout(context.Background(), actor, keyID, 1, false)
	require.NoError(t, err)
	snapshot, changed, err := uc.GetSnapshot(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, int64(7), snapshot.Revision)
}

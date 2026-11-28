package onlineconf

import (
	"testing"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func flagKey(flag models.OnlineConfFlag) models.OnlineConfKey {
	return models.OnlineConfKey{Key: "f", Flag: &flag}
}

func TestEvaluateFlag(t *testing.T) {
	user := models.OnlineConfUser{ID: uuid.NewV4(), Login: "liza"}
	only := []models.OnlineConfUser{user}

	tests := []struct {
		name       string
		key        models.OnlineConfKey
		wantResult bool
		wantReason string
	}{
		{"key without flag", models.OnlineConfKey{Key: "f"}, false, ReasonDisabled},
		{"disabled wins over include", flagKey(models.OnlineConfFlag{Enabled: false, RolloutPercent: 100, IncludeUsers: only}), false, ReasonDisabled},
		{"exclude wins over include and 100%", flagKey(models.OnlineConfFlag{Enabled: true, RolloutPercent: 100, IncludeUsers: only, ExcludeUsers: only}), false, ReasonExcluded},
		{"include works at 0%", flagKey(models.OnlineConfFlag{Enabled: true, RolloutPercent: 0, IncludeUsers: only}), true, ReasonIncluded},
		{"0% is off for everyone else", flagKey(models.OnlineConfFlag{Enabled: true, RolloutPercent: 0}), false, ReasonRollout},
		{"100% is on for everyone else", flagKey(models.OnlineConfFlag{Enabled: true, RolloutPercent: 100}), true, ReasonRollout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, reason := EvaluateFlag(tt.key, user.ID)
			assert.Equal(t, tt.wantResult, result)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}

func TestEvaluateFlagRollout(t *testing.T) {
	userID := uuid.NewV4()
	b := bucket("f", userID)

	// пользователь включается ровно тогда, когда процент больше его корзины
	off, _ := EvaluateFlag(flagKey(models.OnlineConfFlag{Enabled: true, RolloutPercent: int32(b)}), userID)
	on, _ := EvaluateFlag(flagKey(models.OnlineConfFlag{Enabled: true, RolloutPercent: int32(b + 1)}), userID)

	assert.False(t, off)
	assert.True(t, on)
	assert.Equal(t, b, bucket("f", userID), "bucket must not change between calls")
}

package usecase

import (
	"strings"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRolloutPlanInput(t *testing.T) {
	tests := []struct {
		name     string
		input    onlineconf.RolloutPlanInput
		wantName string
	}{
		{
			name: "regular rollout",
			input: onlineconf.RolloutPlanInput{
				Name:        "  Обычная раскатка  ",
				Steps:       []int32{5, 25, 50, 100},
				IntervalMin: 30,
			},
			wantName: "Обычная раскатка",
		},
		{
			name: "start from zero",
			input: onlineconf.RolloutPlanInput{
				Name:        "С нуля",
				Steps:       []int32{0, 10, 100},
				IntervalMin: 1,
			},
			wantName: "С нуля",
		},
		{
			name: "maximum name length",
			input: onlineconf.RolloutPlanInput{
				Name:        strings.Repeat("я", 100),
				Steps:       []int32{10, 100},
				IntervalMin: 15,
			},
			wantName: strings.Repeat("я", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeRolloutPlanInput(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, tt.input.Steps, got.Steps)
			assert.Equal(t, tt.input.IntervalMin, got.IntervalMin)
		})
	}
}

func TestNormalizeRolloutPlanInputInvalid(t *testing.T) {
	tests := []struct {
		name     string
		planName string
		steps    []int32
		interval int32
	}{
		{
			name:     "empty name",
			planName: "",
			steps:    []int32{5, 100},
			interval: 30,
		},
		{
			name:     "whitespace name",
			planName: " \t\n ",
			steps:    []int32{5, 100},
			interval: 30,
		},
		{
			name:     "long name",
			planName: strings.Repeat("я", 101),
			steps:    []int32{5, 100},
			interval: 30,
		},
		{
			name:     "no steps",
			planName: "Test",
			interval: 30,
		},
		{
			name:     "immediate full rollout",
			planName: "Test",
			steps:    []int32{100},
			interval: 30,
		},
		{
			name:     "negative percent",
			planName: "Test",
			steps:    []int32{-1, 100},
			interval: 30,
		},
		{
			name:     "percent above maximum",
			planName: "Test",
			steps:    []int32{5, 101},
			interval: 30,
		},
		{
			name:     "duplicate steps",
			planName: "Test",
			steps:    []int32{5, 5, 100},
			interval: 30,
		},
		{
			name:     "decreasing steps",
			planName: "Test",
			steps:    []int32{25, 5, 100},
			interval: 30,
		},
		{
			name:     "last step is not full rollout",
			planName: "Test",
			steps:    []int32{5, 50},
			interval: 30,
		},
		{
			name:     "zero interval",
			planName: "Test",
			steps:    []int32{5, 100},
			interval: 0,
		},
		{
			name:     "negative interval",
			planName: "Test",
			steps:    []int32{5, 100},
			interval: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeRolloutPlanInput(
				onlineconf.RolloutPlanInput{
					Name:        tt.planName,
					Steps:       tt.steps,
					IntervalMin: tt.interval,
				},
			)

			assert.ErrorIs(t, err, models.ErrValidation)
		})
	}
}

package models

import (
	uuid "github.com/satori/go.uuid"
	"time"
)

type OnlineConfRolloutPlan struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Steps       []int32   `json:"steps"`
	IntervalMin int32     `json:"interval_min"`
}

type OnlineConfRollout struct {
	KeyID          uuid.UUID              `json:"key_id"`
	Plan           *OnlineConfRolloutPlan `json:"plan"`
	Step           *int32                 `json:"step"`
	CurrentPercent int32                  `json:"current_percent"`
	NextPercent    *int32                 `json:"next_percent"`
	StepReadyAt    *time.Time             `json:"step_ready_at"`
	Status         string                 `json:"status"`
	Version        int32                  `json:"version"`
}

type OnlineConfSchedule struct {
	ID             uuid.UUID `json:"id"`
	KeyID          uuid.UUID `json:"key_id"`
	RunAt          time.Time `json:"run_at"`
	Enabled        *bool     `json:"enabled,omitempty"`
	RolloutPercent *int32    `json:"rollout_percent,omitempty"`
	Done           bool      `json:"done"`
	Failed         bool      `json:"failed"`
	Error          *string   `json:"error"`
	CreatedBy      uuid.UUID `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

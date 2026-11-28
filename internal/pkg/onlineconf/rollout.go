package onlineconf

import "time"

type RolloutPlanInput struct {
	Name        string
	Steps       []int32
	IntervalMin int32
}

type ScheduleInput struct {
	RunAt          time.Time
	Enabled        *bool
	RolloutPercent *int32
}

package models

import "errors"

var (
	ErrOnlineConfRolloutStepNotAllowed = errors.New("rollout step not allowed")
	ErrOnlineConfRolloutNotReady       = errors.New("rollout is not ready for approval")
	ErrOnlineConfScheduleNotFound      = errors.New("onlineconf schedule not found")
	ErrOnlineConfScheduleFinished      = errors.New("onlineconf schedule already finished")
	ErrOnlineConfKeyNotFound           = errors.New("onlineconf key not found")
	ErrOnlineConfKeyExists             = errors.New("onlineconf key already exists")
	ErrOnlineConfVersionConflict       = errors.New("onlineconf key version conflict")
	ErrOnlineConfSnapshotNotReady      = errors.New("onlineconf snapshot is not loaded")
	ErrOnlineConfRolloutPlanNotFound   = errors.New("onlineconf rollout plan not found")
	ErrOnlineConfRolloutPlanExists     = errors.New("onlineconf rollout plan already exists")
	ErrOnlineConfRolloutPlanInUse      = errors.New("onlineconf rollout plan is in use")
)

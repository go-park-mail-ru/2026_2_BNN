package convert

import (
	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

func timePointerToProto(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return grpcconv.TimeToProto(*value)
}
func timePointerFromProto(value *timestamppb.Timestamp) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	if err := value.CheckValid(); err != nil {
		return nil, models.ErrValidation
	}
	t := value.AsTime()
	return &t, nil
}
func PlanToProto(p models.OnlineConfRolloutPlan) *gen.RolloutPlan {
	return &gen.RolloutPlan{Id: p.ID.String(), Name: p.Name, Steps: p.Steps, IntervalMin: p.IntervalMin}
}
func PlanFromProto(p *gen.RolloutPlan) (models.OnlineConfRolloutPlan, error) {
	if p == nil {
		return models.OnlineConfRolloutPlan{}, models.ErrValidation
	}
	id, err := grpcconv.ParseUUID(p.Id)
	if err != nil {
		return models.OnlineConfRolloutPlan{}, err
	}
	return models.OnlineConfRolloutPlan{ID: id, Name: p.Name, Steps: p.Steps, IntervalMin: p.IntervalMin}, nil
}
func PlanInputToProto(p onlineconf.RolloutPlanInput) *gen.RolloutPlanInput {
	return &gen.RolloutPlanInput{Name: p.Name, Steps: p.Steps, IntervalMin: p.IntervalMin}
}
func PlanInputFromProto(p *gen.RolloutPlanInput) (onlineconf.RolloutPlanInput, error) {
	if p == nil {
		return onlineconf.RolloutPlanInput{}, models.ErrValidation
	}
	return onlineconf.RolloutPlanInput{Name: p.Name, Steps: p.Steps, IntervalMin: p.IntervalMin}, nil
}
func RolloutToProto(s models.OnlineConfRollout) *gen.RolloutState {
	p := &gen.RolloutState{KeyId: s.KeyID.String(), Step: s.Step, CurrentPercent: s.CurrentPercent, NextPercent: s.NextPercent, StepReadyAt: timePointerToProto(s.StepReadyAt), Status: s.Status, Version: s.Version}
	if s.Plan != nil {
		p.Plan = PlanToProto(*s.Plan)
	}
	return p
}
func RolloutFromProto(p *gen.RolloutState) (models.OnlineConfRollout, error) {
	if p == nil {
		return models.OnlineConfRollout{}, models.ErrValidation
	}
	s := models.OnlineConfRollout{Step: p.Step, CurrentPercent: p.CurrentPercent, NextPercent: p.NextPercent, Status: p.Status, Version: p.Version}
	var err error
	s.KeyID, err = grpcconv.ParseUUID(p.KeyId)
	if err != nil {
		return s, err
	}
	s.StepReadyAt, err = timePointerFromProto(p.StepReadyAt)
	if err != nil {
		return s, err
	}
	if p.Plan != nil {
		plan, err := PlanFromProto(p.Plan)
		if err != nil {
			return s, err
		}
		s.Plan = &plan
	}
	return s, nil
}
func ScheduleToProto(s models.OnlineConfSchedule) *gen.Schedule {
	return &gen.Schedule{Id: s.ID.String(), KeyId: s.KeyID.String(), RunAt: grpcconv.TimeToProto(s.RunAt), Enabled: s.Enabled, RolloutPercent: s.RolloutPercent, Done: s.Done, Failed: s.Failed, Error: s.Error, CreatedBy: s.CreatedBy.String(), CreatedAt: grpcconv.TimeToProto(s.CreatedAt)}
}
func ScheduleFromProto(p *gen.Schedule) (models.OnlineConfSchedule, error) {
	if p == nil {
		return models.OnlineConfSchedule{}, models.ErrValidation
	}
	s := models.OnlineConfSchedule{Enabled: p.Enabled, RolloutPercent: p.RolloutPercent, Done: p.Done, Failed: p.Failed, Error: p.Error}
	var err error
	s.ID, err = grpcconv.ParseUUID(p.Id)
	if err != nil {
		return s, err
	}
	s.KeyID, err = grpcconv.ParseUUID(p.KeyId)
	if err != nil {
		return s, err
	}
	s.CreatedBy, err = grpcconv.ParseUUID(p.CreatedBy)
	if err != nil {
		return s, err
	}
	runAt, err := timePointerFromProto(p.RunAt)
	if err != nil {
		return s, err
	}
	if runAt == nil {
		return s, models.ErrValidation
	}
	s.RunAt = *runAt
	createdAt, err := timePointerFromProto(p.CreatedAt)
	if err != nil {
		return s, err
	}
	if createdAt == nil {
		return s, models.ErrValidation
	}
	s.CreatedAt = *createdAt
	return s, nil
}
func ScheduleInputFromProto(p *gen.ScheduleInput) (onlineconf.ScheduleInput, error) {
	if p == nil {
		return onlineconf.ScheduleInput{}, models.ErrValidation
	}
	t, err := timePointerFromProto(p.RunAt)
	if err != nil {
		return onlineconf.ScheduleInput{}, err
	}
	if t == nil {
		return onlineconf.ScheduleInput{}, models.ErrValidation
	}
	return onlineconf.ScheduleInput{RunAt: *t, Enabled: p.Enabled, RolloutPercent: p.RolloutPercent}, nil
}

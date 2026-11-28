package grpc

import (
	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf/delivery/grpc/convert"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"
	"bnn/internal/pkg/utils/grpcerr"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (h *Handler) ListRolloutPlans(ctx context.Context, in *emptypb.Empty) (*gen.RolloutPlansResponse, error) {
	plans, err := h.uc.ListRolloutPlans(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	result := &gen.RolloutPlansResponse{Plans: make([]*gen.RolloutPlan, 0, len(plans))}
	for _, p := range plans {
		result.Plans = append(result.Plans, convert.PlanToProto(p))
	}
	return result, nil
}
func (h *Handler) GetRolloutPlan(ctx context.Context, in *gen.RolloutPlanRequest) (*gen.RolloutPlan, error) {
	id, err := grpcconv.ParseUUID(in.GetId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	p, err := h.uc.GetRolloutPlan(ctx, id)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return convert.PlanToProto(p), nil
}
func (h *Handler) CreateRolloutPlan(ctx context.Context, in *gen.RolloutPlanInput) (*gen.RolloutPlan, error) {
	input, err := convert.PlanInputFromProto(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	p, err := h.uc.CreateRolloutPlan(ctx, input)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return convert.PlanToProto(p), nil
}
func (h *Handler) UpdateRolloutPlan(ctx context.Context, in *gen.RolloutPlanRequest) (*gen.RolloutPlan, error) {
	id, err := grpcconv.ParseUUID(in.GetId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	input, err := convert.PlanInputFromProto(in.GetInput())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	p, err := h.uc.UpdateRolloutPlan(ctx, id, input)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return convert.PlanToProto(p), nil
}
func (h *Handler) DeleteRolloutPlan(ctx context.Context, in *gen.RolloutPlanRequest) (*emptypb.Empty, error) {
	id, err := grpcconv.ParseUUID(in.GetId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	if err := h.uc.DeleteRolloutPlan(ctx, id); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return &emptypb.Empty{}, nil
}
func (h *Handler) SetRolloutPlan(ctx context.Context, in *gen.SetRolloutPlanRequest) (*gen.Key, error) {
	if in == nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}
	actor, err := grpcconv.ParseUUID(in.UserId)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	id, err := grpcconv.ParseUUID(in.KeyId)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	planID, err := grpcconv.ParseOptionalUUID(in.PlanId)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	k, err := h.uc.SetRolloutPlan(ctx, actor, id, planID, in.Version)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return convert.KeyToProto(k), nil
}
func (h *Handler) GetRollout(ctx context.Context, in *gen.RolloutKeyRequest) (*gen.RolloutState, error) {
	id, err := grpcconv.ParseUUID(in.GetKeyId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	s, err := h.uc.GetRollout(ctx, id)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return convert.RolloutToProto(s), nil
}
func (h *Handler) AdvanceRollout(ctx context.Context, in *gen.RolloutChangeRequest) (*gen.Key, error) {
	actor, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	id, err := grpcconv.ParseUUID(in.GetKeyId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	k, err := h.uc.AdvanceRollout(ctx, actor, id, in.GetVersion(), in.GetRollback())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return convert.KeyToProto(k), nil
}
func (h *Handler) ListSchedules(ctx context.Context, in *gen.RolloutKeyRequest) (*gen.SchedulesResponse, error) {
	id, err := grpcconv.ParseUUID(in.GetKeyId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	schedules, err := h.uc.ListSchedules(ctx, id)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	out := &gen.SchedulesResponse{Schedules: make([]*gen.Schedule, 0, len(schedules))}
	for _, s := range schedules {
		out.Schedules = append(out.Schedules, convert.ScheduleToProto(s))
	}
	return out, nil
}
func (h *Handler) CreateSchedule(ctx context.Context, in *gen.CreateScheduleRequest) (*gen.Schedule, error) {
	actor, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	id, err := grpcconv.ParseUUID(in.GetKeyId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	input, err := convert.ScheduleInputFromProto(in.GetInput())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	s, err := h.uc.CreateSchedule(ctx, actor, id, input)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return convert.ScheduleToProto(s), nil
}
func (h *Handler) DeleteSchedule(ctx context.Context, in *gen.DeleteScheduleRequest) (*emptypb.Empty, error) {
	id, err := grpcconv.ParseUUID(in.GetKeyId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	scheduleID, err := grpcconv.ParseUUID(in.GetScheduleId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	if err := h.uc.DeleteSchedule(ctx, id, scheduleID); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return &emptypb.Empty{}, nil
}

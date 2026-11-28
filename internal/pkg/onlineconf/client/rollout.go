package client

import (
	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/grpc/convert"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"
	"bnn/internal/pkg/utils/grpcerr"
	"context"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (c *Client) ListRolloutPlans(ctx context.Context) ([]models.OnlineConfRolloutPlan, error) {
	response, err := c.client.ListRolloutPlans(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}
	plans := make([]models.OnlineConfRolloutPlan, 0, len(response.Plans))
	for _, p := range response.Plans {
		plan, err := convert.PlanFromProto(p)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, nil
}
func (c *Client) GetRolloutPlan(ctx context.Context, id uuid.UUID) (models.OnlineConfRolloutPlan, error) {
	p, err := c.client.GetRolloutPlan(ctx, &gen.RolloutPlanRequest{Id: id.String()})
	if err != nil {
		return models.OnlineConfRolloutPlan{}, grpcerr.FromStatus(err)
	}
	return convert.PlanFromProto(p)
}
func (c *Client) CreateRolloutPlan(ctx context.Context, input onlineconf.RolloutPlanInput) (models.OnlineConfRolloutPlan, error) {
	p, err := c.client.CreateRolloutPlan(ctx, convert.PlanInputToProto(input))
	if err != nil {
		return models.OnlineConfRolloutPlan{}, grpcerr.FromStatus(err)
	}
	return convert.PlanFromProto(p)
}
func (c *Client) UpdateRolloutPlan(ctx context.Context, id uuid.UUID, input onlineconf.RolloutPlanInput) (models.OnlineConfRolloutPlan, error) {
	p, err := c.client.UpdateRolloutPlan(ctx, &gen.RolloutPlanRequest{Id: id.String(), Input: convert.PlanInputToProto(input)})
	if err != nil {
		return models.OnlineConfRolloutPlan{}, grpcerr.FromStatus(err)
	}
	return convert.PlanFromProto(p)
}
func (c *Client) DeleteRolloutPlan(ctx context.Context, id uuid.UUID) error {
	_, err := c.client.DeleteRolloutPlan(ctx, &gen.RolloutPlanRequest{Id: id.String()})
	return grpcerr.FromStatus(err)
}
func (c *Client) SetRolloutPlan(ctx context.Context, actor, keyID uuid.UUID, planID *uuid.UUID, version int32) (models.OnlineConfKey, error) {
	k, err := c.client.SetRolloutPlan(ctx, &gen.SetRolloutPlanRequest{UserId: actor.String(), KeyId: keyID.String(), PlanId: grpcconv.OptionalUUIDToString(planID), Version: version})
	if err != nil {
		return models.OnlineConfKey{}, grpcerr.FromStatus(err)
	}
	return convert.KeyFromProto(k)
}
func (c *Client) GetRollout(ctx context.Context, keyID uuid.UUID) (models.OnlineConfRollout, error) {
	s, err := c.client.GetRollout(ctx, &gen.RolloutKeyRequest{KeyId: keyID.String()})
	if err != nil {
		return models.OnlineConfRollout{}, grpcerr.FromStatus(err)
	}
	return convert.RolloutFromProto(s)
}
func (c *Client) AdvanceRollout(ctx context.Context, actor, keyID uuid.UUID, version int32, rollback bool) (models.OnlineConfKey, error) {
	k, err := c.client.AdvanceRollout(ctx, &gen.RolloutChangeRequest{UserId: actor.String(), KeyId: keyID.String(), Version: version, Rollback: rollback})
	if err != nil {
		return models.OnlineConfKey{}, grpcerr.FromStatus(err)
	}
	return convert.KeyFromProto(k)
}
func (c *Client) ListSchedules(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfSchedule, error) {
	response, err := c.client.ListSchedules(ctx, &gen.RolloutKeyRequest{KeyId: keyID.String()})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}
	schedules := make([]models.OnlineConfSchedule, 0, len(response.Schedules))
	for _, p := range response.Schedules {
		s, err := convert.ScheduleFromProto(p)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}
func (c *Client) CreateSchedule(ctx context.Context, actor, keyID uuid.UUID, input onlineconf.ScheduleInput) (models.OnlineConfSchedule, error) {
	s, err := c.client.CreateSchedule(ctx, &gen.CreateScheduleRequest{UserId: actor.String(), KeyId: keyID.String(), Input: &gen.ScheduleInput{RunAt: grpcconv.TimeToProto(input.RunAt), Enabled: input.Enabled, RolloutPercent: input.RolloutPercent}})
	if err != nil {
		return models.OnlineConfSchedule{}, grpcerr.FromStatus(err)
	}
	return convert.ScheduleFromProto(s)
}
func (c *Client) DeleteSchedule(ctx context.Context, keyID, scheduleID uuid.UUID) error {
	_, err := c.client.DeleteSchedule(ctx, &gen.DeleteScheduleRequest{KeyId: keyID.String(), ScheduleId: scheduleID.String()})
	return grpcerr.FromStatus(err)
}

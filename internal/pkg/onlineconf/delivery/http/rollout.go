package delivery

import (
	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/utils/ctxutil"
	"context"
	"encoding/json"
	"fmt"
	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
)

func (h *Handler) ListOnlineConfRolloutPlans(ctx context.Context, req api.ListOnlineConfRolloutPlansRequestObject) (api.ListOnlineConfRolloutPlansResponseObject, error) {
	_, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.ListOnlineConfRolloutPlans401Response{}, nil
	}
	fail := func(err error) (api.ListOnlineConfRolloutPlansResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.ListOnlineConfRolloutPlansdefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	plans, err := h.keys.ListRolloutPlans(ctx)
	if err != nil {
		return fail(err)
	}
	response := make(api.ListOnlineConfRolloutPlans200JSONResponse, 0, len(plans))
	for _, p := range plans {
		response = append(response, toAPIPlan(p))
	}
	return response, nil
}
func (h *Handler) GetOnlineConfRolloutPlan(ctx context.Context, req api.GetOnlineConfRolloutPlanRequestObject) (api.GetOnlineConfRolloutPlanResponseObject, error) {
	_, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.GetOnlineConfRolloutPlan401Response{}, nil
	}
	fail := func(err error) (api.GetOnlineConfRolloutPlanResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.GetOnlineConfRolloutPlandefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	plan, err := h.keys.GetRolloutPlan(ctx, uuid.UUID(req.Id))
	if err != nil {
		return fail(err)
	}
	return api.GetOnlineConfRolloutPlan200JSONResponse(toAPIPlan(plan)), nil
}
func (h *Handler) CreateOnlineConfRolloutPlan(ctx context.Context, req api.CreateOnlineConfRolloutPlanRequestObject) (api.CreateOnlineConfRolloutPlanResponseObject, error) {
	_, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.CreateOnlineConfRolloutPlan401Response{}, nil
	}
	fail := func(err error) (api.CreateOnlineConfRolloutPlanResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.CreateOnlineConfRolloutPlandefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	if req.Body == nil {
		return fail(fmt.Errorf("%w: request body is required", models.ErrValidation))
	}
	plan, err := h.keys.CreateRolloutPlan(ctx, planInput(*req.Body))
	if err != nil {
		return fail(err)
	}
	return api.CreateOnlineConfRolloutPlan201JSONResponse(toAPIPlan(plan)), nil
}
func (h *Handler) UpdateOnlineConfRolloutPlan(ctx context.Context, req api.UpdateOnlineConfRolloutPlanRequestObject) (api.UpdateOnlineConfRolloutPlanResponseObject, error) {
	_, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.UpdateOnlineConfRolloutPlan401Response{}, nil
	}
	fail := func(err error) (api.UpdateOnlineConfRolloutPlanResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.UpdateOnlineConfRolloutPlandefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	if req.Body == nil {
		return fail(fmt.Errorf("%w: request body is required", models.ErrValidation))
	}
	plan, err := h.keys.UpdateRolloutPlan(ctx, uuid.UUID(req.Id), planInput(*req.Body))
	if err != nil {
		return fail(err)
	}
	return api.UpdateOnlineConfRolloutPlan200JSONResponse(toAPIPlan(plan)), nil
}
func (h *Handler) DeleteOnlineConfRolloutPlan(ctx context.Context, req api.DeleteOnlineConfRolloutPlanRequestObject) (api.DeleteOnlineConfRolloutPlanResponseObject, error) {
	_, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.DeleteOnlineConfRolloutPlan401Response{}, nil
	}
	fail := func(err error) (api.DeleteOnlineConfRolloutPlanResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.DeleteOnlineConfRolloutPlandefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	err := h.keys.DeleteRolloutPlan(ctx, uuid.UUID(req.Id))
	if err != nil {
		return fail(err)
	}
	return api.DeleteOnlineConfRolloutPlan204Response{}, nil
}
func (h *Handler) SetOnlineConfRolloutPlan(ctx context.Context, req api.SetOnlineConfRolloutPlanRequestObject) (api.SetOnlineConfRolloutPlanResponseObject, error) {
	actor, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.SetOnlineConfRolloutPlan401Response{}, nil
	}
	fail := func(err error) (api.SetOnlineConfRolloutPlanResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.SetOnlineConfRolloutPlandefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	if req.Body == nil {
		return fail(fmt.Errorf("%w: request body is required", models.ErrValidation))
	}
	if len(req.Body.PlanId) == 0 {
		return fail(fmt.Errorf("%w: plan_id is required", models.ErrValidation))
	}
	var planID *uuid.UUID
	if err := json.Unmarshal(req.Body.PlanId, &planID); err != nil {
		return fail(fmt.Errorf("%w: invalid plan_id", models.ErrValidation))
	}
	key, err := h.keys.SetRolloutPlan(ctx, actor, uuid.UUID(req.Id), planID, req.Body.Version)
	if err != nil {
		return fail(err)
	}
	return api.SetOnlineConfRolloutPlan200JSONResponse(toAPIKey(key, true)), nil
}
func (h *Handler) GetOnlineConfRollout(ctx context.Context, req api.GetOnlineConfRolloutRequestObject) (api.GetOnlineConfRolloutResponseObject, error) {
	_, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.GetOnlineConfRollout401Response{}, nil
	}
	fail := func(err error) (api.GetOnlineConfRolloutResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.GetOnlineConfRolloutdefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	state, err := h.keys.GetRollout(ctx, uuid.UUID(req.Id))
	if err != nil {
		return fail(err)
	}
	return api.GetOnlineConfRollout200JSONResponse(toAPIRollout(state)), nil
}
func (h *Handler) ApproveOnlineConfRollout(ctx context.Context, req api.ApproveOnlineConfRolloutRequestObject) (api.ApproveOnlineConfRolloutResponseObject, error) {
	actor, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.ApproveOnlineConfRollout401Response{}, nil
	}
	fail := func(err error) (api.ApproveOnlineConfRolloutResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.ApproveOnlineConfRolloutdefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	if req.Body == nil {
		return fail(fmt.Errorf("%w: request body is required", models.ErrValidation))
	}
	key, err := h.keys.AdvanceRollout(ctx, actor, uuid.UUID(req.Id), req.Body.Version, false)
	if err != nil {
		return fail(err)
	}
	return api.ApproveOnlineConfRollout200JSONResponse(toAPIKey(key, true)), nil
}
func (h *Handler) RollbackOnlineConfRollout(ctx context.Context, req api.RollbackOnlineConfRolloutRequestObject) (api.RollbackOnlineConfRolloutResponseObject, error) {
	actor, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.RollbackOnlineConfRollout401Response{}, nil
	}
	fail := func(err error) (api.RollbackOnlineConfRolloutResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.RollbackOnlineConfRolloutdefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	if req.Body == nil {
		return fail(fmt.Errorf("%w: request body is required", models.ErrValidation))
	}
	key, err := h.keys.AdvanceRollout(ctx, actor, uuid.UUID(req.Id), req.Body.Version, true)
	if err != nil {
		return fail(err)
	}
	return api.RollbackOnlineConfRollout200JSONResponse(toAPIKey(key, true)), nil
}
func (h *Handler) ListOnlineConfSchedules(ctx context.Context, req api.ListOnlineConfSchedulesRequestObject) (api.ListOnlineConfSchedulesResponseObject, error) {
	_, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.ListOnlineConfSchedules401Response{}, nil
	}
	fail := func(err error) (api.ListOnlineConfSchedulesResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.ListOnlineConfSchedulesdefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	schedules, err := h.keys.ListSchedules(ctx, uuid.UUID(req.Id))
	if err != nil {
		return fail(err)
	}
	response := make(api.ListOnlineConfSchedules200JSONResponse, 0, len(schedules))
	for _, s := range schedules {
		response = append(response, toAPISchedule(s))
	}
	return response, nil
}
func (h *Handler) CreateOnlineConfSchedule(ctx context.Context, req api.CreateOnlineConfScheduleRequestObject) (api.CreateOnlineConfScheduleResponseObject, error) {
	actor, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.CreateOnlineConfSchedule401Response{}, nil
	}
	fail := func(err error) (api.CreateOnlineConfScheduleResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.CreateOnlineConfScheduledefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	if req.Body == nil {
		return fail(fmt.Errorf("%w: request body is required", models.ErrValidation))
	}
	schedule, err := h.keys.CreateSchedule(ctx, actor, uuid.UUID(req.Id), onlineconf.ScheduleInput{RunAt: req.Body.RunAt, Enabled: req.Body.Enabled, RolloutPercent: req.Body.RolloutPercent})
	if err != nil {
		return fail(err)
	}
	return api.CreateOnlineConfSchedule201JSONResponse(toAPISchedule(schedule)), nil
}
func (h *Handler) DeleteOnlineConfSchedule(ctx context.Context, req api.DeleteOnlineConfScheduleRequestObject) (api.DeleteOnlineConfScheduleResponseObject, error) {
	_, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.DeleteOnlineConfSchedule401Response{}, nil
	}
	fail := func(err error) (api.DeleteOnlineConfScheduleResponseObject, error) {
		status, body := keyHTTPError(ctx, err)
		return api.DeleteOnlineConfScheduledefaultJSONResponse{StatusCode: status, Body: body}, nil
	}
	err := h.keys.DeleteSchedule(ctx, uuid.UUID(req.Id), uuid.UUID(req.ScheduleId))
	if err != nil {
		return fail(err)
	}
	return api.DeleteOnlineConfSchedule204Response{}, nil
}

func planInput(p api.OnlineConfRolloutPlanInput) onlineconf.RolloutPlanInput {
	return onlineconf.RolloutPlanInput{Name: p.Name, Steps: p.Steps, IntervalMin: p.IntervalMin}
}
func toAPIPlan(p models.OnlineConfRolloutPlan) api.OnlineConfRolloutPlan {
	return api.OnlineConfRolloutPlan{Id: openapi_types.UUID(p.ID), Name: p.Name, Steps: p.Steps, IntervalMin: p.IntervalMin}
}
func toAPIRollout(s models.OnlineConfRollout) api.OnlineConfRollout {
	out := api.OnlineConfRollout{KeyId: openapi_types.UUID(s.KeyID), Step: s.Step, CurrentPercent: s.CurrentPercent, NextPercent: s.NextPercent, StepReadyAt: s.StepReadyAt, Status: s.Status, Version: s.Version}
	if s.Plan != nil {
		plan := toAPIPlan(*s.Plan)
		out.Plan = &plan
	}
	return out
}
func toAPISchedule(s models.OnlineConfSchedule) api.OnlineConfSchedule {
	return api.OnlineConfSchedule{Id: openapi_types.UUID(s.ID), KeyId: openapi_types.UUID(s.KeyID), RunAt: s.RunAt, Enabled: s.Enabled, RolloutPercent: s.RolloutPercent, Done: s.Done, Failed: s.Failed, Error: s.Error, CreatedBy: openapi_types.UUID(s.CreatedBy), CreatedAt: s.CreatedAt}
}

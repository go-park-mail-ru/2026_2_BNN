package delivery

import (
	"context"
	"errors"

	"bnn/internal/models"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/tags"
	"bnn/internal/pkg/tags/delivery/http/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
)

type Handler struct {
	uc tags.Usecase
}

func NewHandler(uc tags.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ListTags(ctx context.Context, req api.ListTagsRequestObject) (api.ListTagsResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.ListTags401Response{}, nil
	}

	list, err := h.uc.ListTags(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := make(api.ListTags200JSONResponse, 0, len(list))
	for _, tag := range list {
		resp = append(resp, toAPITag(tag))
	}

	return resp, nil
}

func (h *Handler) CreateTag(ctx context.Context, req api.CreateTagRequestObject) (api.CreateTagResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.CreateTag401Response{}, nil
	}

	tag, err := h.uc.CreateTag(ctx, userID, req.Body.Name, req.Body.Color)
	if errors.Is(err, models.ErrValidation) {
		return api.CreateTag400Response{}, nil
	}
	if errors.Is(err, models.ErrTagExists) {
		return api.CreateTag409Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.CreateTag201JSONResponse(toAPITag(tag)), nil
}

func (h *Handler) UpdateTag(ctx context.Context, req api.UpdateTagRequestObject) (api.UpdateTagResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.UpdateTag401Response{}, nil
	}

	tag, err := h.uc.UpdateTag(ctx, userID, uuid.UUID(req.Id), req.Body.Name, req.Body.Color)
	if errors.Is(err, models.ErrValidation) {
		return api.UpdateTag400Response{}, nil
	}
	if errors.Is(err, models.ErrTagNotFound) {
		return api.UpdateTag404Response{}, nil
	}
	if errors.Is(err, models.ErrTagExists) {
		return api.UpdateTag409Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.UpdateTag200JSONResponse(toAPITag(tag)), nil
}

func (h *Handler) DeleteTag(ctx context.Context, req api.DeleteTagRequestObject) (api.DeleteTagResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.DeleteTag401Response{}, nil
	}

	err := h.uc.DeleteTag(ctx, userID, uuid.UUID(req.Id))
	if errors.Is(err, models.ErrTagNotFound) {
		return api.DeleteTag404Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.DeleteTag204Response{}, nil
}

func toAPITag(tag models.Tag) api.Tag {
	return api.Tag{
		Id:        openapi_types.UUID(tag.ID),
		Name:      tag.Name,
		Color:     tag.Color,
		CreatedAt: tag.CreatedAt,
		UpdatedAt: tag.UpdatedAt,
	}
}

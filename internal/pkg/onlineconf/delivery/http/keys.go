package delivery

import (
	"context"
	"fmt"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/utils/ctxutil"

	uuid "github.com/satori/go.uuid"
)

func (h *Handler) ListOnlineConfScopes(
	ctx context.Context,
	req api.ListOnlineConfScopesRequestObject,
) (api.ListOnlineConfScopesResponseObject, error) {
	if _, ok := ctxutil.GetUserID(ctx); !ok {
		return api.ListOnlineConfScopes401Response{}, nil
	}

	scopes := h.keys.ListScopes()
	response := make(api.ListOnlineConfScopes200JSONResponse, 0, len(scopes))

	for _, scope := range scopes {
		response = append(response, api.OnlineConfScope(scope))
	}

	return response, nil
}

func (h *Handler) ListOnlineConfKeys(
	ctx context.Context,
	req api.ListOnlineConfKeysRequestObject,
) (api.ListOnlineConfKeysResponseObject, error) {
	if _, ok := ctxutil.GetUserID(ctx); !ok {
		return api.ListOnlineConfKeys401Response{}, nil
	}

	var query, scope, kind string

	if req.Params.Query != nil {
		query = *req.Params.Query
	}
	if req.Params.Scope != nil {
		scope = string(*req.Params.Scope)
	}
	if req.Params.Kind != nil {
		kind = string(*req.Params.Kind)
	}

	keys, err := h.keys.ListKeys(ctx, query, scope, kind)
	if err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.ListOnlineConfKeysdefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	response := make(api.ListOnlineConfKeys200JSONResponse, 0, len(keys))
	for _, key := range keys {
		response = append(response, toAPIKey(key, false))
	}

	return response, nil
}

func (h *Handler) GetOnlineConfKey(
	ctx context.Context,
	req api.GetOnlineConfKeyRequestObject,
) (api.GetOnlineConfKeyResponseObject, error) {
	if _, ok := ctxutil.GetUserID(ctx); !ok {
		return api.GetOnlineConfKey401Response{}, nil
	}

	key, err := h.keys.GetKey(ctx, uuid.UUID(req.Id))
	if err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.GetOnlineConfKeydefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	return api.GetOnlineConfKey200JSONResponse(toAPIKey(key, true)), nil
}

func (h *Handler) CreateOnlineConfKey(
	ctx context.Context,
	req api.CreateOnlineConfKeyRequestObject,
) (api.CreateOnlineConfKeyResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.CreateOnlineConfKey401Response{}, nil
	}

	if req.Body == nil {
		statusCode, body := keyHTTPError(ctx, fmt.Errorf(
			"%w: request body is required", models.ErrValidation,
		))
		return api.CreateOnlineConfKeydefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	input, err := toKeyInput(*req.Body)
	if err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.CreateOnlineConfKeydefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	key, err := h.keys.CreateKey(ctx, userID, input)
	if err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.CreateOnlineConfKeydefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	return api.CreateOnlineConfKey201JSONResponse(toAPIKey(key, true)), nil
}

func (h *Handler) UpdateOnlineConfKey(
	ctx context.Context,
	req api.UpdateOnlineConfKeyRequestObject,
) (api.UpdateOnlineConfKeyResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.UpdateOnlineConfKey401Response{}, nil
	}

	if req.Body == nil || req.Body.Version < 1 {
		statusCode, body := keyHTTPError(ctx, fmt.Errorf(
			"%w: positive version is required", models.ErrValidation,
		))
		return api.UpdateOnlineConfKeydefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	input, err := toKeyInput(api.OnlineConfKeyInput{
		Scope:       req.Body.Scope,
		Key:         req.Body.Key,
		Kind:        req.Body.Kind,
		Description: req.Body.Description,
		IsPublic:    req.Body.IsPublic,
		Config:      req.Body.Config,
		Flag:        req.Body.Flag,
	})
	if err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.UpdateOnlineConfKeydefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	key, err := h.keys.UpdateKey(
		ctx, userID, uuid.UUID(req.Id), req.Body.Version, input,
	)
	if err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.UpdateOnlineConfKeydefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	return api.UpdateOnlineConfKey200JSONResponse(toAPIKey(key, true)), nil
}

func (h *Handler) DeleteOnlineConfKey(
	ctx context.Context,
	req api.DeleteOnlineConfKeyRequestObject,
) (api.DeleteOnlineConfKeyResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.DeleteOnlineConfKey401Response{}, nil
	}

	if req.Params.Version < 1 {
		statusCode, body := keyHTTPError(ctx, fmt.Errorf(
			"%w: positive version is required", models.ErrValidation,
		))
		return api.DeleteOnlineConfKeydefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	err := h.keys.DeleteKey(
		ctx, userID, uuid.UUID(req.Id), req.Params.Version,
	)
	if err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.DeleteOnlineConfKeydefaultJSONResponse{
			StatusCode: statusCode, Body: body,
		}, nil
	}

	return api.DeleteOnlineConfKey204Response{}, nil
}

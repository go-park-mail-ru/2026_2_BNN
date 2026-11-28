package delivery

import (
	"context"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/utils/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
)

func (h *Handler) ListOnlineConfHistory(
	ctx context.Context,
	req api.ListOnlineConfHistoryRequestObject,
) (api.ListOnlineConfHistoryResponseObject, error) {
	if _, ok := ctxutil.GetUserID(ctx); !ok {
		return api.ListOnlineConfHistory401Response{}, nil
	}

	history, err := h.keys.ListHistory(ctx, uuid.UUID(req.Id))
	if err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.ListOnlineConfHistorydefaultJSONResponse{
			StatusCode: statusCode,
			Body:       body,
		}, nil
	}

	response := make(
		api.ListOnlineConfHistory200JSONResponse, 0, len(history),
	)

	for _, entry := range history {
		response = append(response, toAPIHistory(entry))
	}

	return response, nil
}

func toAPIHistory(
	entry models.OnlineConfHistory,
) api.OnlineConfHistoryEntry {
	result := api.OnlineConfHistoryEntry{
		Id:     openapi_types.UUID(entry.ID),
		Action: api.OnlineConfHistoryEntryAction(entry.Action),
		User: toAPIUser(models.User{
			ID:    entry.User.ID,
			Login: entry.User.Login,
		}),
		CreatedAt: entry.CreatedAt,
	}

	if entry.OldValue != nil {
		key := toAPIKey(*entry.OldValue, true)
		result.OldValue = &key
	}
	if entry.NewValue != nil {
		key := toAPIKey(*entry.NewValue, true)
		result.NewValue = &key
	}

	return result
}

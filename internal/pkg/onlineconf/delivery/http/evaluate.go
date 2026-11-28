package delivery

import (
	"context"
	"fmt"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/utils/ctxutil"

	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
)

// EvaluateOnlineConfKey считает флаг для пользователя по тому же снимку и той же функцией, что и продукт.
func (h *Handler) EvaluateOnlineConfKey(
	ctx context.Context,
	req api.EvaluateOnlineConfKeyRequestObject,
) (api.EvaluateOnlineConfKeyResponseObject, error) {
	if _, ok := ctxutil.GetUserID(ctx); !ok {
		return api.EvaluateOnlineConfKey401Response{}, nil
	}

	if req.Body == nil {
		return evaluateError(ctx, fmt.Errorf("%w: user_id is required", models.ErrValidation)), nil
	}

	if uuid.UUID(req.Body.UserId) == uuid.Nil {
		return evaluateError(ctx, fmt.Errorf("%w: user_id is required", models.ErrValidation)), nil
	}

	// снимок берём один раз, чтобы ключ и revision были из одной версии
	snapshot := h.snapshot.Get()

	key, ok := lo.Find(snapshot.Keys, func(k models.OnlineConfKey) bool {
		return k.ID == uuid.UUID(req.Id)
	})
	if !ok {
		return evaluateError(ctx, models.ErrOnlineConfKeyNotFound), nil
	}

	if key.Flag == nil {
		return evaluateError(ctx, fmt.Errorf("%w: key is not a flag", models.ErrValidation)), nil
	}

	result, reason := onlineconf.EvaluateFlag(key, uuid.UUID(req.Body.UserId))

	return api.EvaluateOnlineConfKey200JSONResponse{
		Result:   result,
		Reason:   api.OnlineConfEvaluationReason(reason),
		Revision: snapshot.Revision,
	}, nil
}

func evaluateError(ctx context.Context, err error) api.EvaluateOnlineConfKeydefaultJSONResponse {
	statusCode, body := keyHTTPError(ctx, err)
	return api.EvaluateOnlineConfKeydefaultJSONResponse{StatusCode: statusCode, Body: body}
}

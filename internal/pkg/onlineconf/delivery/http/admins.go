package delivery

import (
	"context"
	"fmt"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/utils/ctxutil"

	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
)

func (h *Handler) ListOnlineConfAdmins(
	ctx context.Context,
	req api.ListOnlineConfAdminsRequestObject,
) (api.ListOnlineConfAdminsResponseObject, error) {
	if _, ok := ctxutil.GetUserID(ctx); !ok {
		return api.ListOnlineConfAdmins401Response{}, nil
	}

	return api.ListOnlineConfAdmins200JSONResponse(lo.Map(h.admins.List(), func(u models.User, _ int) api.OnlineConfUser {
		return toAPIUser(u)
	})), nil
}

func (h *Handler) GrantOnlineConfAdmin(
	ctx context.Context,
	req api.GrantOnlineConfAdminRequestObject,
) (api.GrantOnlineConfAdminResponseObject, error) {
	if _, ok := ctxutil.GetUserID(ctx); !ok {
		return api.GrantOnlineConfAdmin401Response{}, nil
	}

	if err := h.auth.SetAdmin(ctx, uuid.UUID(req.UserId), true); err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.GrantOnlineConfAdmindefaultJSONResponse{StatusCode: statusCode, Body: body}, nil
	}

	// не ждём 15 с: новый админ получает доступ сразу
	h.admins.Refresh(ctx)

	return api.GrantOnlineConfAdmin204Response{}, nil
}

func (h *Handler) RevokeOnlineConfAdmin(
	ctx context.Context,
	req api.RevokeOnlineConfAdminRequestObject,
) (api.RevokeOnlineConfAdminResponseObject, error) {
	currentID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.RevokeOnlineConfAdmin401Response{}, nil
	}

	targetID := uuid.UUID(req.UserId)

	// иначе можно остаться без единого админа
	if targetID == currentID {
		statusCode, body := keyHTTPError(ctx, fmt.Errorf("%w: cannot revoke your own admin rights", models.ErrValidation))
		return api.RevokeOnlineConfAdmindefaultJSONResponse{StatusCode: statusCode, Body: body}, nil
	}

	if err := h.auth.SetAdmin(ctx, targetID, false); err != nil {
		statusCode, body := keyHTTPError(ctx, err)
		return api.RevokeOnlineConfAdmindefaultJSONResponse{StatusCode: statusCode, Body: body}, nil
	}

	// снятые права перестают действовать сразу
	h.admins.Refresh(ctx)

	return api.RevokeOnlineConfAdmin204Response{}, nil
}

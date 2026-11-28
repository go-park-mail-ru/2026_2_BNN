package delivery

import (
	"context"
	"errors"

	"bnn/internal/models"
	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/utils/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/samber/lo"
)

type Handler struct {
	auth   auth.Usecase
	admins *onlineconf.AdminCache
	keys   onlineconf.CoreUsecase
}

func NewHandler(
	authUC auth.Usecase,
	admins *onlineconf.AdminCache,
	keys onlineconf.CoreUsecase,
) *Handler {
	return &Handler{
		auth:   authUC,
		admins: admins,
		keys:   keys,
	}
}

func (h *Handler) GetOnlineConfSession(ctx context.Context, req api.GetOnlineConfSessionRequestObject) (api.GetOnlineConfSessionResponseObject, error) {
	userID, _ := ctxutil.GetUserID(ctx)

	admin, ok := h.admins.Get(userID)
	if !ok {
		return api.GetOnlineConfSession403JSONResponse{Code: onlineconf.CodeForbidden, Message: messageForbidden}, nil
	}

	return api.GetOnlineConfSession200JSONResponse{Login: admin.Login}, nil
}

func (h *Handler) SearchOnlineConfUsers(ctx context.Context, req api.SearchOnlineConfUsersRequestObject) (api.SearchOnlineConfUsersResponseObject, error) {
	users, err := h.auth.SearchUsers(ctx, req.Params.Query)
	if errors.Is(err, models.ErrValidation) {
		return api.SearchOnlineConfUsers400JSONResponse{Code: onlineconf.CodeValidation, Message: err.Error()}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.SearchOnlineConfUsers200JSONResponse(lo.Map(users, func(u models.User, _ int) api.OnlineConfUser { return toAPIUser(u) })), nil
}

func toAPIUser(u models.User) api.OnlineConfUser {
	return api.OnlineConfUser{Id: openapi_types.UUID(u.ID), Login: u.Login}
}

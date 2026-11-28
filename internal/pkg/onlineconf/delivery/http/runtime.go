package delivery

import (
	"context"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/runtimeapi"
	"bnn/internal/pkg/utils/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type RuntimeHandler struct {
	snapshot *onlineconf.SnapshotCache
}

func NewRuntimeHandler(snapshot *onlineconf.SnapshotCache) *RuntimeHandler {
	return &RuntimeHandler{snapshot: snapshot}
}

func (h *RuntimeHandler) GetOnlineConfRuntime(ctx context.Context, req runtimeapi.GetOnlineConfRuntimeRequestObject) (runtimeapi.GetOnlineConfRuntimeResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return runtimeapi.GetOnlineConfRuntime401Response{}, nil
	}

	snapshot := h.snapshot.Get()
	resp := runtimeapi.OnlineConfRuntime{
		Revision: snapshot.Revision,
		UserId:   openapi_types.UUID(userID),
		Configs:  map[string]map[string]interface{}{},
		Flags:    map[string]map[string]bool{},
	}

	for _, key := range snapshot.Keys {
		// продуктовому фронту — только скоуп frontend и публичные ключи
		if key.Scope != onlineconf.ScopeFrontend && !key.IsPublic {
			continue
		}

		if key.Config != nil {
			put(resp.Configs, key, any(key.Config.Value))
		}
		if key.Flag != nil {
			enabled, _ := onlineconf.EvaluateFlag(key, userID)
			put(resp.Flags, key, enabled)
		}
	}

	return runtimeapi.GetOnlineConfRuntime200JSONResponse(resp), nil
}

// put кладёт значение в values[scope][key], создавая вложенную карту при первом ключе скоупа.
func put[V any](values map[string]map[string]V, key models.OnlineConfKey, value V) {
	if values[key.Scope] == nil {
		values[key.Scope] = map[string]V{}
	}
	values[key.Scope][key.Key] = value
}

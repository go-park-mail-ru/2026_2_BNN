package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/utils/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateOnlineConfKey(t *testing.T) {
	adminID := uuid.NewV4()
	userID := uuid.NewV4()
	flagID := uuid.NewV4()
	configID := uuid.NewV4()

	snapshot := models.Snapshot{Revision: 7, Keys: []models.OnlineConfKey{
		{ID: flagID, Scope: "backend", Key: "attachments.upload_enabled",
			Flag: &models.OnlineConfFlag{Enabled: true, IncludeUsers: []models.OnlineConfUser{{ID: userID}}}},
		{ID: configID, Scope: "backend", Key: "attachments.max_file_size_bytes",
			Config: &models.OnlineConfConfig{ValueType: "number", Value: json.RawMessage("10485760")}},
	}}

	h := NewHandler(nil, nil, nil, newSnapshotCache(t, snapshot))
	ctx := ctxutil.WithUserID(context.Background(), adminID)

	evaluate := func(ctx context.Context, keyID uuid.UUID, body *api.OnlineConfEvaluateInput) api.EvaluateOnlineConfKeyResponseObject {
		resp, err := h.EvaluateOnlineConfKey(ctx, api.EvaluateOnlineConfKeyRequestObject{
			Id: openapi_types.UUID(keyID), Body: body,
		})
		assert.NoError(t, err)
		return resp
	}
	forUser := &api.OnlineConfEvaluateInput{UserId: openapi_types.UUID(userID)}

	t.Run("included", func(t *testing.T) {
		assert.Equal(t, api.EvaluateOnlineConfKey200JSONResponse{
			Result: true, Reason: api.Included, Revision: 7,
		}, evaluate(ctx, flagID, forUser))
	})

	t.Run("config key", func(t *testing.T) {
		resp := evaluate(ctx, configID, forUser).(api.EvaluateOnlineConfKeydefaultJSONResponse)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, onlineconf.CodeValidation, resp.Body.Code)
	})

	t.Run("unknown key", func(t *testing.T) {
		resp := evaluate(ctx, uuid.NewV4(), forUser).(api.EvaluateOnlineConfKeydefaultJSONResponse)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("no body", func(t *testing.T) {
		resp := evaluate(ctx, flagID, nil).(api.EvaluateOnlineConfKeydefaultJSONResponse)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("no session", func(t *testing.T) {
		assert.Equal(t, api.EvaluateOnlineConfKey401Response{}, evaluate(context.Background(), flagID, forUser))
	})
}

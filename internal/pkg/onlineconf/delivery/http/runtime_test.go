package delivery

import (
	"context"
	"encoding/json"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/runtimeapi"
	"bnn/internal/pkg/onlineconf/mocks"
	"bnn/internal/pkg/utils/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newSnapshotCache отдаёт кеш, в котором уже лежит snapshot.
func newSnapshotCache(t *testing.T, snapshot models.Snapshot) *onlineconf.SnapshotCache {
	source := mocks.NewMockSnapshotSource(gomock.NewController(t))
	cache := onlineconf.NewSnapshotCache(source)

	ctx, cancel := context.WithCancel(context.Background())
	source.EXPECT().GetSnapshot(gomock.Any(), gomock.Nil()).DoAndReturn(func(context.Context, *int64) (models.Snapshot, bool, error) {
		cancel()
		return snapshot, true, nil
	})
	cache.Run(ctx)

	return cache
}

func TestGetOnlineConfRuntime(t *testing.T) {
	userID := uuid.NewV4()

	snapshot := models.Snapshot{Revision: 42, Keys: []models.OnlineConfKey{
		{Scope: "backend", Key: "attachments.max_file_size_bytes", IsPublic: true,
			Config: &models.OnlineConfConfig{ValueType: "number", Value: json.RawMessage("10485760")}},
		{Scope: "backend", Key: "attachments.upload_enabled", IsPublic: true,
			Flag: &models.OnlineConfFlag{Enabled: true, RolloutPercent: 100}},
		{Scope: "frontend", Key: "new_editor",
			Flag: &models.OnlineConfFlag{Enabled: true, ExcludeUsers: []models.OnlineConfUser{{ID: userID}}}},
		{Scope: "notes", Key: "secret",
			Config: &models.OnlineConfConfig{ValueType: "string", Value: json.RawMessage(`"hidden"`)}},
	}}

	h := NewRuntimeHandler(newSnapshotCache(t, snapshot))

	resp, err := h.GetOnlineConfRuntime(ctxutil.WithUserID(context.Background(), userID), runtimeapi.GetOnlineConfRuntimeRequestObject{})
	assert.NoError(t, err)
	assert.Equal(t, runtimeapi.GetOnlineConfRuntime200JSONResponse{
		Revision: 42,
		UserId:   openapi_types.UUID(userID),
		Configs:  map[string]map[string]interface{}{"backend": {"attachments.max_file_size_bytes": json.RawMessage("10485760")}},
		Flags: map[string]map[string]bool{
			"backend":  {"attachments.upload_enabled": true},
			"frontend": {"new_editor": false},
		},
	}, resp)
}

func TestGetOnlineConfRuntimeNoUser(t *testing.T) {
	h := NewRuntimeHandler(onlineconf.NewSnapshotCache(nil))

	resp, err := h.GetOnlineConfRuntime(context.Background(), runtimeapi.GetOnlineConfRuntimeRequestObject{})
	assert.NoError(t, err)
	assert.Equal(t, runtimeapi.GetOnlineConfRuntime401Response{}, resp)
}

func TestGetOnlineConfRuntimeBeforeFirstSnapshot(t *testing.T) {
	userID := uuid.NewV4()
	h := NewRuntimeHandler(onlineconf.NewSnapshotCache(nil))

	resp, err := h.GetOnlineConfRuntime(ctxutil.WithUserID(context.Background(), userID), runtimeapi.GetOnlineConfRuntimeRequestObject{})
	assert.NoError(t, err)
	assert.Equal(t, runtimeapi.GetOnlineConfRuntime200JSONResponse{
		UserId:  openapi_types.UUID(userID),
		Configs: map[string]map[string]interface{}{},
		Flags:   map[string]map[string]bool{},
	}, resp)
}

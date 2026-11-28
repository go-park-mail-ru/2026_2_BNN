package delivery

import (
	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"context"
	"encoding/json"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"net/http"
	"testing"
	"time"
)

func TestP2AdminAccess(t *testing.T) {
	id := uuid.NewV4().String()
	for _, route := range []struct{ method, path, body string }{
		{http.MethodGet, "rollout-plans", ""},
		{http.MethodPost, "rollout-plans", `{"name":"Basic","steps":[5,100],"interval_min":30}`},
		{http.MethodGet, "rollout-plans/" + id, ""},
		{http.MethodPut, "rollout-plans/" + id, `{"name":"Basic","steps":[5,100],"interval_min":30}`},
		{http.MethodDelete, "rollout-plans/" + id, ""},
		{http.MethodPut, "keys/" + id + "/rollout-plan", `{"plan_id":null,"version":1}`},
		{http.MethodGet, "keys/" + id + "/rollout", ""},
		{http.MethodPost, "keys/" + id + "/rollout/approve", `{"version":1}`},
		{http.MethodPost, "keys/" + id + "/rollout/rollback", `{"version":1}`},
		{http.MethodGet, "keys/" + id + "/schedules", ""},
		{http.MethodPost, "keys/" + id + "/schedules", `{"run_at":"2099-01-01T00:00:00Z","enabled":false}`},
		{http.MethodDelete, "keys/" + id + "/schedules/" + id, ""},
	} {
		for _, token := range []string{"", "user"} {
			t.Run(route.method+"/"+route.path+"/"+token, func(t *testing.T) {
				router, _, _ := newKeysRouter(t)
				response := keyRequest(router, route.method, "/api/onlineconf/admin/"+route.path, route.body, token, true)
				expected := http.StatusForbidden
				if token == "" {
					expected = http.StatusUnauthorized
				}
				assert.Equal(t, expected, response.Code, response.Body.String())
				assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			})
		}
	}
}
func TestP2WritesRequireCSRF(t *testing.T) {
	id := uuid.NewV4().String()
	for _, path := range []string{"keys/" + id + "/rollout/approve", "keys/" + id + "/rollout/rollback", "keys/" + id + "/schedules", "rollout-plans"} {
		t.Run(path, func(t *testing.T) {
			router, _, _ := newKeysRouter(t)
			response := keyRequest(router, http.MethodPost, "/api/onlineconf/admin/"+path, "{}", "admin", false)
			assert.Equal(t, http.StatusForbidden, response.Code)
		})
	}
}
func TestCreateOnlineConfRolloutPlan(t *testing.T) {
	router, keys, _ := newKeysRouter(t)
	input := onlineconf.RolloutPlanInput{Name: "Basic", Steps: []int32{5, 100}, IntervalMin: 30}
	plan := models.OnlineConfRolloutPlan{ID: uuid.NewV4(), Name: input.Name, Steps: input.Steps, IntervalMin: input.IntervalMin}
	keys.EXPECT().CreateRolloutPlan(gomock.Any(), input).Return(plan, nil)
	response := keyRequest(router, http.MethodPost, "/api/onlineconf/admin/rollout-plans", `{"name":"Basic","steps":[5,100],"interval_min":30}`, "admin", true)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var got models.OnlineConfRolloutPlan
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &got))
	assert.Equal(t, plan, got)
}
func TestSetRolloutPlanRequiresExplicitPlanID(t *testing.T) {
	router, _, _ := newKeysRouter(t)
	response := keyRequest(router, http.MethodPut, "/api/onlineconf/admin/keys/"+uuid.NewV4().String()+"/rollout-plan", `{"version":1}`, "admin", true)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}
func TestSetRolloutPlanNullUnassigns(t *testing.T) {
	router, keys, actor := newKeysRouter(t)
	keyID := uuid.NewV4()
	key := models.OnlineConfKey{ID: keyID, Scope: onlineconf.ScopeBackend, Key: onlineconf.KeyAttachmentsUploadEnabled, Kind: onlineconf.KindFlag, Flag: &models.OnlineConfFlag{Enabled: true, RolloutPercent: 25}, Version: 2, UpdatedBy: models.OnlineConfUser{ID: actor}, UpdatedAt: time.Now().UTC()}
	keys.EXPECT().SetRolloutPlan(gomock.Any(), actor, keyID, gomock.Nil(), int32(1)).Return(key, nil)
	response := keyRequest(router, http.MethodPut, "/api/onlineconf/admin/keys/"+keyID.String()+"/rollout-plan", `{"plan_id":null,"version":1}`, "admin", true)
	assert.Equal(t, http.StatusOK, response.Code, response.Body.String())
}
func TestApproveOnlineConfRolloutNotReady(t *testing.T) {
	router, keys, actor := newKeysRouter(t)
	keyID := uuid.NewV4()
	keys.EXPECT().AdvanceRollout(gomock.Any(), actor, keyID, int32(1), false).Return(models.OnlineConfKey{}, models.ErrOnlineConfRolloutNotReady)
	response := keyRequest(router, http.MethodPost, "/api/onlineconf/admin/keys/"+keyID.String()+"/rollout/approve", `{"version":1}`, "admin", true)
	require.Equal(t, http.StatusConflict, response.Code)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, onlineconf.CodeRolloutNotReady, body.Code)
}
func TestCreateOnlineConfSchedulePreservesFalse(t *testing.T) {
	router, keys, actor := newKeysRouter(t)
	keyID := uuid.NewV4()
	runAt := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	keys.EXPECT().CreateSchedule(gomock.Any(), actor, keyID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, _ uuid.UUID, input onlineconf.ScheduleInput) (models.OnlineConfSchedule, error) {
		require.NotNil(t, input.Enabled)
		assert.False(t, *input.Enabled)
		assert.Nil(t, input.RolloutPercent)
		return models.OnlineConfSchedule{ID: uuid.NewV4(), KeyID: keyID, RunAt: runAt, Enabled: input.Enabled, CreatedBy: actor, CreatedAt: time.Now().UTC()}, nil
	})
	response := keyRequest(router, http.MethodPost, "/api/onlineconf/admin/keys/"+keyID.String()+"/schedules", `{"run_at":"2099-01-01T00:00:00Z","enabled":false}`, "admin", true)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var s models.OnlineConfSchedule
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &s))
	require.NotNil(t, s.Enabled)
	assert.False(t, *s.Enabled)
}

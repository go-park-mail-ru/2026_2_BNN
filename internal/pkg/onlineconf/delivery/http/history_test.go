package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListOnlineConfHistory(t *testing.T) {
	router, keys, adminID := newKeysRouter(t)

	now := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	actor := models.OnlineConfUser{
		ID: adminID, Login: "liza",
	}

	key := models.OnlineConfKey{
		ID:        uuid.NewV4(),
		Scope:     onlineconf.ScopeBackend,
		Key:       "test.settings",
		Kind:      onlineconf.KindConfig,
		Version:   1,
		UpdatedBy: actor,
		UpdatedAt: now,
		Config: &models.OnlineConfConfig{
			ValueType: onlineconf.ValueObject,
			Value:     json.RawMessage(`{"limits":[1,2]}`),
		},
	}

	keys.EXPECT().
		ListHistory(gomock.Any(), key.ID).
		Return([]models.OnlineConfHistory{
			{
				ID:        uuid.NewV4(),
				KeyID:     key.ID,
				Action:    "delete",
				OldValue:  &key,
				User:      actor,
				CreatedAt: now.Add(time.Minute),
			},
			{
				ID:        uuid.NewV4(),
				KeyID:     key.ID,
				Action:    "create",
				NewValue:  &key,
				User:      actor,
				CreatedAt: now,
			},
		}, nil)

	response := keyRequest(
		router,
		http.MethodGet,
		"/api/onlineconf/admin/keys/"+key.ID.String()+"/history",
		"",
		"admin",
		false,
	)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))

	var body []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Len(t, body, 2)

	assert.JSONEq(t, `"delete"`, string(body[0]["action"]))
	assert.JSONEq(t, `"create"`, string(body[1]["action"]))

	require.Contains(t, body[0], "new_value")
	assert.Equal(t, "null", string(body[0]["new_value"]))

	require.Contains(t, body[1], "old_value")
	assert.Equal(t, "null", string(body[1]["old_value"]))

	var oldKey models.OnlineConfKey
	require.NoError(t, json.Unmarshal(body[0]["old_value"], &oldKey))
	assert.Equal(t, key, oldKey)

	var newKey models.OnlineConfKey
	require.NoError(t, json.Unmarshal(body[1]["new_value"], &newKey))
	assert.Equal(t, key, newKey)

	var user models.OnlineConfUser
	require.NoError(t, json.Unmarshal(body[0]["user"], &user))
	assert.Equal(t, actor, user)
}

func TestListOnlineConfHistoryEmpty(t *testing.T) {
	router, keys, _ := newKeysRouter(t)
	keyID := uuid.NewV4()

	keys.EXPECT().
		ListHistory(gomock.Any(), keyID).
		Return(nil, nil)

	response := keyRequest(
		router,
		http.MethodGet,
		"/api/onlineconf/admin/keys/"+keyID.String()+"/history",
		"",
		"admin",
		false,
	)

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[]`, response.Body.String())
}

func TestListOnlineConfHistoryAccess(t *testing.T) {
	for _, token := range []string{"", "user"} {
		t.Run(token, func(t *testing.T) {
			router, _, _ := newKeysRouter(t)

			response := keyRequest(
				router,
				http.MethodGet,
				"/api/onlineconf/admin/keys/"+uuid.NewV4().String()+"/history",
				"",
				token,
				false,
			)

			expected := http.StatusForbidden
			if token == "" {
				expected = http.StatusUnauthorized
			}

			assert.Equal(t, expected, response.Code)
		})
	}
}

func TestListOnlineConfHistoryUnavailable(t *testing.T) {
	router, keys, _ := newKeysRouter(t)
	keyID := uuid.NewV4()

	keys.EXPECT().
		ListHistory(gomock.Any(), keyID).
		Return(nil, context.DeadlineExceeded)

	response := keyRequest(
		router,
		http.MethodGet,
		"/api/onlineconf/admin/keys/"+keyID.String()+"/history",
		"",
		"admin",
		false,
	)

	assert.Equal(t, http.StatusServiceUnavailable, response.Code)

	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, codeUnavailable, body.Code)
}

package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bnn/internal/models"
	authDelivery "bnn/internal/pkg/auth/delivery/http"
	authMocks "bnn/internal/pkg/auth/mocks"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/onlineconf/mocks"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newKeysRouter(t *testing.T) (
	http.Handler,
	*mocks.MockUsecase,
	uuid.UUID,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	keys := mocks.NewMockUsecase(ctrl)
	auth := authMocks.NewMockUsecase(ctrl)
	adminSource := mocks.NewMockAdminSource(ctrl)

	adminID := uuid.NewV4()
	regularID := uuid.NewV4()

	auth.EXPECT().Authenticate(gomock.Any(), "admin").
		Return(adminID, nil).AnyTimes()
	auth.EXPECT().Authenticate(gomock.Any(), "user").
		Return(regularID, nil).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adminSource.EXPECT().ListAdmins(gomock.Any()).
		DoAndReturn(func(context.Context) ([]models.User, error) {
			cancel()
			return []models.User{{ID: adminID, Login: "liza"}}, nil
		})

	admins := onlineconf.NewAdminCache(adminSource)
	admins.Run(ctx)

	handler := NewHandler(auth, admins, keys, nil)
	authHandler := authDelivery.NewHandler(auth, "")

	router := mux.NewRouter()
	router.Use(NoStore)

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.Use(authHandler.Middleware)

	adminRouter := apiRouter.NewRoute().Subrouter()
	adminRouter.Use(AdminOnly(admins))

	strict := api.NewStrictHandlerWithOptions(
		handler,
		nil,
		api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  RequestError,
			ResponseErrorHandlerFunc: ResponseError,
		},
	)

	api.HandlerWithOptions(strict, api.GorillaServerOptions{
		BaseRouter:       adminRouter,
		ErrorHandlerFunc: RequestError,
	})

	return router, keys, adminID
}

func keyRequest(
	router http.Handler,
	method, path, body, token string,
	csrf bool,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	if token != "" {
		request.AddCookie(&http.Cookie{Name: "bnn_access", Value: token})
	}
	if csrf {
		request.AddCookie(&http.Cookie{Name: "bnn_csrf", Value: "csrf"})
		request.Header.Set("X-CSRF-Token", "csrf")
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestKeysAccess(t *testing.T) {
	for _, token := range []string{"", "user"} {
		t.Run(token, func(t *testing.T) {
			router, _, _ := newKeysRouter(t)

			response := keyRequest(
				router, http.MethodGet,
				"/api/onlineconf/admin/keys", "", token, false,
			)

			expected := http.StatusForbidden
			if token == "" {
				expected = http.StatusUnauthorized
			}

			assert.Equal(t, expected, response.Code)
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		})
	}
}

func TestListScopesHTTP(t *testing.T) {
	router, keys, _ := newKeysRouter(t)
	keys.EXPECT().ListScopes().
		Return([]string{"frontend", "backend", "auth", "notes"})

	response := keyRequest(
		router, http.MethodGet,
		"/api/onlineconf/admin/scopes", "", "admin", false,
	)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t,
		`["frontend","backend","auth","notes"]`,
		response.Body.String(),
	)
}

func TestListKeysHidesOverrides(t *testing.T) {
	router, keys, adminID := newKeysRouter(t)
	key := models.OnlineConfKey{
		ID: uuid.NewV4(), Scope: "backend",
		Key: "attachments.upload_enabled", Kind: "flag", Version: 1,
		UpdatedBy: models.OnlineConfUser{ID: adminID, Login: "liza"},
		UpdatedAt: time.Now().UTC(),
		Flag: &models.OnlineConfFlag{
			Enabled: true, RolloutPercent: 25,
			IncludeUsers: []models.OnlineConfUser{
				{ID: uuid.NewV4(), Login: "included"},
			},
			ExcludeUsers: []models.OnlineConfUser{},
		},
	}

	keys.EXPECT().ListKeys(gomock.Any(), "upload", "backend", "flag").
		Return([]models.OnlineConfKey{key}, nil)

	response := keyRequest(
		router, http.MethodGet,
		"/api/onlineconf/admin/keys?query=upload&scope=backend&kind=flag",
		"", "admin", false,
	)

	require.Equal(t, http.StatusOK, response.Code)

	var result []struct {
		Flag map[string]json.RawMessage `json:"flag"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.Len(t, result, 1)
	assert.NotContains(t, result[0].Flag, "include_users")
	assert.NotContains(t, result[0].Flag, "exclude_users")
}

func TestCreateKeyPreservesNumberAndActor(t *testing.T) {
	router, keys, adminID := newKeysRouter(t)

	input := onlineconf.KeyInput{
		Scope: "backend", Key: "test.value", Kind: "config",
		IsPublic: true,
		Config: &models.OnlineConfConfig{
			ValueType: "number",
			Value:     json.RawMessage("9007199254740993"),
		},
	}

	keys.EXPECT().CreateKey(gomock.Any(), adminID, input).
		DoAndReturn(func(
			_ context.Context,
			actor uuid.UUID,
			input onlineconf.KeyInput,
		) (models.OnlineConfKey, error) {
			return models.OnlineConfKey{
				ID: uuid.NewV4(), Scope: input.Scope,
				Key: input.Key, Kind: input.Kind, IsPublic: input.IsPublic,
				Config: input.Config, Version: 1,
				UpdatedBy: models.OnlineConfUser{ID: actor, Login: "liza"},
				UpdatedAt: time.Now().UTC(),
			}, nil
		})

	response := keyRequest(
		router, http.MethodPost, "/api/onlineconf/admin/keys",
		`{
			"scope":"backend",
			"key":"test.value",
			"kind":"config",
			"is_public":true,
			"updated_by":{"id":"00000000-0000-0000-0000-000000000000"},
			"config":{"value_type":"number","value":9007199254740993}
		}`,
		"admin", true,
	)

	require.Equal(t, http.StatusCreated, response.Code)
	assert.Contains(t, response.Body.String(), `"value":9007199254740993`)
}

func TestUpdateVersionConflictHTTP(t *testing.T) {
	router, keys, adminID := newKeysRouter(t)
	id := uuid.NewV4()

	keys.EXPECT().UpdateKey(
		gomock.Any(), adminID, id, int32(3), gomock.Any(),
	).Return(models.OnlineConfKey{}, models.ErrOnlineConfVersionConflict)

	response := keyRequest(
		router, http.MethodPut, "/api/onlineconf/admin/keys/"+id.String(),
		`{
			"scope":"backend","key":"test.value","kind":"config",
			"is_public":true,"version":3,
			"config":{"value_type":"number","value":42}
		}`,
		"admin", true,
	)

	require.Equal(t, http.StatusConflict, response.Code)
	assert.Contains(t, response.Body.String(), `"code":"VERSION_CONFLICT"`)
}

func TestGetKeyErrorsHTTP(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"missing", models.ErrOnlineConfKeyNotFound, 404, "NOT_FOUND"},
		{"unavailable", status.Error(codes.Unavailable, "offline"), 503, "SERVICE_UNAVAILABLE"},
		{"internal", errors.New("private database details"), 500, "INTERNAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, keys, _ := newKeysRouter(t)
			id := uuid.NewV4()

			keys.EXPECT().GetKey(gomock.Any(), id).
				Return(models.OnlineConfKey{}, tt.err)

			response := keyRequest(
				router, http.MethodGet,
				"/api/onlineconf/admin/keys/"+id.String(),
				"", "admin", false,
			)

			assert.Equal(t, tt.status, response.Code)
			assert.Contains(t, response.Body.String(), `"code":"`+tt.code+`"`)
			assert.NotContains(t, response.Body.String(), "private database details")
		})
	}
}

func TestInvalidRequestsHTTP(t *testing.T) {
	id := uuid.NewV4().String()

	tests := []struct {
		name, method, path, body string
	}{
		{"bad json", "POST", "/api/onlineconf/admin/keys", "{"},
		{"bad id", "GET", "/api/onlineconf/admin/keys/broken", ""},
		{"missing delete version", "DELETE", "/api/onlineconf/admin/keys/" + id, ""},
		{"bad delete version", "DELETE", "/api/onlineconf/admin/keys/" + id + "?version=bad", ""},
		{
			"missing update version", "PUT",
			"/api/onlineconf/admin/keys/" + id,
			`{
				"scope":"backend","key":"test.value","kind":"config",
				"is_public":true,
				"config":{"value_type":"number","value":42}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _, _ := newKeysRouter(t)

			response := keyRequest(
				router, tt.method, tt.path, tt.body, "admin", true,
			)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Contains(t, response.Body.String(), `"code":"VALIDATION"`)
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		})
	}
}

func TestCreateKeyRequiresCSRF(t *testing.T) {
	router, _, _ := newKeysRouter(t)

	response := keyRequest(
		router, http.MethodPost, "/api/onlineconf/admin/keys",
		`{}`, "admin", false,
	)

	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
}

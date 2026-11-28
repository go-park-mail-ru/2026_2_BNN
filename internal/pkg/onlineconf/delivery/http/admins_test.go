package delivery

import (
	"context"
	"net/http"
	"testing"

	"bnn/internal/models"
	authMocks "bnn/internal/pkg/auth/mocks"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/onlineconf/mocks"
	"bnn/internal/pkg/utils/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type adminsFixture struct {
	handler *Handler
	auth    *authMocks.MockUsecase
	source  *mocks.MockAdminSource
	admins  *onlineconf.AdminCache
	me      models.User
	ctx     context.Context
}

// newAdminsFixture — хендлер, у которого в кеше уже лежит один админ (me).
func newAdminsFixture(t *testing.T) adminsFixture {
	ctrl := gomock.NewController(t)
	auth := authMocks.NewMockUsecase(ctrl)
	source := mocks.NewMockAdminSource(ctrl)
	me := models.User{ID: uuid.NewV4(), Login: "seyran"}

	admins := onlineconf.NewAdminCache(source)
	source.EXPECT().ListAdmins(gomock.Any()).Return([]models.User{me}, nil)
	admins.Refresh(context.Background())

	return adminsFixture{
		handler: NewHandler(auth, admins, nil, nil),
		auth:    auth,
		source:  source,
		admins:  admins,
		me:      me,
		ctx:     ctxutil.WithUserID(context.Background(), me.ID),
	}
}

func TestListOnlineConfAdmins(t *testing.T) {
	f := newAdminsFixture(t)
	liza := models.User{ID: uuid.NewV4(), Login: "liza"}
	f.source.EXPECT().ListAdmins(gomock.Any()).Return([]models.User{f.me, liza}, nil)
	f.admins.Refresh(context.Background())

	resp, err := f.handler.ListOnlineConfAdmins(f.ctx, api.ListOnlineConfAdminsRequestObject{})
	assert.NoError(t, err)
	// по алфавиту логинов
	assert.Equal(t, api.ListOnlineConfAdmins200JSONResponse{
		{Id: openapi_types.UUID(liza.ID), Login: "liza"},
		{Id: openapi_types.UUID(f.me.ID), Login: "seyran"},
	}, resp)
}

func TestGrantOnlineConfAdmin(t *testing.T) {
	f := newAdminsFixture(t)
	liza := models.User{ID: uuid.NewV4(), Login: "liza"}

	f.auth.EXPECT().SetAdmin(gomock.Any(), liza.ID, true).Return(nil)
	// кеш перечитывается сразу, без ожидания 15 с
	f.source.EXPECT().ListAdmins(gomock.Any()).Return([]models.User{f.me, liza}, nil)

	resp, err := f.handler.GrantOnlineConfAdmin(f.ctx, api.GrantOnlineConfAdminRequestObject{UserId: openapi_types.UUID(liza.ID)})
	assert.NoError(t, err)
	assert.Equal(t, api.GrantOnlineConfAdmin204Response{}, resp)

	_, ok := f.admins.Get(liza.ID)
	assert.True(t, ok)
}

func TestGrantOnlineConfAdminUnknownUser(t *testing.T) {
	f := newAdminsFixture(t)
	unknown := uuid.NewV4()
	f.auth.EXPECT().SetAdmin(gomock.Any(), unknown, true).Return(models.ErrUserNotFound)

	resp, err := f.handler.GrantOnlineConfAdmin(f.ctx, api.GrantOnlineConfAdminRequestObject{UserId: openapi_types.UUID(unknown)})
	assert.NoError(t, err)

	failure := resp.(api.GrantOnlineConfAdmindefaultJSONResponse)
	assert.Equal(t, http.StatusNotFound, failure.StatusCode)
	assert.Equal(t, "NOT_FOUND", failure.Body.Code)
}

func TestRevokeOnlineConfAdmin(t *testing.T) {
	f := newAdminsFixture(t)
	liza := uuid.NewV4()

	f.auth.EXPECT().SetAdmin(gomock.Any(), liza, false).Return(nil)
	f.source.EXPECT().ListAdmins(gomock.Any()).Return([]models.User{f.me}, nil)

	resp, err := f.handler.RevokeOnlineConfAdmin(f.ctx, api.RevokeOnlineConfAdminRequestObject{UserId: openapi_types.UUID(liza)})
	assert.NoError(t, err)
	assert.Equal(t, api.RevokeOnlineConfAdmin204Response{}, resp)
}

func TestRevokeOnlineConfAdminSelf(t *testing.T) {
	f := newAdminsFixture(t)

	// SetAdmin не вызывается: gomock упадёт, если вызовут
	resp, err := f.handler.RevokeOnlineConfAdmin(f.ctx, api.RevokeOnlineConfAdminRequestObject{UserId: openapi_types.UUID(f.me.ID)})
	assert.NoError(t, err)

	failure := resp.(api.RevokeOnlineConfAdmindefaultJSONResponse)
	assert.Equal(t, http.StatusBadRequest, failure.StatusCode)
	assert.Equal(t, onlineconf.CodeValidation, failure.Body.Code)
}

func TestOnlineConfAdminsNoSession(t *testing.T) {
	f := newAdminsFixture(t)
	ctx := context.Background()

	list, _ := f.handler.ListOnlineConfAdmins(ctx, api.ListOnlineConfAdminsRequestObject{})
	assert.Equal(t, api.ListOnlineConfAdmins401Response{}, list)

	grant, _ := f.handler.GrantOnlineConfAdmin(ctx, api.GrantOnlineConfAdminRequestObject{})
	assert.Equal(t, api.GrantOnlineConfAdmin401Response{}, grant)

	revoke, _ := f.handler.RevokeOnlineConfAdmin(ctx, api.RevokeOnlineConfAdminRequestObject{})
	assert.Equal(t, api.RevokeOnlineConfAdmin401Response{}, revoke)
}

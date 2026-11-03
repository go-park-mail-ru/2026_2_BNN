package delivery

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/auth/delivery/http/api"
	"bnn/internal/pkg/auth/mocks"
	"bnn/internal/pkg/cookies"
	"bnn/internal/pkg/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var tokens = models.AuthTokens{
	AccessToken:      "access",
	AccessExpiresAt:  time.Now().Add(time.Hour),
	RefreshToken:     "refresh",
	RefreshExpiresAt: time.Now().Add(24 * time.Hour),
}

func withCookies(reqCookies []*http.Cookie, call func(ctx context.Context)) map[string]*http.Cookie {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	for _, c := range reqCookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()

	cookies.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	result := make(map[string]*http.Cookie)
	for _, c := range rec.Result().Cookies() {
		result[c.Name] = c
	}
	return result
}

func newForm(t *testing.T, field string, data []byte) *multipart.Reader {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	part, err := w.CreateFormFile(field, "avatar")
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return multipart.NewReader(&buf, w.Boundary())
}

func TestSignUp(t *testing.T) {
	user := models.User{ID: uuid.NewV4(), Login: "seyran"}
	body := &api.SignUpJSONRequestBody{Login: "seyran", Password: "password123"}

	tests := []struct {
		name        string
		ucErr       error
		expected    api.SignUpResponseObject
		withCookies bool
	}{
		{name: "ok", expected: api.SignUp201JSONResponse(toAPIUser(user)), withCookies: true},
		{name: "validation", ucErr: models.ErrValidation, expected: api.SignUp400Response{}},
		{name: "user exists", ucErr: models.ErrUserExists, expected: api.SignUp409Response{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			uc.EXPECT().SignUp(gomock.Any(), "seyran", "password123").Return(user, tokens, tt.ucErr)

			var resp api.SignUpResponseObject
			var err error
			set := withCookies(nil, func(ctx context.Context) {
				resp, err = NewHandler(uc).SignUp(ctx, api.SignUpRequestObject{Body: body})
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
			if tt.withCookies {
				assert.Equal(t, "access", set[accessCookieName].Value)
				assert.Equal(t, "refresh", set[refreshCookieName].Value)
			} else {
				assert.Empty(t, set)
			}
		})
	}
}

func TestSignIn(t *testing.T) {
	user := models.User{ID: uuid.NewV4(), Login: "seyran"}
	body := &api.SignInJSONRequestBody{Login: "seyran", Password: "password123"}

	tests := []struct {
		name     string
		ucErr    error
		expected api.SignInResponseObject
	}{
		{name: "ok", expected: api.SignIn200JSONResponse(toAPIUser(user))},
		{name: "invalid credentials", ucErr: models.ErrInvalidCredentials, expected: api.SignIn401Response{}},
		{name: "oauth account", ucErr: models.ErrOAuthAccount, expected: api.SignIn403Response{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			uc.EXPECT().SignIn(gomock.Any(), "seyran", "password123").Return(user, tokens, tt.ucErr)

			var resp api.SignInResponseObject
			var err error
			set := withCookies(nil, func(ctx context.Context) {
				resp, err = NewHandler(uc).SignIn(ctx, api.SignInRequestObject{Body: body})
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
			if tt.ucErr == nil {
				assert.Equal(t, "access", set[accessCookieName].Value)
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	refreshCookie := &http.Cookie{Name: refreshCookieName, Value: "old-refresh"}

	tests := []struct {
		name     string
		cookies  []*http.Cookie
		prepare  func(uc *mocks.MockUsecase)
		expected api.RefreshResponseObject
	}{
		{
			name:    "ok",
			cookies: []*http.Cookie{refreshCookie},
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Refresh(gomock.Any(), "old-refresh").Return(tokens, nil)
			},
			expected: api.Refresh200Response{},
		},
		{
			name:     "no cookie",
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.Refresh401Response{},
		},
		{
			name:    "invalid token",
			cookies: []*http.Cookie{refreshCookie},
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Refresh(gomock.Any(), "old-refresh").Return(models.AuthTokens{}, models.ErrInvalidToken)
			},
			expected: api.Refresh401Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			var resp api.RefreshResponseObject
			var err error
			set := withCookies(tt.cookies, func(ctx context.Context) {
				resp, err = NewHandler(uc).Refresh(ctx, api.RefreshRequestObject{})
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
			if _, ok := tt.expected.(api.Refresh200Response); ok {
				assert.Equal(t, "refresh", set[refreshCookieName].Value)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name    string
		cookies []*http.Cookie
		prepare func(uc *mocks.MockUsecase)
	}{
		{
			name:    "with refresh cookie",
			cookies: []*http.Cookie{{Name: refreshCookieName, Value: "refresh"}},
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Logout(gomock.Any(), "refresh").Return(nil)
			},
		},
		{
			name:    "without cookie",
			prepare: func(uc *mocks.MockUsecase) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			var resp api.LogoutResponseObject
			var err error
			set := withCookies(tt.cookies, func(ctx context.Context) {
				resp, err = NewHandler(uc).Logout(ctx, api.LogoutRequestObject{})
			})

			require.NoError(t, err)
			assert.Equal(t, api.Logout200Response{}, resp)
			assert.Negative(t, set[accessCookieName].MaxAge)
			assert.Negative(t, set[refreshCookieName].MaxAge)
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	userID := uuid.NewV4()
	user := models.User{ID: userID, Login: "seyran", Avatar: "/avatars/a.png"}

	tests := []struct {
		name     string
		ctx      context.Context
		prepare  func(uc *mocks.MockUsecase)
		expected api.GetCurrentUserResponseObject
	}{
		{
			name: "ok",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetUser(gomock.Any(), userID).Return(user, nil)
			},
			expected: api.GetCurrentUser200JSONResponse{
				Id:     openapi_types.UUID(userID),
				Login:  "seyran",
				Avatar: "/avatars/a.png",
			},
		},
		{
			name: "user not found",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetUser(gomock.Any(), userID).Return(models.User{}, models.ErrUserNotFound)
			},
			expected: api.GetCurrentUser404Response{},
		},
		{
			name:     "no user",
			ctx:      context.Background(),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.GetCurrentUser401Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			resp, err := NewHandler(uc).GetCurrentUser(tt.ctx, api.GetCurrentUserRequestObject{})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestChangePassword(t *testing.T) {
	userID := uuid.NewV4()
	body := &api.ChangePasswordJSONRequestBody{OldPassword: "old", NewPassword: "newpassword"}

	tests := []struct {
		name     string
		ucErr    error
		expected api.ChangePasswordResponseObject
	}{
		{name: "ok", expected: api.ChangePassword204Response{}},
		{name: "validation", ucErr: models.ErrValidation, expected: api.ChangePassword400Response{}},
		{name: "wrong password", ucErr: models.ErrWrongPassword, expected: api.ChangePassword403Response{}},
		{name: "user deleted", ucErr: models.ErrUserNotFound, expected: api.ChangePassword401Response{}},
		{name: "oauth account", ucErr: models.ErrOAuthAccount, expected: api.ChangePassword400Response{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			uc.EXPECT().ChangePassword(gomock.Any(), userID, "old", "newpassword").Return(tokens, tt.ucErr)

			var resp api.ChangePasswordResponseObject
			var err error
			set := withCookies(nil, func(ctx context.Context) {
				ctx = ctxutil.WithUserID(ctx, userID)
				resp, err = NewHandler(uc).ChangePassword(ctx, api.ChangePasswordRequestObject{Body: body})
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
			if tt.ucErr == nil {
				assert.Equal(t, "access", set[accessCookieName].Value)
				assert.Equal(t, "refresh", set[refreshCookieName].Value)
			}
		})
	}
}

func TestChangePasswordNoUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := mocks.NewMockUsecase(ctrl)

	resp, err := NewHandler(uc).ChangePassword(context.Background(), api.ChangePasswordRequestObject{})

	require.NoError(t, err)
	assert.Equal(t, api.ChangePassword401Response{}, resp)
}

func TestUpdateAvatar(t *testing.T) {
	userID := uuid.NewV4()
	png := []byte("\x89PNG\r\n\x1a\navatar")
	user := models.User{ID: userID, Login: "seyran", Avatar: "/avatars/new.png"}

	tests := []struct {
		name     string
		ctx      context.Context
		form     func(t *testing.T) *multipart.Reader
		prepare  func(uc *mocks.MockUsecase)
		expected api.UpdateAvatarResponseObject
	}{
		{
			name: "ok",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			form: func(t *testing.T) *multipart.Reader { return newForm(t, avatarField, png) },
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateAvatar(gomock.Any(), userID, png, "image/png").Return(user, nil)
			},
			expected: api.UpdateAvatar200JSONResponse(toAPIUser(user)),
		},
		{
			name:     "no user",
			ctx:      context.Background(),
			form:     func(t *testing.T) *multipart.Reader { return newForm(t, avatarField, png) },
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.UpdateAvatar401Response{},
		},
		{
			name:     "file too large",
			ctx:      ctxutil.WithUserID(context.Background(), userID),
			form:     func(t *testing.T) *multipart.Reader { return newForm(t, avatarField, make([]byte, maxAvatarSize5MB+1)) },
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.UpdateAvatar413Response{},
		},
		{
			name:     "no avatar field",
			ctx:      ctxutil.WithUserID(context.Background(), userID),
			form:     func(t *testing.T) *multipart.Reader { return newForm(t, "other", png) },
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.UpdateAvatar400Response{},
		},
		{
			name: "unsupported type",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			form: func(t *testing.T) *multipart.Reader { return newForm(t, avatarField, []byte("text")) },
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateAvatar(gomock.Any(), userID, []byte("text"), "text/plain; charset=utf-8").
					Return(models.User{}, models.ErrUnsupportedFileType)
			},
			expected: api.UpdateAvatar415Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			resp, err := NewHandler(uc).UpdateAvatar(tt.ctx, api.UpdateAvatarRequestObject{Body: tt.form(t)})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestMiddleware(t *testing.T) {
	userID := uuid.NewV4()

	tests := []struct {
		name         string
		path         string
		cookie       *http.Cookie
		prepare      func(uc *mocks.MockUsecase)
		expectedCode int
		expectedUser bool
	}{
		{
			name:         "public path without cookie",
			path:         "/api/auth/signin",
			prepare:      func(uc *mocks.MockUsecase) {},
			expectedCode: http.StatusOK,
		},
		{
			name:         "no cookie",
			path:         "/api/notes/all",
			prepare:      func(uc *mocks.MockUsecase) {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "invalid token",
			path:   "/api/notes/all",
			cookie: &http.Cookie{Name: accessCookieName, Value: "bad"},
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Authenticate(gomock.Any(), "bad").Return(uuid.Nil, models.ErrInvalidToken)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "ok",
			path:   "/api/notes/all",
			cookie: &http.Cookie{Name: accessCookieName, Value: "good"},
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().Authenticate(gomock.Any(), "good").Return(userID, nil)
			},
			expectedCode: http.StatusOK,
			expectedUser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			var gotUser bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, ok := ctxutil.GetUserID(r.Context())
				gotUser = ok && id == userID
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()

			NewHandler(uc).Middleware(next).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
			assert.Equal(t, tt.expectedUser, gotUser)
		})
	}
}

func TestHandlersInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := mocks.NewMockUsecase(ctrl)
	userID := uuid.NewV4()
	dbErr := errors.New("db is down")

	uc.EXPECT().SignUp(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.User{}, models.AuthTokens{}, dbErr)
	uc.EXPECT().SignIn(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.User{}, models.AuthTokens{}, dbErr)
	uc.EXPECT().Refresh(gomock.Any(), gomock.Any()).Return(models.AuthTokens{}, dbErr)
	uc.EXPECT().Logout(gomock.Any(), gomock.Any()).Return(dbErr)
	uc.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(models.User{}, dbErr)
	uc.EXPECT().ChangePassword(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(models.AuthTokens{}, dbErr)
	uc.EXPECT().UpdateAvatar(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(models.User{}, dbErr)

	h := NewHandler(uc)
	refresh := []*http.Cookie{{Name: refreshCookieName, Value: "refresh"}}

	withCookies(refresh, func(ctx context.Context) {
		_, err := h.SignUp(ctx, api.SignUpRequestObject{Body: &api.SignUpJSONRequestBody{}})
		assert.ErrorIs(t, err, dbErr)

		_, err = h.SignIn(ctx, api.SignInRequestObject{Body: &api.SignInJSONRequestBody{}})
		assert.ErrorIs(t, err, dbErr)

		_, err = h.Refresh(ctx, api.RefreshRequestObject{})
		assert.ErrorIs(t, err, dbErr)

		_, err = h.Logout(ctx, api.LogoutRequestObject{})
		assert.ErrorIs(t, err, dbErr)

		ctx = ctxutil.WithUserID(ctx, userID)

		_, err = h.GetCurrentUser(ctx, api.GetCurrentUserRequestObject{})
		assert.ErrorIs(t, err, dbErr)

		_, err = h.ChangePassword(ctx, api.ChangePasswordRequestObject{Body: &api.ChangePasswordJSONRequestBody{}})
		assert.ErrorIs(t, err, dbErr)

		_, err = h.UpdateAvatar(ctx, api.UpdateAvatarRequestObject{Body: newForm(t, avatarField, []byte("x"))})
		assert.ErrorIs(t, err, dbErr)
	})
}

func TestUpdateAvatarRequestTooLarge(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := mocks.NewMockUsecase(ctrl)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(avatarField, "avatar")
	require.NoError(t, err)
	_, err = part.Write([]byte("avatar"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	body := http.MaxBytesReader(nil, io.NopCloser(&buf), 10)
	req := api.UpdateAvatarRequestObject{Body: multipart.NewReader(body, w.Boundary())}

	resp, err := NewHandler(uc).UpdateAvatar(ctxutil.WithUserID(context.Background(), uuid.NewV4()), req)

	require.NoError(t, err)
	assert.Equal(t, api.UpdateAvatar413Response{}, resp)
}

func TestToAPIUser(t *testing.T) {
	github := api.Github

	tests := []struct {
		name     string
		user     models.User
		expected api.User
	}{
		{
			name:     "password account",
			user:     models.User{Login: "seyran"},
			expected: api.User{Login: "seyran"},
		},
		{
			name:     "github account with 2fa",
			user:     models.User{Login: "a@b.ru", IsExternal: true, Is2FAEnabled: true},
			expected: api.User{Login: "a@b.ru", OauthProvider: &github, Is2faEnabled: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, toAPIUser(tt.user))
		})
	}
}

func TestSignInWithTwoFactor(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := mocks.NewMockUsecase(ctrl)
	pending := models.AuthTokens{TwoFactorToken: "pending", TwoFactorExpiresAt: time.Now().Add(5 * time.Minute)}
	uc.EXPECT().SignIn(gomock.Any(), "seyran", "password123").Return(models.User{}, pending, nil)

	var resp api.SignInResponseObject
	var err error
	set := withCookies(nil, func(ctx context.Context) {
		body := &api.SignInJSONRequestBody{Login: "seyran", Password: "password123"}
		resp, err = NewHandler(uc).SignIn(ctx, api.SignInRequestObject{Body: body})
	})

	require.NoError(t, err)
	assert.Equal(t, api.SignIn202Response{}, resp)
	assert.Equal(t, "pending", set[twoFactorCookieName].Value)
	assert.NotContains(t, set, accessCookieName)
	assert.NotContains(t, set, refreshCookieName)
}

func TestVerify2FA(t *testing.T) {
	user := models.User{ID: uuid.NewV4(), Login: "seyran", Is2FAEnabled: true}
	pendingCookie := []*http.Cookie{{Name: twoFactorCookieName, Value: "pending"}}
	body := &api.Verify2FAJSONRequestBody{Code: "123456"}

	tests := []struct {
		name     string
		cookies  []*http.Cookie
		prepare  func(uc *mocks.MockUsecase)
		expected api.Verify2FAResponseObject
	}{
		{
			name:    "ok",
			cookies: pendingCookie,
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().VerifyTwoFactor(gomock.Any(), "pending", "123456").Return(user, tokens, nil)
			},
			expected: api.Verify2FA200JSONResponse(toAPIUser(user)),
		},
		{
			name:     "no cookie",
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.Verify2FA401Response{},
		},
		{
			name:    "expired token",
			cookies: pendingCookie,
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().VerifyTwoFactor(gomock.Any(), "pending", "123456").Return(models.User{}, models.AuthTokens{}, models.ErrInvalidToken)
			},
			expected: api.Verify2FA401Response{},
		},
		{
			name:    "wrong code",
			cookies: pendingCookie,
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().VerifyTwoFactor(gomock.Any(), "pending", "123456").Return(models.User{}, models.AuthTokens{}, models.ErrInvalidCode)
			},
			expected: api.Verify2FA403Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			var resp api.Verify2FAResponseObject
			var err error
			set := withCookies(tt.cookies, func(ctx context.Context) {
				resp, err = NewHandler(uc).Verify2FA(ctx, api.Verify2FARequestObject{Body: body})
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
			if _, ok := tt.expected.(api.Verify2FA200JSONResponse); ok {
				assert.Equal(t, "access", set[accessCookieName].Value)
				assert.Negative(t, set[twoFactorCookieName].MaxAge)
			}
		})
	}
}

func TestTwoFactorSettings(t *testing.T) {
	userID := uuid.NewV4()
	ctx := ctxutil.WithUserID(context.Background(), userID)
	code := &api.TwoFactorCodeRequest{Code: "123456"}

	t.Run("generate", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		uc := mocks.NewMockUsecase(ctrl)
		uc.EXPECT().Generate2FA(gomock.Any(), userID).Return([]byte("png"), nil)
		uc.EXPECT().Generate2FA(gomock.Any(), userID).Return(nil, models.ErrTwoFactorEnabled)
		h := NewHandler(uc)

		resp, err := h.Generate2FA(ctx, api.Generate2FARequestObject{})
		require.NoError(t, err)
		assert.IsType(t, api.Generate2FA200ImagepngResponse{}, resp)

		resp, err = h.Generate2FA(ctx, api.Generate2FARequestObject{})
		require.NoError(t, err)
		assert.Equal(t, api.Generate2FA409Response{}, resp)
	})

	t.Run("enable", func(t *testing.T) {
		tests := []struct {
			ucErr    error
			expected api.Enable2FAResponseObject
		}{
			{expected: api.Enable2FA200Response{}},
			{ucErr: models.ErrValidation, expected: api.Enable2FA400Response{}},
			{ucErr: models.ErrInvalidCode, expected: api.Enable2FA403Response{}},
		}

		for _, tt := range tests {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			uc.EXPECT().Enable2FA(gomock.Any(), userID, "123456").Return(tt.ucErr)

			resp, err := NewHandler(uc).Enable2FA(ctx, api.Enable2FARequestObject{Body: code})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		}
	})

	t.Run("disable", func(t *testing.T) {
		tests := []struct {
			ucErr    error
			expected api.Disable2FAResponseObject
		}{
			{expected: api.Disable2FA200Response{}},
			{ucErr: models.ErrInvalidCode, expected: api.Disable2FA403Response{}},
		}

		for _, tt := range tests {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			uc.EXPECT().Disable2FA(gomock.Any(), userID, "123456").Return(tt.ucErr)

			resp, err := NewHandler(uc).Disable2FA(ctx, api.Disable2FARequestObject{Body: code})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		}
	})
}

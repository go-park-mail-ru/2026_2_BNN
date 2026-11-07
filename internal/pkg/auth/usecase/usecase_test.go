package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/auth/mocks"

	"github.com/dgryski/dgoogauth"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const secret = "test_secret"

var errDB = errors.New("db is down")

func TestSignUp(t *testing.T) {
	tests := []struct {
		name        string
		login       string
		password    string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name:     "ok",
			login:    "seyran",
			password: "password123",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(nil)
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:        "short login",
			login:       "ab",
			password:    "password123",
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrValidation,
		},
		{
			name:        "short password",
			login:       "seyran",
			password:    "short",
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrValidation,
		},
		{
			name:     "user exists",
			login:    "seyran",
			password: "password123",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(models.ErrUserExists)
			},
			expectedErr: models.ErrUserExists,
		},
		{
			name:     "save token error",
			login:    "seyran",
			password: "password123",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(nil)
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			user, tokens, err := NewUsecase(repo, nil, nil, secret).SignUp(context.Background(), tt.login, tt.password)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.login, user.Login)
			assert.Empty(t, user.Avatar)
			assert.True(t, verifyPassword(tt.password, user.PasswordHash))
			assert.NotEmpty(t, tokens.AccessToken)
			assert.NotEmpty(t, tokens.RefreshToken)
		})
	}
}

func TestSignIn(t *testing.T) {
	hash, err := hashPassword("password123")
	require.NoError(t, err)
	user := models.User{ID: uuid.NewV4(), Login: "seyran", PasswordHash: hash}

	tests := []struct {
		name        string
		password    string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name:     "ok",
			password: "password123",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByLogin(gomock.Any(), "seyran").Return(user, nil)
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:     "wrong password",
			password: "wrongpassword",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByLogin(gomock.Any(), "seyran").Return(user, nil)
			},
			expectedErr: models.ErrInvalidCredentials,
		},
		{
			name:     "user not found",
			password: "password123",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByLogin(gomock.Any(), "seyran").Return(models.User{}, models.ErrUserNotFound)
			},
			expectedErr: models.ErrInvalidCredentials,
		},
		{
			name:     "oauth account",
			password: "any",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByLogin(gomock.Any(), "seyran").Return(models.User{ID: user.ID, Login: "seyran", IsExternal: true}, nil)
			},
			expectedErr: models.ErrOAuthAccount,
		},
		{
			name:     "read user error",
			password: "password123",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByLogin(gomock.Any(), "seyran").Return(models.User{}, errDB)
			},
			expectedErr: errDB,
		},
		{
			name:     "save token error",
			password: "password123",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByLogin(gomock.Any(), "seyran").Return(user, nil)
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			got, tokens, err := NewUsecase(repo, nil, nil, secret).SignIn(context.Background(), "seyran", tt.password)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, user.ID, got.ID)
			assert.NotEmpty(t, tokens.AccessToken)
		})
	}
}

func TestRefresh(t *testing.T) {
	userID := uuid.NewV4()
	hash := hashRefreshToken("old-refresh")

	tests := []struct {
		name        string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name: "ok",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().DeleteRefreshToken(gomock.Any(), hash).
					Return(models.RefreshToken{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil)
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "unknown token",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().DeleteRefreshToken(gomock.Any(), hash).Return(models.RefreshToken{}, models.ErrRefreshTokenNotFound)
			},
			expectedErr: models.ErrInvalidToken,
		},
		{
			name: "expired token",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().DeleteRefreshToken(gomock.Any(), hash).
					Return(models.RefreshToken{UserID: userID, ExpiresAt: time.Now().Add(-time.Hour)}, nil)
			},
			expectedErr: models.ErrInvalidToken,
		},
		{
			name: "db error",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().DeleteRefreshToken(gomock.Any(), hash).Return(models.RefreshToken{}, errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			u := NewUsecase(repo, nil, nil, secret)
			tokens, err := u.Refresh(context.Background(), "old-refresh")

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.NotEqual(t, "old-refresh", tokens.RefreshToken)

			id, err := u.Authenticate(context.Background(), tokens.AccessToken)
			require.NoError(t, err)
			assert.Equal(t, userID, id)
		})
	}
}

func TestLogout(t *testing.T) {
	hash := hashRefreshToken("refresh")

	tests := []struct {
		name        string
		repoErr     error
		expectedErr error
	}{
		{name: "ok"},
		{name: "already deleted", repoErr: models.ErrRefreshTokenNotFound},
		{name: "db error", repoErr: errDB, expectedErr: errDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			repo.EXPECT().DeleteRefreshToken(gomock.Any(), hash).Return(models.RefreshToken{}, tt.repoErr)

			err := NewUsecase(repo, nil, nil, secret).Logout(context.Background(), "refresh")

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestAuthenticate(t *testing.T) {
	userID := uuid.NewV4()
	u := NewUsecase(nil, nil, nil, secret)

	valid, err := u.newToken(userID, accessAudience, time.Now().Add(time.Minute))
	require.NoError(t, err)

	expired, err := u.newToken(userID, accessAudience, time.Now().Add(-time.Minute))
	require.NoError(t, err)

	foreign, err := NewUsecase(nil, nil, nil, "other_secret").newToken(userID, accessAudience, time.Now().Add(time.Minute))
	require.NoError(t, err)

	twoFactor, err := u.newToken(userID, twoFactorAudience, time.Now().Add(time.Minute))
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectedErr error
	}{
		{name: "ok", token: valid},
		{name: "2fa token", token: twoFactor, expectedErr: models.ErrInvalidToken},
		{name: "expired", token: expired, expectedErr: models.ErrInvalidToken},
		{name: "other secret", token: foreign, expectedErr: models.ErrInvalidToken},
		{name: "garbage", token: "not-a-token", expectedErr: models.ErrInvalidToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := u.Authenticate(context.Background(), tt.token)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				assert.Equal(t, uuid.Nil, id)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, userID, id)
		})
	}
}

func TestGetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepo(ctrl)
	user := models.User{ID: uuid.NewV4(), Login: "seyran"}

	repo.EXPECT().ReadUserByID(gomock.Any(), user.ID).Return(user, nil)

	got, err := NewUsecase(repo, nil, nil, secret).GetUser(context.Background(), user.ID)

	require.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestChangePassword(t *testing.T) {
	userID := uuid.NewV4()

	hash, err := hashPassword("oldpassword")
	require.NoError(t, err)

	user := models.User{ID: userID, PasswordHash: hash}

	tests := []struct {
		name        string
		oldPassword string
		newPassword string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name:        "ok",
			oldPassword: "oldpassword",
			newPassword: "newpassword",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(user, nil)
				repo.EXPECT().UpdatePassword(gomock.Any(), userID, gomock.Any()).
					DoAndReturn(func(ctx context.Context, id uuid.UUID, newHash []byte) error {
						assert.True(t, verifyPassword("newpassword", newHash))
						return nil
					})
				repo.EXPECT().DeleteUserRefreshTokens(gomock.Any(), userID).Return(nil)
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:        "wrong old password",
			oldPassword: "wrongpassword",
			newPassword: "newpassword",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(user, nil)
			},
			expectedErr: models.ErrWrongPassword,
		},
		{
			name:        "oauth account",
			oldPassword: "oldpassword",
			newPassword: "newpassword",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(models.User{ID: userID, IsExternal: true}, nil)
			},
			expectedErr: models.ErrOAuthAccount,
		},
		{
			name:        "new password too short",
			oldPassword: "oldpassword",
			newPassword: "short",
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrValidation,
		},
		{
			name:        "user not found",
			oldPassword: "oldpassword",
			newPassword: "newpassword",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(models.User{}, models.ErrUserNotFound)
			},
			expectedErr: models.ErrUserNotFound,
		},
		{
			name:        "update error",
			oldPassword: "oldpassword",
			newPassword: "newpassword",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(user, nil)
				repo.EXPECT().UpdatePassword(gomock.Any(), userID, gomock.Any()).Return(errDB)
			},
			expectedErr: errDB,
		},
		{
			name:        "delete tokens error",
			oldPassword: "oldpassword",
			newPassword: "newpassword",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(user, nil)
				repo.EXPECT().UpdatePassword(gomock.Any(), userID, gomock.Any()).Return(nil)
				repo.EXPECT().DeleteUserRefreshTokens(gomock.Any(), userID).Return(errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			tokens, err := NewUsecase(repo, nil, nil, secret).ChangePassword(context.Background(), userID, tt.oldPassword, tt.newPassword)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, tokens.AccessToken)
			assert.NotEmpty(t, tokens.RefreshToken)
		})
	}
}

func TestUpdateAvatar(t *testing.T) {
	userID := uuid.NewV4()
	data := []byte("image")

	tests := []struct {
		name           string
		contentType    string
		prepare        func(repo *mocks.MockRepo, storage *mocks.MockStorage)
		expectedErr    error
		expectedAvatar string
	}{
		{
			name:        "ok",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, storage *mocks.MockStorage) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).
					Return(models.User{ID: userID, Avatar: "/avatars/old.jpg"}, nil)
				storage.EXPECT().PublicPath(gomock.Any()).Return("/avatars/new.png", nil)
				storage.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(nil)
				repo.EXPECT().UpdateAvatar(gomock.Any(), gomock.Any()).Return(nil)
				storage.EXPECT().DeleteFile(gomock.Any(), "old.jpg").Return(nil)
			},
			expectedAvatar: "/avatars/new.png",
		},
		{
			name:        "first avatar",
			contentType: "image/jpeg",
			prepare: func(repo *mocks.MockRepo, storage *mocks.MockStorage) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(models.User{ID: userID}, nil)
				storage.EXPECT().PublicPath(gomock.Any()).Return("/avatars/new.jpg", nil)
				storage.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/jpeg").Return(nil)
				repo.EXPECT().UpdateAvatar(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedAvatar: "/avatars/new.jpg",
		},
		{
			name:        "unsupported type",
			contentType: "text/plain; charset=utf-8",
			prepare:     func(repo *mocks.MockRepo, storage *mocks.MockStorage) {},
			expectedErr: models.ErrUnsupportedFileType,
		},
		{
			name:        "read user error",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, storage *mocks.MockStorage) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(models.User{}, errDB)
			},
			expectedErr: errDB,
		},
		{
			name:        "public path error",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, storage *mocks.MockStorage) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(models.User{ID: userID}, nil)
				storage.EXPECT().PublicPath(gomock.Any()).Return("", errDB)
			},
			expectedErr: errDB,
		},
		{
			name:        "put file error",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, storage *mocks.MockStorage) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(models.User{ID: userID}, nil)
				storage.EXPECT().PublicPath(gomock.Any()).Return("/avatars/new.png", nil)
				storage.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(errDB)
			},
			expectedErr: errDB,
		},
		{
			name:        "db error removes new file",
			contentType: "image/png",
			prepare: func(repo *mocks.MockRepo, storage *mocks.MockStorage) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).
					Return(models.User{ID: userID, Avatar: "/avatars/old.jpg"}, nil)
				storage.EXPECT().PublicPath(gomock.Any()).Return("/avatars/new.png", nil)
				storage.EXPECT().PutFile(gomock.Any(), gomock.Any(), data, "image/png").Return(nil)
				repo.EXPECT().UpdateAvatar(gomock.Any(), gomock.Any()).Return(errDB)
				storage.EXPECT().DeleteFile(gomock.Any(), gomock.Not("old.jpg")).Return(nil)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			storage := mocks.NewMockStorage(ctrl)
			tt.prepare(repo, storage)

			user, err := NewUsecase(repo, storage, nil, secret).UpdateAvatar(context.Background(), userID, data, tt.contentType)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAvatar, user.Avatar)
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	hash, err := hashPassword("password123")
	require.NoError(t, err)

	tests := []struct {
		name     string
		password string
		hash     string
		expected bool
	}{
		{name: "correct", password: "password123", hash: string(hash), expected: true},
		{name: "wrong password", password: "password124", hash: string(hash), expected: false},
		{name: "empty hash", password: "password123", hash: "", expected: false},
		{name: "other algorithm", password: "password123", hash: "$bcrypt$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA", expected: false},
		{name: "wrong version", password: "password123", hash: "$argon2id$v=1$m=1,t=1,p=1$c2FsdA$aGFzaA", expected: false},
		{name: "bad params", password: "password123", hash: "$argon2id$v=19$m=1,t=0,p=1$c2FsdA$aGFzaA", expected: false},
		{name: "bad salt", password: "password123", hash: "$argon2id$v=19$m=1,t=1,p=1$!!!$aGFzaA", expected: false},
		{name: "bad hash", password: "password123", hash: "$argon2id$v=19$m=1,t=1,p=1$c2FsdA$!!!", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, verifyPassword(tt.password, []byte(tt.hash)))
		})
	}
}

const totpSecret = "JBSWY3DPEHPK3PXP"

func currentCode() string {
	return fmt.Sprintf("%06d", dgoogauth.ComputeCode(totpSecret, time.Now().Unix()/30))
}

func TestSignInWithTwoFactor(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepo(ctrl)
	hash, err := hashPassword("password123")
	require.NoError(t, err)
	user := models.User{ID: uuid.NewV4(), Login: "seyran", PasswordHash: hash, Is2FAEnabled: true}

	repo.EXPECT().ReadUserByLogin(gomock.Any(), "seyran").Return(user, nil)

	u := NewUsecase(repo, nil, nil, secret)
	_, tokens, err := u.SignIn(context.Background(), "seyran", "password123")

	require.NoError(t, err)
	assert.Empty(t, tokens.AccessToken)
	assert.Empty(t, tokens.RefreshToken)
	assert.NotEmpty(t, tokens.TwoFactorToken)

	id, err := u.parseToken(tokens.TwoFactorToken, twoFactorAudience)
	require.NoError(t, err)
	assert.Equal(t, user.ID, id)
}

func TestVerifyTwoFactor(t *testing.T) {
	userID := uuid.NewV4()
	u := NewUsecase(nil, nil, nil, secret)
	pending, err := u.newToken(userID, twoFactorAudience, time.Now().Add(time.Minute))
	require.NoError(t, err)
	access, err := u.newToken(userID, accessAudience, time.Now().Add(time.Minute))
	require.NoError(t, err)
	user := models.User{ID: userID, Is2FAEnabled: true, TOTPSecret: totpSecret}

	tests := []struct {
		name        string
		token       string
		code        string
		prepare     func(repo *mocks.MockRepo)
		expectedErr error
	}{
		{
			name:  "ok",
			token: pending,
			code:  currentCode(),
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(user, nil)
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:  "wrong code",
			token: pending,
			code:  "000000",
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(user, nil)
			},
			expectedErr: models.ErrInvalidCode,
		},
		{
			name:        "access token instead of 2fa token",
			token:       access,
			code:        currentCode(),
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrInvalidToken,
		},
		{
			name:        "garbage token",
			token:       "not-a-token",
			code:        currentCode(),
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrInvalidToken,
		},
		{
			name:  "read user error",
			token: pending,
			code:  currentCode(),
			prepare: func(repo *mocks.MockRepo) {
				repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(models.User{}, errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			tt.prepare(repo)

			got, tokens, err := NewUsecase(repo, nil, nil, secret).VerifyTwoFactor(context.Background(), tt.token, tt.code)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, userID, got.ID)
			assert.NotEmpty(t, tokens.AccessToken)
			assert.NotEmpty(t, tokens.RefreshToken)
		})
	}
}

func TestGenerate2FA(t *testing.T) {
	userID := uuid.NewV4()

	t.Run("ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepo(ctrl)
		repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(models.User{ID: userID, Login: "seyran"}, nil)
		repo.EXPECT().UpdateTOTPSecret(gomock.Any(), userID, gomock.Any()).Return(nil)

		png, err := NewUsecase(repo, nil, nil, secret).Generate2FA(context.Background(), userID)

		require.NoError(t, err)
		assert.NotEmpty(t, png)
	})

	t.Run("already enabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepo(ctrl)
		repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(models.User{ID: userID, Is2FAEnabled: true}, nil)

		_, err := NewUsecase(repo, nil, nil, secret).Generate2FA(context.Background(), userID)

		assert.ErrorIs(t, err, models.ErrTwoFactorEnabled)
	})
}

func TestEnable2FA(t *testing.T) {
	userID := uuid.NewV4()

	tests := []struct {
		name        string
		user        models.User
		code        string
		enable      bool
		expectedErr error
	}{
		{name: "ok", user: models.User{ID: userID, TOTPSecret: totpSecret}, code: currentCode(), enable: true},
		{name: "no secret", user: models.User{ID: userID}, code: currentCode(), expectedErr: models.ErrValidation},
		{name: "wrong code", user: models.User{ID: userID, TOTPSecret: totpSecret}, code: "000000", expectedErr: models.ErrInvalidCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(tt.user, nil)
			if tt.enable {
				repo.EXPECT().Set2FAStatus(gomock.Any(), userID, true).Return(nil)
			}

			err := NewUsecase(repo, nil, nil, secret).Enable2FA(context.Background(), userID, tt.code)

			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestDisable2FA(t *testing.T) {
	userID := uuid.NewV4()
	enabled := models.User{ID: userID, Is2FAEnabled: true, TOTPSecret: totpSecret}

	tests := []struct {
		name        string
		user        models.User
		code        string
		disable     bool
		expectedErr error
	}{
		{name: "ok", user: enabled, code: currentCode(), disable: true},
		{name: "wrong code", user: enabled, code: "000000", expectedErr: models.ErrInvalidCode},
		{name: "not enabled", user: models.User{ID: userID}, code: currentCode(), expectedErr: models.ErrInvalidCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			repo.EXPECT().ReadUserByID(gomock.Any(), userID).Return(tt.user, nil)
			if tt.disable {
				repo.EXPECT().Set2FAStatus(gomock.Any(), userID, false).Return(nil)
				repo.EXPECT().UpdateTOTPSecret(gomock.Any(), userID, "").Return(nil)
			}

			err := NewUsecase(repo, nil, nil, secret).Disable2FA(context.Background(), userID, tt.code)

			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestGetOAuthURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	oauth := mocks.NewMockOAuthProvider(ctrl)
	oauth.EXPECT().AuthURL("state-1").Return("https://github.com/login")

	u := NewUsecase(nil, nil, oauth, secret)

	assert.Equal(t, "https://github.com/login", u.GetOAuthURL("github", "state-1"))
	assert.Empty(t, u.GetOAuthURL("google", "state-1"))
}

func TestHandleOAuthCallback(t *testing.T) {
	profile := models.OAuthProfile{Email: "a@b.ru", AvatarURL: "https://avatars/1"}
	existing := models.User{ID: uuid.NewV4(), Login: "a@b.ru", Email: "a@b.ru", IsExternal: true}

	tests := []struct {
		name        string
		prepare     func(repo *mocks.MockRepo, oauth *mocks.MockOAuthProvider)
		expectedErr error
		twoFactor   bool
	}{
		{
			name: "existing user",
			prepare: func(repo *mocks.MockRepo, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().ReadProfile(gomock.Any(), "code").Return(profile, nil)
				repo.EXPECT().ReadUserByEmail(gomock.Any(), "a@b.ru").Return(existing, nil)
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "new user",
			prepare: func(repo *mocks.MockRepo, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().ReadProfile(gomock.Any(), "code").Return(profile, nil)
				repo.EXPECT().ReadUserByEmail(gomock.Any(), "a@b.ru").Return(models.User{}, models.ErrUserNotFound)
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, user models.User) error {
					assert.Equal(t, "a@b.ru", user.Login)
					assert.Equal(t, "a@b.ru", user.Email)
					assert.Equal(t, "https://avatars/1", user.Avatar)
					assert.True(t, user.IsExternal)
					assert.Nil(t, user.PasswordHash)
					return nil
				})
				repo.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "user with 2fa",
			prepare: func(repo *mocks.MockRepo, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().ReadProfile(gomock.Any(), "code").Return(profile, nil)
				withTwoFactor := existing
				withTwoFactor.Is2FAEnabled = true
				repo.EXPECT().ReadUserByEmail(gomock.Any(), "a@b.ru").Return(withTwoFactor, nil)
			},
			twoFactor: true,
		},
		{
			name: "github error",
			prepare: func(repo *mocks.MockRepo, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().ReadProfile(gomock.Any(), "code").Return(models.OAuthProfile{}, errDB)
			},
			expectedErr: errDB,
		},
		{
			name: "read user error",
			prepare: func(repo *mocks.MockRepo, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().ReadProfile(gomock.Any(), "code").Return(profile, nil)
				repo.EXPECT().ReadUserByEmail(gomock.Any(), "a@b.ru").Return(models.User{}, errDB)
			},
			expectedErr: errDB,
		},
		{
			name: "create user error",
			prepare: func(repo *mocks.MockRepo, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().ReadProfile(gomock.Any(), "code").Return(profile, nil)
				repo.EXPECT().ReadUserByEmail(gomock.Any(), "a@b.ru").Return(models.User{}, models.ErrUserNotFound)
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(errDB)
			},
			expectedErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRepo(ctrl)
			oauth := mocks.NewMockOAuthProvider(ctrl)
			tt.prepare(repo, oauth)

			user, tokens, err := NewUsecase(repo, nil, oauth, secret).HandleOAuthCallback(context.Background(), "github", "code")

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "a@b.ru", user.Email)
			if tt.twoFactor {
				assert.NotEmpty(t, tokens.TwoFactorToken)
				assert.Empty(t, tokens.AccessToken)
			} else {
				assert.NotEmpty(t, tokens.AccessToken)
			}
		})
	}
}

func TestHandleOAuthCallbackUnknownProvider(t *testing.T) {
	_, _, err := NewUsecase(nil, nil, nil, secret).HandleOAuthCallback(context.Background(), "google", "code")

	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestHTTPSURL(t *testing.T) {
	tests := []struct {
		raw      string
		expected string
	}{
		{raw: "https://avatars.githubusercontent.com/u/1?v=4", expected: "https://avatars.githubusercontent.com/u/1?v=4"},
		{raw: "javascript:alert(1)", expected: ""},
		{raw: "http://example.com/a.png", expected: ""},
		{raw: "", expected: ""},
		{raw: "%zz", expected: ""},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, httpsURL(tt.raw), tt.raw)
	}
}

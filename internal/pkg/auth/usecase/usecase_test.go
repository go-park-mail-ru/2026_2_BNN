package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/auth/mocks"

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

			user, tokens, err := NewUsecase(repo, nil, secret).SignUp(context.Background(), tt.login, tt.password)

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
			name:        "validation",
			password:    "short",
			prepare:     func(repo *mocks.MockRepo) {},
			expectedErr: models.ErrValidation,
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

			got, tokens, err := NewUsecase(repo, nil, secret).SignIn(context.Background(), "seyran", tt.password)

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

			u := NewUsecase(repo, nil, secret)
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

			err := NewUsecase(repo, nil, secret).Logout(context.Background(), "refresh")

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
	u := NewUsecase(nil, nil, secret)

	valid, err := u.newAccessToken(userID, time.Now().Add(time.Minute))
	require.NoError(t, err)

	expired, err := u.newAccessToken(userID, time.Now().Add(-time.Minute))
	require.NoError(t, err)

	foreign, err := NewUsecase(nil, nil, "other_secret").newAccessToken(userID, time.Now().Add(time.Minute))
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectedErr error
	}{
		{name: "ok", token: valid},
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

	got, err := NewUsecase(repo, nil, secret).GetUser(context.Background(), user.ID)

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

			tokens, err := NewUsecase(repo, nil, secret).ChangePassword(context.Background(), userID, tt.oldPassword, tt.newPassword)

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

			user, err := NewUsecase(repo, storage, secret).UpdateAvatar(context.Background(), userID, data, tt.contentType)

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

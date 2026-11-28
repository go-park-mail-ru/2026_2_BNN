package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errDependency = errors.New("service unavailable")

func newUsecase(t *testing.T) (*mocks.MockRepo, *mocks.MockAuthClient, *Usecase) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepo(ctrl)
	auth := mocks.NewMockAuthClient(ctrl)
	return repo, auth, NewUsecase(repo, auth)
}

func configInput(valueType, value string) onlineconf.KeyInput {
	return onlineconf.KeyInput{
		Scope: onlineconf.ScopeBackend, Key: "test.value", Kind: onlineconf.KindConfig,
		Description: "Настройка", IsPublic: true,
		Config: &models.OnlineConfConfig{ValueType: valueType, Value: json.RawMessage(value)},
	}
}

func flagInput() onlineconf.KeyInput {
	return onlineconf.KeyInput{
		Scope: onlineconf.ScopeBackend, Key: "test.flag", Kind: onlineconf.KindFlag,
		Flag: &onlineconf.FlagInput{Enabled: true, RolloutPercent: 25},
	}
}

func expectSnapshotRefresh(repo *mocks.MockRepo) {
	repo.EXPECT().ReadSnapshot(gomock.Any()).Return(models.Snapshot{Revision: 1}, nil)
}

func TestCreateConfig(t *testing.T) {
	tests := []struct {
		name, valueType, value string
	}{
		{"string", "string", `"Привет"`},
		{"empty string", "string", `""`},
		{"number", "number", "1.25"},
		{"large integer", "number", "9007199254740993"},
		{"scientific number", "number", "1e3"},
		{"boolean", "boolean", "false"},
		{"object", "object", `{"nested":{"on":true},"items":[1,null]}`},
		{"empty object", "object", `{}`},
		{"array", "array", `[1,"two",false,null]`},
		{"empty array", "array", `[]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, uc := newUsecase(t)
			actor := uuid.NewV4()
			input := configInput(tt.valueType, tt.value)
			var saved models.OnlineConfKey
			repo.EXPECT().CreateKey(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error) {
					assert.NotEqual(t, uuid.Nil, key.ID)
					assert.Equal(t, actor, key.UpdatedBy.ID)
					assert.Equal(t, input.Scope, key.Scope)
					assert.Equal(t, input.Key, key.Key)
					assert.Equal(t, input.Kind, key.Kind)
					assert.Equal(t, input.Description, key.Description)
					assert.Equal(t, input.IsPublic, key.IsPublic)
					assert.Equal(t, input.Config, key.Config)
					assert.Nil(t, key.Flag)
					key.Version = 1
					key.UpdatedBy.Login = "liza"
					key.UpdatedAt = time.Now().UTC()
					saved = key
					return key, nil
				})

			expectSnapshotRefresh(repo)
			got, err := uc.CreateKey(context.Background(), actor, input)

			require.NoError(t, err)
			assert.Equal(t, saved, got)
			assert.Equal(t, tt.value, string(got.Config.Value))
		})
	}
}

func TestCreateConfigCopiesValue(t *testing.T) {
	repo, _, uc := newUsecase(t)
	input := configInput("object", `{"on":true}`)
	repo.EXPECT().CreateKey(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error) { return key, nil })
	expectSnapshotRefresh(repo)

	got, err := uc.CreateKey(context.Background(), uuid.NewV4(), input)
	require.NoError(t, err)
	input.Config.Value[0] = '!'

	assert.Equal(t, `{"on":true}`, string(got.Config.Value))
}

func TestInvalidKeyInput(t *testing.T) {
	tests := []struct {
		name   string
		change func(*onlineconf.KeyInput)
	}{
		{"empty name", func(in *onlineconf.KeyInput) { in.Key = "" }},
		{"long name", func(in *onlineconf.KeyInput) { in.Key = strings.Repeat("a", 101) }},
		{"non latin name", func(in *onlineconf.KeyInput) { in.Key = "ключ" }},
		{"digit prefix", func(in *onlineconf.KeyInput) { in.Key = "1key" }},
		{"name with space", func(in *onlineconf.KeyInput) { in.Key = "key name" }},
		{"name with slash", func(in *onlineconf.KeyInput) { in.Key = "key/name" }},
		{"empty scope", func(in *onlineconf.KeyInput) { in.Scope = "" }},
		{"unknown scope", func(in *onlineconf.KeyInput) { in.Scope = "payments" }},
		{"unknown kind", func(in *onlineconf.KeyInput) { in.Kind = "toggle" }},
		{"missing config", func(in *onlineconf.KeyInput) { in.Config = nil }},
		{"missing flag", func(in *onlineconf.KeyInput) { in.Kind = "flag"; in.Config = nil }},
		{"config and flag", func(in *onlineconf.KeyInput) { in.Flag = &onlineconf.FlagInput{} }},
		{"flag and config", func(in *onlineconf.KeyInput) { in.Kind = "flag"; in.Flag = &onlineconf.FlagInput{} }},
		{"invalid json", func(in *onlineconf.KeyInput) { in.Config.Value = json.RawMessage(`{"on":`) }},
		{"empty value", func(in *onlineconf.KeyInput) { in.Config.Value = nil }},
		{"multiple json values", func(in *onlineconf.KeyInput) { in.Config.Value = json.RawMessage("1 2") }},
		{"unknown value type", func(in *onlineconf.KeyInput) { in.Config.ValueType = "integer" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, uc := newUsecase(t)
			input := configInput("number", "42")
			tt.change(&input)

			_, err := uc.CreateKey(context.Background(), uuid.NewV4(), input)

			assert.ErrorIs(t, err, models.ErrValidation)
		})
	}
}

func TestConfigTypeMismatch(t *testing.T) {
	for _, valueType := range []string{"string", "number", "boolean", "object", "array"} {
		for _, value := range []string{"null", `"text"`, "42", "true", `{}`, `[]`} {
			matches := (valueType == "string" && value == `"text"`) ||
				(valueType == "number" && value == "42") || (valueType == "boolean" && value == "true") ||
				(valueType == "object" && value == `{}`) || (valueType == "array" && value == `[]`)
			if matches {
				continue
			}
			t.Run(valueType+"/"+value, func(t *testing.T) {
				_, _, uc := newUsecase(t)

				_, err := uc.CreateKey(context.Background(), uuid.NewV4(), configInput(valueType, value))

				assert.ErrorIs(t, err, models.ErrValidation)
			})
		}
	}
}

func TestAttachmentSizeValidation(t *testing.T) {
	for _, value := range []string{"1048575", "1048576", "1572864", "10485760", "10485761", "1048576.1", "-1", "0"} {
		t.Run(value, func(t *testing.T) {
			repo, _, uc := newUsecase(t)
			input := configInput("number", value)
			input.Key = attachmentSizeKey
			valid := value == "1048576" || value == "1572864" || value == "10485760"
			if valid {
				repo.EXPECT().CreateKey(gomock.Any(), gomock.Any()).Return(models.OnlineConfKey{}, nil)
				expectSnapshotRefresh(repo)
			}

			_, err := uc.CreateKey(context.Background(), uuid.NewV4(), input)

			if valid {
				require.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, models.ErrValidation)
			}
		})
	}
}

func TestKnownKeyKindAndType(t *testing.T) {
	for _, input := range []onlineconf.KeyInput{
		{Scope: "backend", Key: attachmentSizeKey, Kind: "config", Config: &models.OnlineConfConfig{ValueType: "string", Value: json.RawMessage(`"1048576"`)}},
		{Scope: "backend", Key: attachmentSizeKey, Kind: "flag", Flag: &onlineconf.FlagInput{}},
		{Scope: "backend", Key: attachmentUploadKey, Kind: "config", Config: &models.OnlineConfConfig{ValueType: "boolean", Value: json.RawMessage("true")}},
	} {
		t.Run(input.Key+"/"+input.Kind, func(t *testing.T) {
			_, _, uc := newUsecase(t)

			_, err := uc.CreateKey(context.Background(), uuid.NewV4(), input)

			assert.ErrorIs(t, err, models.ErrValidation)
		})
	}
}

func TestCreateFlagResolvesUsers(t *testing.T) {
	repo, auth, uc := newUsecase(t)
	actor, includedID, excludedID := uuid.NewV4(), uuid.NewV4(), uuid.NewV4()
	input := flagInput()
	input.Flag.IncludeUserIDs = []uuid.UUID{includedID}
	input.Flag.ExcludeUserIDs = []uuid.UUID{excludedID}
	auth.EXPECT().GetUser(gomock.Any(), includedID).Return(models.User{ID: includedID, Login: "included"}, nil)
	auth.EXPECT().GetUser(gomock.Any(), excludedID).Return(models.User{ID: excludedID, Login: "excluded"}, nil)
	repo.EXPECT().CreateKey(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error) {
			assert.Equal(t, actor, key.UpdatedBy.ID)
			assert.Nil(t, key.Config)
			assert.True(t, key.Flag.Enabled)
			assert.Equal(t, int32(25), key.Flag.RolloutPercent)
			assert.Equal(t, []models.OnlineConfUser{{ID: includedID, Login: "included"}}, key.Flag.IncludeUsers)
			assert.Equal(t, []models.OnlineConfUser{{ID: excludedID, Login: "excluded"}}, key.Flag.ExcludeUsers)
			return key, nil
		})
	expectSnapshotRefresh(repo)

	_, err := uc.CreateKey(context.Background(), actor, input)

	require.NoError(t, err)
}

func TestInvalidFlagInput(t *testing.T) {
	id := uuid.NewV4()
	tests := []struct {
		name   string
		change func(*onlineconf.FlagInput)
	}{
		{"negative percent", func(flag *onlineconf.FlagInput) { flag.RolloutPercent = -1 }},
		{"percent above 100", func(flag *onlineconf.FlagInput) { flag.RolloutPercent = 101 }},
		{"too many included", func(flag *onlineconf.FlagInput) { flag.IncludeUserIDs = make([]uuid.UUID, 101) }},
		{"too many excluded", func(flag *onlineconf.FlagInput) { flag.ExcludeUserIDs = make([]uuid.UUID, 101) }},
		{"empty included id", func(flag *onlineconf.FlagInput) { flag.IncludeUserIDs = []uuid.UUID{uuid.Nil} }},
		{"empty excluded id", func(flag *onlineconf.FlagInput) { flag.ExcludeUserIDs = []uuid.UUID{uuid.Nil} }},
		{"duplicate included", func(flag *onlineconf.FlagInput) { flag.IncludeUserIDs = []uuid.UUID{id, id} }},
		{"duplicate excluded", func(flag *onlineconf.FlagInput) { flag.ExcludeUserIDs = []uuid.UUID{id, id} }},
		{"overlapping lists", func(flag *onlineconf.FlagInput) {
			flag.IncludeUserIDs = []uuid.UUID{id}
			flag.ExcludeUserIDs = []uuid.UUID{id}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, uc := newUsecase(t)
			input := flagInput()
			tt.change(input.Flag)

			_, err := uc.CreateKey(context.Background(), uuid.NewV4(), input)

			assert.ErrorIs(t, err, models.ErrValidation)
		})
	}
}

func TestFlagBoundaries(t *testing.T) {
	for _, percent := range []int32{0, 100} {
		t.Run(fmt.Sprint(percent), func(t *testing.T) {
			repo, _, uc := newUsecase(t)
			input := flagInput()
			input.Key = attachmentUploadKey
			input.Flag.Enabled = false
			input.Flag.RolloutPercent = percent
			repo.EXPECT().CreateKey(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error) {
					assert.False(t, key.Flag.Enabled)
					assert.Equal(t, percent, key.Flag.RolloutPercent)
					assert.NotNil(t, key.Flag.IncludeUsers)
					assert.NotNil(t, key.Flag.ExcludeUsers)
					return key, nil
				})

			expectSnapshotRefresh(repo)
			_, err := uc.CreateKey(context.Background(), uuid.NewV4(), input)

			require.NoError(t, err)
		})
	}
}

func TestOverrideListBoundaries(t *testing.T) {
	repo, auth, uc := newUsecase(t)
	input := flagInput()
	for i := 0; i < 100; i++ {
		input.Flag.IncludeUserIDs = append(input.Flag.IncludeUserIDs, uuid.NewV4())
		input.Flag.ExcludeUserIDs = append(input.Flag.ExcludeUserIDs, uuid.NewV4())
	}
	auth.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(200).DoAndReturn(
		func(_ context.Context, id uuid.UUID) (models.User, error) {
			return models.User{ID: id, Login: "user"}, nil
		})
	repo.EXPECT().CreateKey(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error) {
			assert.Len(t, key.Flag.IncludeUsers, 100)
			assert.Len(t, key.Flag.ExcludeUsers, 100)
			return key, nil
		})
	expectSnapshotRefresh(repo)

	_, err := uc.CreateKey(context.Background(), uuid.NewV4(), input)

	require.NoError(t, err)
}

func TestAuthFailuresDoNotWriteKey(t *testing.T) {
	tests := []struct {
		name    string
		authErr error
		wantErr error
	}{
		{"missing user", models.ErrUserNotFound, models.ErrValidation},
		{"wrapped missing user", fmt.Errorf("lookup: %w", models.ErrUserNotFound), models.ErrValidation},
		{"auth unavailable", errDependency, errDependency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, auth, uc := newUsecase(t)
			input := flagInput()
			id := uuid.NewV4()
			input.Flag.ExcludeUserIDs = []uuid.UUID{id}
			auth.EXPECT().GetUser(gomock.Any(), id).Return(models.User{}, tt.authErr)

			_, err := uc.CreateKey(context.Background(), uuid.NewV4(), input)

			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != models.ErrValidation {
				assert.NotErrorIs(t, err, models.ErrValidation)
			}
		})
	}
}

func TestUnexpectedAuthUser(t *testing.T) {
	_, auth, uc := newUsecase(t)
	input := flagInput()
	id := uuid.NewV4()
	input.Flag.IncludeUserIDs = []uuid.UUID{id}
	auth.EXPECT().GetUser(gomock.Any(), id).Return(models.User{ID: uuid.NewV4()}, nil)

	_, err := uc.CreateKey(context.Background(), uuid.NewV4(), input)

	require.Error(t, err)
	assert.NotErrorIs(t, err, models.ErrValidation)
}

func TestUpdateKey(t *testing.T) {
	repo, _, uc := newUsecase(t)
	actor, id := uuid.NewV4(), uuid.NewV4()
	input := configInput("number", "2097152")
	input.Key = attachmentSizeKey
	repo.EXPECT().ReadKey(gomock.Any(), id).Return(models.OnlineConfKey{ID: id, Kind: "config", Version: 3}, nil)
	repo.EXPECT().UpdateKey(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error) {
			assert.Equal(t, id, key.ID)
			assert.Equal(t, int32(3), key.Version)
			assert.Equal(t, actor, key.UpdatedBy.ID)
			assert.Equal(t, input.Config, key.Config)
			key.Version++
			return key, nil
		})
	expectSnapshotRefresh(repo)

	got, err := uc.UpdateKey(context.Background(), actor, id, 3, input)

	require.NoError(t, err)
	assert.Equal(t, int32(4), got.Version)
}

func TestUpdateFlag(t *testing.T) {
	repo, auth, uc := newUsecase(t)
	actor, id, includedID := uuid.NewV4(), uuid.NewV4(), uuid.NewV4()
	input := flagInput()
	input.Flag.IncludeUserIDs = []uuid.UUID{includedID}
	gomock.InOrder(
		repo.EXPECT().ReadKey(gomock.Any(), id).Return(models.OnlineConfKey{ID: id, Kind: "flag", Version: 1}, nil),
		auth.EXPECT().GetUser(gomock.Any(), includedID).Return(models.User{ID: includedID, Login: "included"}, nil),
		repo.EXPECT().UpdateKey(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, key models.OnlineConfKey) (models.OnlineConfKey, error) {
				assert.Equal(t, []models.OnlineConfUser{{ID: includedID, Login: "included"}}, key.Flag.IncludeUsers)
				return key, nil
			}),
		repo.EXPECT().ReadSnapshot(gomock.Any()).Return(models.Snapshot{Revision: 1}, nil),
	)

	_, err := uc.UpdateKey(context.Background(), actor, id, 1, input)

	require.NoError(t, err)
}

func TestUpdateKeyFailures(t *testing.T) {
	for _, stage := range []string{"missing", "read error", "stale version", "change kind", "concurrent update"} {
		t.Run(stage, func(t *testing.T) {
			repo, _, uc := newUsecase(t)
			actor, id := uuid.NewV4(), uuid.NewV4()
			input := configInput("number", "42")
			current := models.OnlineConfKey{ID: id, Kind: "config", Version: 2}
			var wantErr error
			switch stage {
			case "missing":
				repo.EXPECT().ReadKey(gomock.Any(), id).Return(models.OnlineConfKey{}, models.ErrOnlineConfKeyNotFound)
				wantErr = models.ErrOnlineConfKeyNotFound
			case "read error":
				repo.EXPECT().ReadKey(gomock.Any(), id).Return(models.OnlineConfKey{}, errDependency)
				wantErr = errDependency
			case "stale version":
				current.Version = 3
				repo.EXPECT().ReadKey(gomock.Any(), id).Return(current, nil)
				wantErr = models.ErrOnlineConfVersionConflict
			case "change kind":
				current.Kind = "flag"
				repo.EXPECT().ReadKey(gomock.Any(), id).Return(current, nil)
				wantErr = models.ErrValidation
			case "concurrent update":
				repo.EXPECT().ReadKey(gomock.Any(), id).Return(current, nil)
				repo.EXPECT().UpdateKey(gomock.Any(), gomock.Any()).Return(models.OnlineConfKey{}, models.ErrOnlineConfVersionConflict)
				wantErr = models.ErrOnlineConfVersionConflict
			}

			_, err := uc.UpdateKey(context.Background(), actor, id, 2, input)

			assert.ErrorIs(t, err, wantErr)
		})
	}
}

func TestInvalidWriteIdentity(t *testing.T) {
	id, actor := uuid.NewV4(), uuid.NewV4()
	tests := []struct {
		name          string
		userID, keyID uuid.UUID
		version       int32
	}{
		{"missing actor", uuid.Nil, id, 1},
		{"missing key", actor, uuid.Nil, 1},
		{"missing version", actor, id, 0},
		{"negative version", actor, id, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, uc := newUsecase(t)

			_, err := uc.UpdateKey(context.Background(), tt.userID, tt.keyID, tt.version, configInput("number", "42"))
			assert.ErrorIs(t, err, models.ErrValidation)
			err = uc.DeleteKey(context.Background(), tt.userID, tt.keyID, tt.version)
			assert.ErrorIs(t, err, models.ErrValidation)
		})
	}
	_, _, uc := newUsecase(t)
	_, err := uc.CreateKey(context.Background(), uuid.Nil, configInput("number", "42"))
	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestDeleteKey(t *testing.T) {
	for _, repoErr := range []error{nil, models.ErrOnlineConfVersionConflict, models.ErrOnlineConfKeyNotFound, errDependency} {
		t.Run(fmt.Sprint(repoErr), func(t *testing.T) {
			repo, _, uc := newUsecase(t)
			actor, id := uuid.NewV4(), uuid.NewV4()
			repo.EXPECT().DeleteKey(gomock.Any(), id, int32(3), actor).Return(repoErr)
			if repoErr == nil {
				expectSnapshotRefresh(repo)
			}

			err := uc.DeleteKey(context.Background(), actor, id, 3)

			assert.ErrorIs(t, err, repoErr)
		})
	}
}

func TestCreateKeyRepoErrors(t *testing.T) {
	for _, repoErr := range []error{models.ErrOnlineConfKeyExists, errDependency} {
		t.Run(repoErr.Error(), func(t *testing.T) {
			repo, _, uc := newUsecase(t)
			repo.EXPECT().CreateKey(gomock.Any(), gomock.Any()).Return(models.OnlineConfKey{}, repoErr)

			_, err := uc.CreateKey(context.Background(), uuid.NewV4(), configInput("number", "42"))

			assert.ErrorIs(t, err, repoErr)
		})
	}
}

func TestListKeys(t *testing.T) {
	for _, scope := range []string{"", "frontend", "backend", "auth", "notes"} {
		t.Run(scope, func(t *testing.T) {
			repo, _, uc := newUsecase(t)
			expected := []models.OnlineConfKey{{ID: uuid.NewV4()}}
			repo.EXPECT().ReadKeys(gomock.Any(), "attachments", scope, "flag").Return(expected, nil)

			got, err := uc.ListKeys(context.Background(), "attachments", scope, "flag")

			require.NoError(t, err)
			assert.Equal(t, expected, got)
		})
	}
}

func TestInvalidListFilters(t *testing.T) {
	for _, filter := range [][2]string{{"other", "flag"}, {"backend", "other"}} {
		t.Run(filter[0]+"/"+filter[1], func(t *testing.T) {
			_, _, uc := newUsecase(t)

			_, err := uc.ListKeys(context.Background(), "", filter[0], filter[1])

			assert.ErrorIs(t, err, models.ErrValidation)
		})
	}
}

func TestReadKeyAndHistory(t *testing.T) {
	repo, _, uc := newUsecase(t)
	id := uuid.NewV4()
	expected := models.OnlineConfKey{ID: id}
	history := []models.OnlineConfHistory{{KeyID: id, Action: "delete"}}
	repo.EXPECT().ReadKey(gomock.Any(), id).Return(expected, nil)
	repo.EXPECT().ReadHistory(gomock.Any(), id).Return(history, nil)

	got, err := uc.GetKey(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
	gotHistory, err := uc.ListHistory(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, history, gotHistory)
}

func TestReadMissingIdentity(t *testing.T) {
	_, _, uc := newUsecase(t)

	_, err := uc.GetKey(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, models.ErrValidation)
	_, err = uc.ListHistory(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, models.ErrValidation)
}

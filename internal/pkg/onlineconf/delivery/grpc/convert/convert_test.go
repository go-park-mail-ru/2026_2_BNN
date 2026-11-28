package convert

import (
	"encoding/json"
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyRoundTrip(t *testing.T) {
	base := models.OnlineConfKey{
		ID:          uuid.NewV4(),
		Scope:       "backend",
		Key:         "test.value",
		Description: "Настройка",
		IsPublic:    true,
		Version:     3,
		UpdatedBy: models.OnlineConfUser{
			ID: uuid.NewV4(), Login: "liza",
		},
		UpdatedAt: time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC),
	}

	config := base
	config.Kind = "config"
	config.Config = &models.OnlineConfConfig{
		ValueType: "number",
		Value:     json.RawMessage("9007199254740993"),
	}

	flag := base
	flag.Kind = "flag"
	flag.Flag = &models.OnlineConfFlag{
		Enabled:        true,
		RolloutPercent: 25,
		IncludeUsers: []models.OnlineConfUser{
			{ID: uuid.NewV4(), Login: "included"},
		},
		ExcludeUsers: []models.OnlineConfUser{
			{ID: uuid.NewV4(), Login: "excluded"},
		},
	}

	for _, expected := range []models.OnlineConfKey{config, flag} {
		t.Run(expected.Kind, func(t *testing.T) {
			actual, err := KeyFromProto(KeyToProto(expected))

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestInputRoundTrip(t *testing.T) {
	inputs := []onlineconf.KeyInput{
		{
			Scope: "backend", Key: "test.value", Kind: "config",
			Description: "Конфигурация", IsPublic: true,
			Config: &models.OnlineConfConfig{
				ValueType: "object",
				Value:     json.RawMessage(`{"limit":9007199254740993}`),
			},
		},
		{
			Scope: "backend", Key: "test.flag", Kind: "flag",
			Flag: &onlineconf.FlagInput{
				Enabled:        false,
				RolloutPercent: 0,
				IncludeUserIDs: []uuid.UUID{uuid.NewV4()},
				ExcludeUserIDs: []uuid.UUID{uuid.NewV4()},
			},
		},
	}

	for _, expected := range inputs {
		t.Run(expected.Kind, func(t *testing.T) {
			encoded, err := InputToProto(expected)
			require.NoError(t, err)

			actual, err := InputFromProto(encoded)
			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestInvalidInput(t *testing.T) {
	_, err := InputFromProto(nil)
	assert.ErrorIs(t, err, models.ErrValidation)

	_, err = InputFromProto(&gen.KeyInput{
		Kind: "flag",
		Settings: &gen.KeyInput_Flag{
			Flag: &gen.FlagInput{
				IncludeUserIds: []string{"broken"},
			},
		},
	})
	assert.ErrorIs(t, err, models.ErrValidation)

	_, err = InputToProto(onlineconf.KeyInput{
		Kind: "config",
		Config: &models.OnlineConfConfig{
			ValueType: "boolean",
			Value:     json.RawMessage("true"),
		},
		Flag: &onlineconf.FlagInput{},
	})
	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestEmptySnapshot(t *testing.T) {
	actual, err := SnapshotFromProto(
		SnapshotToProto(models.Snapshot{Revision: 0}),
	)

	require.NoError(t, err)
	assert.Zero(t, actual.Revision)
	assert.NotNil(t, actual.Keys)
	assert.Empty(t, actual.Keys)

	_, err = SnapshotFromProto(nil)
	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestConfigConversionCopiesBytes(t *testing.T) {
	source := models.OnlineConfKey{
		ID:   uuid.NewV4(),
		Kind: "config",
		UpdatedBy: models.OnlineConfUser{
			ID: uuid.NewV4(),
		},
		Config: &models.OnlineConfConfig{
			ValueType: "number",
			Value:     json.RawMessage("123"),
		},
	}

	encoded := KeyToProto(source)
	encoded.GetConfig().ValueJson[0] = '9'
	assert.Equal(t, "123", string(source.Config.Value))

	actual, err := KeyFromProto(encoded)
	require.NoError(t, err)
	encoded.GetConfig().ValueJson[0] = '8'

	assert.Equal(t, "923", string(actual.Config.Value))
}

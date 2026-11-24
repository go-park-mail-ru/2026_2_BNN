package grpcconv

import (
	"testing"
	"time"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUUID(t *testing.T) {
	id := uuid.NewV4()

	got, err := ParseUUID(id.String())
	require.NoError(t, err)
	assert.Equal(t, id, got)

	_, err = ParseUUID("bad")
	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestParseOptionalUUID(t *testing.T) {
	got, err := ParseOptionalUUID(nil)
	require.NoError(t, err)
	assert.Nil(t, got)

	raw := uuid.NewV4().String()
	got, err = ParseOptionalUUID(&raw)
	require.NoError(t, err)
	assert.Equal(t, raw, got.String())

	bad := "bad"
	_, err = ParseOptionalUUID(&bad)
	assert.ErrorIs(t, err, models.ErrValidation)
}

func TestOptionalUUIDToString(t *testing.T) {
	assert.Nil(t, OptionalUUIDToString(nil))

	id := uuid.NewV4()
	assert.Equal(t, id.String(), *OptionalUUIDToString(&id))
}

func TestTime(t *testing.T) {
	assert.Nil(t, TimeToProto(time.Time{}))
	assert.True(t, TimeFromProto(nil).IsZero())

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, now, TimeFromProto(TimeToProto(now)))
}

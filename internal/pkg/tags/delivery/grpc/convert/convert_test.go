package convert

import (
	"testing"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/tags/delivery/grpc/gen"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagConvert(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	tag := models.Tag{ID: uuid.NewV4(), CreatedBy: uuid.NewV4(), Name: "учёба", Color: "#FF880080", CreatedAt: now, UpdatedAt: now}

	got, err := TagsFromProto([]*gen.Tag{TagToProto(tag)})
	require.NoError(t, err)
	assert.Equal(t, []models.Tag{tag}, got)

	_, err = TagsFromProto([]*gen.Tag{{Id: "bad"}})
	assert.ErrorIs(t, err, models.ErrValidation)

	_, err = TagFromProto(&gen.Tag{Id: tag.ID.String(), CreatedBy: "bad"})
	assert.ErrorIs(t, err, models.ErrValidation)
}

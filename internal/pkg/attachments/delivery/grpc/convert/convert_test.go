package convert

import (
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments/delivery/grpc/gen"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentConvert(t *testing.T) {
	attachment := models.Attachment{ID: uuid.NewV4(), BlockID: uuid.NewV4(), Path: "attachments/file.png"}

	got, err := AttachmentFromProto(AttachmentToProto(attachment))
	require.NoError(t, err)
	assert.Equal(t, attachment, got)

	_, err = AttachmentFromProto(&gen.Attachment{Id: "bad"})
	assert.ErrorIs(t, err, models.ErrValidation)

	_, err = AttachmentFromProto(&gen.Attachment{Id: attachment.ID.String(), BlockId: "bad"})
	assert.ErrorIs(t, err, models.ErrValidation)
}

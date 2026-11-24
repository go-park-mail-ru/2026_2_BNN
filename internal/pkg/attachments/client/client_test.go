package client

import (
	"context"
	"io"
	"strings"
	"testing"

	"bnn/internal/models"
	attachmentsgrpc "bnn/internal/pkg/attachments/delivery/grpc"
	"bnn/internal/pkg/attachments/delivery/grpc/gen"
	"bnn/internal/pkg/attachments/mocks"
	"bnn/internal/pkg/grpcutil/grpctest"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

func newClient(t *testing.T) (*Client, *mocks.MockUsecase) {
	uc := mocks.NewMockUsecase(gomock.NewController(t))
	conn := grpctest.Dial(t, func(server *grpc.Server) {
		gen.RegisterAttachmentsServer(server, attachmentsgrpc.NewHandler(uc))
	})
	return NewClient(conn), uc
}

func TestClient(t *testing.T) {
	ctx := context.Background()
	client, uc := newClient(t)

	userID := uuid.NewV4()
	attachment := models.Attachment{ID: uuid.NewV4(), BlockID: uuid.NewV4(), Path: "attachments/file.png"}
	data := []byte("png")

	uc.EXPECT().UploadAttachment(gomock.Any(), userID, attachment.BlockID, data, "image/png").Return(attachment, nil)
	uploaded, err := client.UploadAttachment(ctx, userID, attachment.BlockID, data, "image/png")
	require.NoError(t, err)
	assert.Equal(t, attachment, uploaded)

	uc.EXPECT().GetAttachment(gomock.Any(), userID, attachment.ID).
		Return(models.File{Body: io.NopCloser(strings.NewReader("png")), ContentType: "image/png", Size: 3}, nil)
	file, err := client.GetAttachment(ctx, userID, attachment.ID)
	require.NoError(t, err)
	body, err := io.ReadAll(file.Body)
	require.NoError(t, err)
	assert.Equal(t, data, body)
	assert.Equal(t, "image/png", file.ContentType)
	assert.Equal(t, int64(3), file.Size)

	uc.EXPECT().DeleteAttachment(gomock.Any(), userID, attachment.ID).Return(nil)
	assert.NoError(t, client.DeleteAttachment(ctx, userID, attachment.ID))
}

func TestClientErrors(t *testing.T) {
	ctx := context.Background()
	client, uc := newClient(t)
	userID := uuid.NewV4()
	id := uuid.NewV4()

	uc.EXPECT().UploadAttachment(gomock.Any(), userID, id, gomock.Any(), "").Return(models.Attachment{}, models.ErrUnsupportedFileType)
	_, err := client.UploadAttachment(ctx, userID, id, nil, "")
	assert.ErrorIs(t, err, models.ErrUnsupportedFileType)

	uc.EXPECT().GetAttachment(gomock.Any(), userID, id).Return(models.File{}, models.ErrAttachmentNotFound)
	_, err = client.GetAttachment(ctx, userID, id)
	assert.ErrorIs(t, err, models.ErrAttachmentNotFound)

	uc.EXPECT().DeleteAttachment(gomock.Any(), userID, id).Return(models.ErrForbidden)
	assert.ErrorIs(t, client.DeleteAttachment(ctx, userID, id), models.ErrForbidden)
}

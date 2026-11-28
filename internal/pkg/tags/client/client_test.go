package client

import (
	"context"
	"testing"
	"time"

	"bnn/internal/models"
	tagsgrpc "bnn/internal/pkg/tags/delivery/grpc"
	"bnn/internal/pkg/tags/delivery/grpc/gen"
	"bnn/internal/pkg/tags/mocks"
	"bnn/internal/pkg/utils/grpcutil/grpctest"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

func newClient(t *testing.T) (*Client, *mocks.MockUsecase) {
	uc := mocks.NewMockUsecase(gomock.NewController(t))
	conn := grpctest.Dial(t, func(server *grpc.Server) {
		gen.RegisterTagsServer(server, tagsgrpc.NewHandler(uc))
	})
	return NewClient(conn), uc
}

func TestClient(t *testing.T) {
	ctx := context.Background()
	client, uc := newClient(t)

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	userID := uuid.NewV4()
	tag := models.Tag{ID: uuid.NewV4(), CreatedBy: userID, Name: "учёба", Color: "#FF880080", CreatedAt: now, UpdatedAt: now}

	uc.EXPECT().ListTags(gomock.Any(), userID).Return([]models.Tag{tag}, nil)
	list, err := client.ListTags(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, []models.Tag{tag}, list)

	uc.EXPECT().CreateTag(gomock.Any(), userID, tag.Name, tag.Color).Return(tag, nil)
	created, err := client.CreateTag(ctx, userID, tag.Name, tag.Color)
	require.NoError(t, err)
	assert.Equal(t, tag, created)

	uc.EXPECT().UpdateTag(gomock.Any(), userID, tag.ID, tag.Name, tag.Color).Return(tag, nil)
	updated, err := client.UpdateTag(ctx, userID, tag.ID, tag.Name, tag.Color)
	require.NoError(t, err)
	assert.Equal(t, tag, updated)

	uc.EXPECT().DeleteTag(gomock.Any(), userID, tag.ID).Return(nil)
	assert.NoError(t, client.DeleteTag(ctx, userID, tag.ID))
}

func TestClientErrors(t *testing.T) {
	ctx := context.Background()
	client, uc := newClient(t)
	userID := uuid.NewV4()
	tagID := uuid.NewV4()

	uc.EXPECT().ListTags(gomock.Any(), userID).Return(nil, models.ErrValidation)
	_, err := client.ListTags(ctx, userID)
	assert.ErrorIs(t, err, models.ErrValidation)

	uc.EXPECT().CreateTag(gomock.Any(), userID, "", "").Return(models.Tag{}, models.ErrTagExists)
	_, err = client.CreateTag(ctx, userID, "", "")
	assert.ErrorIs(t, err, models.ErrTagExists)

	uc.EXPECT().UpdateTag(gomock.Any(), userID, tagID, "", "").Return(models.Tag{}, models.ErrTagNotFound)
	_, err = client.UpdateTag(ctx, userID, tagID, "", "")
	assert.ErrorIs(t, err, models.ErrTagNotFound)

	uc.EXPECT().DeleteTag(gomock.Any(), userID, tagID).Return(models.ErrTagNotFound)
	assert.ErrorIs(t, client.DeleteTag(ctx, userID, tagID), models.ErrTagNotFound)
}

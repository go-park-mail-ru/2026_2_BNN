package grpc

import (
	"context"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/tags/delivery/grpc/gen"
	"bnn/internal/pkg/tags/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandler(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewV4()
	tagID := uuid.NewV4()
	tag := models.Tag{ID: tagID, CreatedBy: userID, Name: "учёба", Color: "#FF880080"}

	tests := []struct {
		name    string
		prepare func(uc *mocks.MockUsecase)
		call    func(h *Handler) error
		code    codes.Code
	}{
		{
			name: "list ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().ListTags(gomock.Any(), userID).Return([]models.Tag{tag}, nil)
			},
			call: func(h *Handler) error {
				_, err := h.ListTags(ctx, &gen.ListTagsRequest{UserId: userID.String()})
				return err
			},
			code: codes.OK,
		},
		{
			name:    "list bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.ListTags(ctx, &gen.ListTagsRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "list internal",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().ListTags(gomock.Any(), userID).Return(nil, assert.AnError)
			},
			call: func(h *Handler) error {
				_, err := h.ListTags(ctx, &gen.ListTagsRequest{UserId: userID.String()})
				return err
			},
			code: codes.Internal,
		},
		{
			name: "create ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CreateTag(gomock.Any(), userID, "учёба", "#FF880080").Return(tag, nil)
			},
			call: func(h *Handler) error {
				_, err := h.CreateTag(ctx, &gen.CreateTagRequest{UserId: userID.String(), Name: "учёба", Color: "#FF880080"})
				return err
			},
			code: codes.OK,
		},
		{
			name: "create exists",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().CreateTag(gomock.Any(), userID, "учёба", "#FF880080").Return(models.Tag{}, models.ErrTagExists)
			},
			call: func(h *Handler) error {
				_, err := h.CreateTag(ctx, &gen.CreateTagRequest{UserId: userID.String(), Name: "учёба", Color: "#FF880080"})
				return err
			},
			code: codes.AlreadyExists,
		},
		{
			name: "update ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateTag(gomock.Any(), userID, tagID, "учёба", "#FF880080").Return(tag, nil)
			},
			call: func(h *Handler) error {
				_, err := h.UpdateTag(ctx, &gen.UpdateTagRequest{UserId: userID.String(), TagId: tagID.String(), Name: "учёба", Color: "#FF880080"})
				return err
			},
			code: codes.OK,
		},
		{
			name:    "update bad tag",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.UpdateTag(ctx, &gen.UpdateTagRequest{UserId: userID.String(), TagId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "update not found",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UpdateTag(gomock.Any(), userID, tagID, "", "").Return(models.Tag{}, models.ErrTagNotFound)
			},
			call: func(h *Handler) error {
				_, err := h.UpdateTag(ctx, &gen.UpdateTagRequest{UserId: userID.String(), TagId: tagID.String()})
				return err
			},
			code: codes.NotFound,
		},
		{
			name: "delete ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().DeleteTag(gomock.Any(), userID, tagID).Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.DeleteTag(ctx, &gen.DeleteTagRequest{UserId: userID.String(), TagId: tagID.String()})
				return err
			},
			code: codes.OK,
		},
		{
			name: "delete not found",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().DeleteTag(gomock.Any(), userID, tagID).Return(models.ErrTagNotFound)
			},
			call: func(h *Handler) error {
				_, err := h.DeleteTag(ctx, &gen.DeleteTagRequest{UserId: userID.String(), TagId: tagID.String()})
				return err
			},
			code: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			err := tt.call(NewHandler(uc))
			assert.Equal(t, tt.code, status.Code(err))
		})
	}
}

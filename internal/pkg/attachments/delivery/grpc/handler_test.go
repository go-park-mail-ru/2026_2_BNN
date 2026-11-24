package grpc

import (
	"context"
	"io"
	"strings"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments/delivery/grpc/gen"
	"bnn/internal/pkg/attachments/mocks"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandler(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewV4()
	blockID := uuid.NewV4()
	attachmentID := uuid.NewV4()
	data := []byte("png")

	tests := []struct {
		name    string
		prepare func(uc *mocks.MockUsecase)
		call    func(h *Handler) error
		code    codes.Code
	}{
		{
			name: "upload ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UploadAttachment(gomock.Any(), userID, blockID, data, "image/png").
					Return(models.Attachment{ID: attachmentID, BlockID: blockID}, nil)
			},
			call: func(h *Handler) error {
				_, err := h.UploadAttachment(ctx, &gen.UploadAttachmentRequest{UserId: userID.String(), BlockId: blockID.String(), Data: data, ContentType: "image/png"})
				return err
			},
			code: codes.OK,
		},
		{
			name:    "upload bad block",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.UploadAttachment(ctx, &gen.UploadAttachmentRequest{UserId: userID.String(), BlockId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "upload forbidden",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UploadAttachment(gomock.Any(), userID, blockID, data, "image/png").Return(models.Attachment{}, models.ErrForbidden)
			},
			call: func(h *Handler) error {
				_, err := h.UploadAttachment(ctx, &gen.UploadAttachmentRequest{UserId: userID.String(), BlockId: blockID.String(), Data: data, ContentType: "image/png"})
				return err
			},
			code: codes.PermissionDenied,
		},
		{
			name: "get ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetAttachment(gomock.Any(), userID, attachmentID).
					Return(models.File{Body: io.NopCloser(strings.NewReader("png")), ContentType: "image/png", Size: 3}, nil)
			},
			call: func(h *Handler) error {
				_, err := h.GetAttachment(ctx, &gen.AttachmentRequest{UserId: userID.String(), AttachmentId: attachmentID.String()})
				return err
			},
			code: codes.OK,
		},
		{
			name:    "get bad attachment",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.GetAttachment(ctx, &gen.AttachmentRequest{UserId: userID.String(), AttachmentId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "get not found",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetAttachment(gomock.Any(), userID, attachmentID).Return(models.File{}, models.ErrAttachmentNotFound)
			},
			call: func(h *Handler) error {
				_, err := h.GetAttachment(ctx, &gen.AttachmentRequest{UserId: userID.String(), AttachmentId: attachmentID.String()})
				return err
			},
			code: codes.NotFound,
		},
		{
			name: "delete ok",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().DeleteAttachment(gomock.Any(), userID, attachmentID).Return(nil)
			},
			call: func(h *Handler) error {
				_, err := h.DeleteAttachment(ctx, &gen.AttachmentRequest{UserId: userID.String(), AttachmentId: attachmentID.String()})
				return err
			},
			code: codes.OK,
		},
		{
			name:    "delete bad user",
			prepare: func(uc *mocks.MockUsecase) {},
			call: func(h *Handler) error {
				_, err := h.DeleteAttachment(ctx, &gen.AttachmentRequest{UserId: "bad"})
				return err
			},
			code: codes.InvalidArgument,
		},
		{
			name: "delete not found",
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().DeleteAttachment(gomock.Any(), userID, attachmentID).Return(models.ErrAttachmentNotFound)
			},
			call: func(h *Handler) error {
				_, err := h.DeleteAttachment(ctx, &gen.AttachmentRequest{UserId: userID.String(), AttachmentId: attachmentID.String()})
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

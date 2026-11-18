package grpc

import (
	"context"
	"io"

	"bnn/internal/pkg/attachments"
	"bnn/internal/pkg/attachments/delivery/grpc/gen"
	"bnn/internal/pkg/grpcconv"
	"bnn/internal/pkg/grpcerr"

	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	gen.UnimplementedAttachmentsServer
	uc attachments.Usecase
}

func NewHandler(uc attachments.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) UploadAttachment(ctx context.Context, in *gen.UploadAttachmentRequest) (*gen.Attachment, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	blockID, err := grpcconv.ParseUUID(in.GetBlockId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	attachment, err := h.uc.UploadAttachment(ctx, userID, blockID, in.GetData(), in.GetContentType())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return AttachmentToProto(attachment), nil
}

func (h *Handler) GetAttachment(ctx context.Context, in *gen.AttachmentRequest) (*gen.File, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	attachmentID, err := grpcconv.ParseUUID(in.GetAttachmentId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	file, err := h.uc.GetAttachment(ctx, userID, attachmentID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	defer file.Body.Close()

	data, err := io.ReadAll(file.Body)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &gen.File{
		Data:        data,
		ContentType: file.ContentType,
		Size:        file.Size,
	}, nil
}

func (h *Handler) DeleteAttachment(ctx context.Context, in *gen.AttachmentRequest) (*emptypb.Empty, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	attachmentID, err := grpcconv.ParseUUID(in.GetAttachmentId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	if err := h.uc.DeleteAttachment(ctx, userID, attachmentID); err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

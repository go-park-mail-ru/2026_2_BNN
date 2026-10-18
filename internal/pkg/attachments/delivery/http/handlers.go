package delivery

import (
	"context"
	"errors"

	"bnn/internal/pkg/attachments/delivery/http/api"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) UploadAttachment(ctx context.Context, req api.UploadAttachmentRequestObject) (api.UploadAttachmentResponseObject, error) {
	return nil, errors.New("not implemented")
}

func (h *Handler) GetAttachment(ctx context.Context, req api.GetAttachmentRequestObject) (api.GetAttachmentResponseObject, error) {
	return nil, errors.New("not implemented")
}

func (h *Handler) DeleteAttachment(ctx context.Context, req api.DeleteAttachmentRequestObject) (api.DeleteAttachmentResponseObject, error) {
	return nil, errors.New("not implemented")
}

package delivery

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments"
	"bnn/internal/pkg/attachments/delivery/http/api"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/utils/ctxutil"
	"bnn/internal/pkg/utils/upload"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
)

const (
	fileField = "file"

	defaultUploadEnabled = true

	messageUploadDisabled = "attachment uploads are disabled"
	messageFileTooLarge   = "file is too large"
	messageNoteForbidden  = "no access to note"
)

type Handler struct {
	uc   attachments.Usecase
	conf attachments.OnlineConf
}

func NewHandler(uc attachments.Usecase, conf attachments.OnlineConf) *Handler {
	return &Handler{uc: uc, conf: conf}
}

func (h *Handler) UploadAttachment(ctx context.Context, req api.UploadAttachmentRequestObject) (api.UploadAttachmentResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.UploadAttachment401Response{}, nil
	}

	if !h.conf.FlagEnabled(onlineconf.ScopeBackend, onlineconf.KeyAttachmentsUploadEnabled, userID, defaultUploadEnabled) {
		return api.UploadAttachment403JSONResponse{Code: onlineconf.CodeFeatureDisabled, Message: messageUploadDisabled}, nil
	}

	maxSize := h.conf.ConfigInt(
		onlineconf.ScopeBackend,
		onlineconf.KeyAttachmentsMaxFileSize,
		onlineconf.MaxAttachmentSizeBytes,
	)
	data, err := upload.ReadFile(req.Body, fileField, maxSize)
	if errors.Is(err, upload.ErrFileTooLarge) || errors.As(err, new(*http.MaxBytesError)) {
		return api.UploadAttachment413JSONResponse{Code: onlineconf.CodeFileTooLarge, Message: messageFileTooLarge, LimitBytes: maxSize}, nil
	}
	if err != nil {
		return api.UploadAttachment400Response{}, nil
	}

	contentType := http.DetectContentType(data)

	attachment, err := h.uc.UploadAttachment(ctx, userID, uuid.UUID(req.Id), data, contentType)
	if errors.Is(err, models.ErrForbidden) {
		return api.UploadAttachment403JSONResponse{Code: onlineconf.CodeForbidden, Message: messageNoteForbidden}, nil
	}
	if errors.Is(err, models.ErrBlockNotFound) {
		return api.UploadAttachment404Response{}, nil
	}
	if errors.Is(err, models.ErrUnsupportedFileType) {
		return api.UploadAttachment415Response{}, nil
	}
	if err != nil {
		return nil, err
	}
	resp, err := toAPIAttachment(attachment)
	if err != nil {
		return nil, err
	}

	return api.UploadAttachment201JSONResponse(resp), nil
}

func (h *Handler) GetAttachment(ctx context.Context, req api.GetAttachmentRequestObject) (api.GetAttachmentResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.GetAttachment401Response{}, nil
	}

	file, err := h.uc.GetAttachment(ctx, userID, uuid.UUID(req.Id))
	if errors.Is(err, models.ErrForbidden) {
		return api.GetAttachment403Response{}, nil
	}
	if errors.Is(err, models.ErrAttachmentNotFound) {
		return api.GetAttachment404Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.GetAttachment200AsteriskResponse{
		Body:          file.Body,
		ContentType:   file.ContentType,
		ContentLength: file.Size,
	}, nil
}

func (h *Handler) DeleteAttachment(ctx context.Context, req api.DeleteAttachmentRequestObject) (api.DeleteAttachmentResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.DeleteAttachment401Response{}, nil
	}

	err := h.uc.DeleteAttachment(ctx, userID, uuid.UUID(req.Id))
	if errors.Is(err, models.ErrForbidden) {
		return api.DeleteAttachment403Response{}, nil
	}
	if errors.Is(err, models.ErrAttachmentNotFound) {
		return api.DeleteAttachment404Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.DeleteAttachment204Response{}, nil
}

func toAPIAttachment(a models.Attachment) (api.Attachment, error) {
	link, err := url.JoinPath("/api/attachments", a.ID.String())
	if err != nil {
		return api.Attachment{}, err
	}

	return api.Attachment{
		Id:      openapi_types.UUID(a.ID),
		BlockId: openapi_types.UUID(a.BlockID),
		Url:     link,
	}, nil
}

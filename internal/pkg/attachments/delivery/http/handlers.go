package delivery

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments"
	"bnn/internal/pkg/attachments/delivery/http/api"
	"bnn/internal/pkg/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
)

const (
	fileField   = "file"
	maxFileSize = 10 << 20
)

var errFileTooLarge = errors.New("file too large")

type Handler struct {
	uc attachments.Usecase
}

func NewHandler(uc attachments.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) UploadAttachment(ctx context.Context, req api.UploadAttachmentRequestObject) (api.UploadAttachmentResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.UploadAttachment401Response{}, nil
	}

	data, err := readFile(req.Body)
	if errors.Is(err, errFileTooLarge) || errors.As(err, new(*http.MaxBytesError)) {
		return api.UploadAttachment413Response{}, nil
	}
	if err != nil {
		return api.UploadAttachment400Response{}, nil
	}

	contentType := http.DetectContentType(data)

	attachment, err := h.uc.UploadAttachment(ctx, userID, uuid.UUID(req.Id), data, contentType)
	if errors.Is(err, models.ErrBlockNotFound) {
		return api.UploadAttachment404Response{}, nil
	}
	if errors.Is(err, models.ErrUnsupportedFileType) {
		return api.UploadAttachment415Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.UploadAttachment201JSONResponse(toAPIAttachment(attachment)), nil
}

func (h *Handler) GetAttachment(ctx context.Context, req api.GetAttachmentRequestObject) (api.GetAttachmentResponseObject, error) {
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		return api.GetAttachment401Response{}, nil
	}

	file, err := h.uc.GetAttachment(ctx, userID, uuid.UUID(req.Id))
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
	if errors.Is(err, models.ErrAttachmentNotFound) {
		return api.DeleteAttachment404Response{}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.DeleteAttachment204Response{}, nil
}

func readFile(form *multipart.Reader) ([]byte, error) {
	for {
		part, err := form.NextPart()
		if err != nil {
			return nil, err
		}

		if part.FormName() != fileField {
			continue
		}

		data, err := io.ReadAll(io.LimitReader(part, maxFileSize+1))
		if err != nil {
			return nil, err
		}

		if len(data) > maxFileSize {
			return nil, errFileTooLarge
		}

		return data, nil
	}
}

func toAPIAttachment(a models.Attachment) api.Attachment {
	return api.Attachment{
		Id:      openapi_types.UUID(a.ID),
		BlockId: openapi_types.UUID(a.BlockID),
		Url:     "/api/attachments/" + a.ID.String(),
	}
}

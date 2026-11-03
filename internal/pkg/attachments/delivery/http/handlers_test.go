package delivery

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments/delivery/http/api"
	"bnn/internal/pkg/attachments/mocks"
	"bnn/internal/pkg/ctxutil"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var pngData = []byte("\x89PNG\r\n\x1a\nfile")

func newForm(t *testing.T, field string, data []byte) *multipart.Reader {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	part, err := w.CreateFormFile(field, "file")
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return multipart.NewReader(&buf, w.Boundary())
}

func TestUploadAttachment(t *testing.T) {
	userID := uuid.NewV4()
	blockID := uuid.NewV4()
	attachmentID := uuid.NewV4()

	tests := []struct {
		name     string
		ctx      context.Context
		form     func(t *testing.T) *multipart.Reader
		prepare  func(uc *mocks.MockUsecase)
		expected api.UploadAttachmentResponseObject
	}{
		{
			name: "ok",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			form: func(t *testing.T) *multipart.Reader { return newForm(t, fileField, pngData) },
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UploadAttachment(gomock.Any(), userID, blockID, pngData, "image/png").
					Return(models.Attachment{ID: attachmentID, BlockID: blockID}, nil)
			},
			expected: api.UploadAttachment201JSONResponse{
				Id:      openapi_types.UUID(attachmentID),
				BlockId: openapi_types.UUID(blockID),
				Url:     "/api/attachments/" + attachmentID.String(),
			},
		},
		{
			name:     "no user",
			ctx:      context.Background(),
			form:     func(t *testing.T) *multipart.Reader { return newForm(t, fileField, pngData) },
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.UploadAttachment401Response{},
		},
		{
			name:     "file too large",
			ctx:      ctxutil.WithUserID(context.Background(), userID),
			form:     func(t *testing.T) *multipart.Reader { return newForm(t, fileField, make([]byte, maxFileSize10MB+1)) },
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.UploadAttachment413Response{},
		},
		{
			name: "request body too large",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			form: func(t *testing.T) *multipart.Reader {
				var buf bytes.Buffer
				w := multipart.NewWriter(&buf)
				part, err := w.CreateFormFile(fileField, "file")
				require.NoError(t, err)
				_, err = part.Write(pngData)
				require.NoError(t, err)
				require.NoError(t, w.Close())

				body := http.MaxBytesReader(nil, io.NopCloser(&buf), 10)
				return multipart.NewReader(body, w.Boundary())
			},
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.UploadAttachment413Response{},
		},
		{
			name:     "no file field",
			ctx:      ctxutil.WithUserID(context.Background(), userID),
			form:     func(t *testing.T) *multipart.Reader { return newForm(t, "other", pngData) },
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.UploadAttachment400Response{},
		},
		{
			name: "block not found",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			form: func(t *testing.T) *multipart.Reader { return newForm(t, fileField, pngData) },
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UploadAttachment(gomock.Any(), userID, blockID, gomock.Any(), gomock.Any()).
					Return(models.Attachment{}, models.ErrBlockNotFound)
			},
			expected: api.UploadAttachment404Response{},
		},
		{
			name: "unsupported type",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			form: func(t *testing.T) *multipart.Reader { return newForm(t, fileField, []byte("text")) },
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().UploadAttachment(gomock.Any(), userID, blockID, []byte("text"), "text/plain; charset=utf-8").
					Return(models.Attachment{}, models.ErrUnsupportedFileType)
			},
			expected: api.UploadAttachment415Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			req := api.UploadAttachmentRequestObject{Id: openapi_types.UUID(blockID), Body: tt.form(t)}
			resp, err := NewHandler(uc).UploadAttachment(tt.ctx, req)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestGetAttachment(t *testing.T) {
	userID := uuid.NewV4()
	attachmentID := uuid.NewV4()
	body := io.NopCloser(strings.NewReader("file"))

	tests := []struct {
		name     string
		ctx      context.Context
		prepare  func(uc *mocks.MockUsecase)
		expected api.GetAttachmentResponseObject
	}{
		{
			name: "ok",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetAttachment(gomock.Any(), userID, attachmentID).
					Return(models.File{Body: body, ContentType: "image/png", Size: 4}, nil)
			},
			expected: api.GetAttachment200AsteriskResponse{Body: body, ContentType: "image/png", ContentLength: 4},
		},
		{
			name: "not found",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().GetAttachment(gomock.Any(), userID, attachmentID).Return(models.File{}, models.ErrAttachmentNotFound)
			},
			expected: api.GetAttachment404Response{},
		},
		{
			name:     "no user",
			ctx:      context.Background(),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.GetAttachment401Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			req := api.GetAttachmentRequestObject{Id: openapi_types.UUID(attachmentID)}
			resp, err := NewHandler(uc).GetAttachment(tt.ctx, req)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestDeleteAttachment(t *testing.T) {
	userID := uuid.NewV4()
	attachmentID := uuid.NewV4()

	tests := []struct {
		name     string
		ctx      context.Context
		prepare  func(uc *mocks.MockUsecase)
		expected api.DeleteAttachmentResponseObject
	}{
		{
			name: "ok",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().DeleteAttachment(gomock.Any(), userID, attachmentID).Return(nil)
			},
			expected: api.DeleteAttachment204Response{},
		},
		{
			name: "not found",
			ctx:  ctxutil.WithUserID(context.Background(), userID),
			prepare: func(uc *mocks.MockUsecase) {
				uc.EXPECT().DeleteAttachment(gomock.Any(), userID, attachmentID).Return(models.ErrAttachmentNotFound)
			},
			expected: api.DeleteAttachment404Response{},
		},
		{
			name:     "no user",
			ctx:      context.Background(),
			prepare:  func(uc *mocks.MockUsecase) {},
			expected: api.DeleteAttachment401Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockUsecase(ctrl)
			tt.prepare(uc)

			req := api.DeleteAttachmentRequestObject{Id: openapi_types.UUID(attachmentID)}
			resp, err := NewHandler(uc).DeleteAttachment(tt.ctx, req)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestHandlersInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := mocks.NewMockUsecase(ctrl)
	ctx := ctxutil.WithUserID(context.Background(), uuid.NewV4())
	dbErr := errors.New("db is down")

	uc.EXPECT().UploadAttachment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(models.Attachment{}, dbErr)
	uc.EXPECT().GetAttachment(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.File{}, dbErr)
	uc.EXPECT().DeleteAttachment(gomock.Any(), gomock.Any(), gomock.Any()).Return(dbErr)

	h := NewHandler(uc)

	_, err := h.UploadAttachment(ctx, api.UploadAttachmentRequestObject{Body: newForm(t, fileField, pngData)})
	assert.ErrorIs(t, err, dbErr)

	_, err = h.GetAttachment(ctx, api.GetAttachmentRequestObject{})
	assert.ErrorIs(t, err, dbErr)

	_, err = h.DeleteAttachment(ctx, api.DeleteAttachmentRequestObject{})
	assert.ErrorIs(t, err, dbErr)
}

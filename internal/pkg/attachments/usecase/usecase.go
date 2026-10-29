package usecase

import (
	"context"
	"fmt"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments"

	uuid "github.com/satori/go.uuid"
)

var allowedTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"image/gif":       ".gif",
	"application/pdf": ".pdf",
}

type Usecase struct {
	repo    attachments.Repo
	storage attachments.FileRepo
}

func NewUsecase(repo attachments.Repo, storage attachments.FileRepo) *Usecase {
	return &Usecase{
		repo:    repo,
		storage: storage,
	}
}

func (u *Usecase) UploadAttachment(ctx context.Context, userID, blockID uuid.UUID, data []byte, contentType string) (models.Attachment, error) {
	// Проверка до загрузки в S3, чтобы не грузить файл зря.
	// Окончательно права проверяет транзакция в CreateAttachment.
	canEdit, err := u.repo.CanEditBlock(ctx, userID, blockID)
	if err != nil {
		return models.Attachment{}, err
	}
	if !canEdit {
		return models.Attachment{}, models.ErrBlockNotFound
	}

	ext, ok := allowedTypes[contentType]
	if !ok {
		return models.Attachment{}, models.ErrUnsupportedFileType
	}

	id := uuid.NewV4()

	attachment := models.Attachment{
		ID:      id,
		BlockID: blockID,
		Path:    fmt.Sprintf("%s%s", id, ext),
	}

	if err := u.storage.PutFile(ctx, attachment.Path, data, contentType); err != nil {
		return models.Attachment{}, err
	}

	if err := u.repo.CreateAttachment(ctx, userID, attachment); err != nil {
		_ = u.storage.DeleteFile(ctx, attachment.Path)
		return models.Attachment{}, err
	}

	return attachment, nil
}

func (u *Usecase) GetAttachment(ctx context.Context, userID, attachmentID uuid.UUID) (models.File, error) {
	path, err := u.repo.ReadAttachmentPath(ctx, userID, attachmentID)
	if err != nil {
		return models.File{}, err
	}

	return u.storage.GetFile(ctx, path)
}

func (u *Usecase) DeleteAttachment(ctx context.Context, userID, attachmentID uuid.UUID) error {
	path, err := u.repo.DeleteAttachment(ctx, userID, attachmentID)
	if err != nil {
		return err
	}

	return u.storage.DeleteFile(ctx, path)
}

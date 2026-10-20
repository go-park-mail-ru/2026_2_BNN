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
	if err := u.checkBlockOwner(ctx, userID, blockID); err != nil {
		return models.Attachment{}, err
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

	if err := u.repo.CreateAttachment(ctx, attachment); err != nil {
		_ = u.storage.DeleteFile(ctx, attachment.Path)
		return models.Attachment{}, err
	}

	return attachment, nil
}

func (u *Usecase) GetAttachment(ctx context.Context, userID, attachmentID uuid.UUID) (models.File, error) {
	attachment, err := u.readOwnAttachment(ctx, userID, attachmentID)
	if err != nil {
		return models.File{}, err
	}

	return u.storage.GetFile(ctx, attachment.Path)
}

func (u *Usecase) DeleteAttachment(ctx context.Context, userID, attachmentID uuid.UUID) error {
	attachment, err := u.readOwnAttachment(ctx, userID, attachmentID)
	if err != nil {
		return err
	}

	if err := u.repo.DeleteAttachment(ctx, attachmentID); err != nil {
		return err
	}

	return u.storage.DeleteFile(ctx, attachment.Path)
}

func (u *Usecase) readOwnAttachment(ctx context.Context, userID, attachmentID uuid.UUID) (models.Attachment, error) {
	attachment, err := u.repo.ReadAttachment(ctx, attachmentID)
	if err != nil {
		return models.Attachment{}, err
	}

	if err := u.checkBlockOwner(ctx, userID, attachment.BlockID); err != nil {
		return models.Attachment{}, models.ErrAttachmentNotFound
	}

	return attachment, nil
}

func (u *Usecase) checkBlockOwner(ctx context.Context, userID, blockID uuid.UUID) error {
	ownerID, err := u.repo.ReadBlockOwner(ctx, blockID)
	if err != nil {
		return err
	}

	if ownerID != userID {
		return models.ErrBlockNotFound
	}

	return nil
}

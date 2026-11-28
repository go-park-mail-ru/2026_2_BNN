package usecase

import (
	"context"
	"errors"
	"fmt"

	"bnn/internal/models"
	"bnn/internal/pkg/attachments"

	uuid "github.com/satori/go.uuid"
)

type NotesClient interface {
	CheckAccess(ctx context.Context, userID, noteID uuid.UUID) (string, error)
}

var allowedTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"image/gif":       ".gif",
	"application/pdf": ".pdf",
}

type Usecase struct {
	repo        attachments.Repo
	storage     attachments.FileRepo
	notesClient NotesClient
}

func NewUsecase(repo attachments.Repo, storage attachments.FileRepo, notesClient NotesClient) *Usecase {
	return &Usecase{
		repo:        repo,
		storage:     storage,
		notesClient: notesClient,
	}
}

func (u *Usecase) UploadAttachment(ctx context.Context, userID, blockID uuid.UUID, data []byte, contentType string) (models.Attachment, error) {
	if err := u.checkEditAccess(ctx, userID, blockID); err != nil {
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
	attachment, err := u.readAttachmentForRead(ctx, userID, attachmentID)
	if err != nil {
		return models.File{}, err
	}

	return u.storage.GetFile(ctx, attachment.Path)
}

func (u *Usecase) DeleteAttachment(ctx context.Context, userID, attachmentID uuid.UUID) error {
	attachment, err := u.readAttachmentForEdit(ctx, userID, attachmentID)
	if err != nil {
		return err
	}

	if err := u.repo.DeleteAttachment(ctx, attachmentID); err != nil {
		return err
	}

	return u.storage.DeleteFile(ctx, attachment.Path)
}

func (u *Usecase) readAttachmentForRead(ctx context.Context, userID, attachmentID uuid.UUID) (models.Attachment, error) {
	attachment, err := u.repo.ReadAttachment(ctx, attachmentID)
	if err != nil {
		return models.Attachment{}, err
	}

	if err := u.checkReadAccess(ctx, userID, attachment.BlockID); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return models.Attachment{}, models.ErrForbidden
		}
		return models.Attachment{}, models.ErrAttachmentNotFound
	}

	return attachment, nil
}

func (u *Usecase) readAttachmentForEdit(ctx context.Context, userID, attachmentID uuid.UUID) (models.Attachment, error) {
	attachment, err := u.repo.ReadAttachment(ctx, attachmentID)
	if err != nil {
		return models.Attachment{}, err
	}

	if err := u.checkEditAccess(ctx, userID, attachment.BlockID); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return models.Attachment{}, models.ErrForbidden
		}
		return models.Attachment{}, models.ErrAttachmentNotFound
	}

	return attachment, nil
}

func (u *Usecase) checkReadAccess(ctx context.Context, userID, blockID uuid.UUID) error {
	noteID, err := u.repo.ReadBlockNoteID(ctx, blockID)
	if err != nil {
		return models.ErrBlockNotFound
	}

	return u.ensureReadAccess(ctx, userID, noteID)
}

func (u *Usecase) checkEditAccess(ctx context.Context, userID, blockID uuid.UUID) error {
	noteID, err := u.repo.ReadBlockNoteID(ctx, blockID)
	if err != nil {
		return models.ErrBlockNotFound
	}

	return u.ensureWriteAccess(ctx, userID, noteID)
}

func (u *Usecase) ensureWriteAccess(ctx context.Context, userID, noteID uuid.UUID) error {
	role, err := u.notesClient.CheckAccess(ctx, userID, noteID)
	if err != nil {
		return err
	}
	if role != "owner" && role != "editor" {
		return models.ErrForbidden
	}
	return nil
}

func (u *Usecase) ensureReadAccess(ctx context.Context, userID, noteID uuid.UUID) error {
	_, err := u.notesClient.CheckAccess(ctx, userID, noteID)
	return err
}

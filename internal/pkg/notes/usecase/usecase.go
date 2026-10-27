package usecase

import (
	"context"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/notes"

	uuid "github.com/satori/go.uuid"
)

type Usecase struct {
	repo notes.Repo
}

func NewUsecase(repo notes.Repo) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) GetNote(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error) {
	return u.repo.ReadNoteWithBlocks(ctx, userID, noteID)
}

func (u *Usecase) GetNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error) {
	return u.repo.ReadNotes(ctx, userID, limit, offset)
}

func (u *Usecase) CreateNote(ctx context.Context, userID uuid.UUID, req notes.CreateNoteRequest) (models.Note, error) {
	now := time.Now().UTC()
	note := models.Note{
		ID:           uuid.NewV4(),
		ParentNoteID: req.ParentNoteID,
		CreatedBy:    userID,
		Title:        req.Title,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.Header != nil {
		note.Header = *req.Header
	}
	if req.Icon != nil {
		note.Icon = *req.Icon
	}

	if err := u.repo.CreateNote(ctx, note); err != nil {
		return models.Note{}, err
	}
	return note, nil
}

func (u *Usecase) UpdateNote(ctx context.Context, userID, noteID uuid.UUID, req notes.UpdateNoteRequest) (models.Note, error) {
	return u.repo.UpdateNote(ctx, userID, noteID, req)
}

func (u *Usecase) DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error {
	return u.repo.DeleteNote(ctx, userID, noteID)
}

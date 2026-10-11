package usecase

import (
	"context"

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
	note, err := u.repo.ReadNote(ctx, noteID)
	if err != nil {
		return models.Note{}, err
	}

	if note.CreatedBy != userID {
		return models.Note{}, models.ErrNoteNotFound
	}

	blocks, err := u.repo.ReadBlocks(ctx, noteID)
	if err != nil {
		return models.Note{}, err
	}

	note.Blocks = blocks

	return note, nil
}

func (u *Usecase) GetNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error) {
	return u.repo.ReadNotes(ctx, userID, limit, offset)
}

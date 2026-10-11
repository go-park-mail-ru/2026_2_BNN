package notes

import (
	"context"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type Usecase interface {
	GetNote(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error)
	GetNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error)
}

type Repo interface {
	ReadNote(ctx context.Context, id uuid.UUID) (models.Note, error)
	ReadNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error)
	ReadBlocks(ctx context.Context, noteID uuid.UUID) ([]models.Block, error)
}

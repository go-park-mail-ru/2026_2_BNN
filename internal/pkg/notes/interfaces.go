package notes

import (
	"context"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

//go:generate go run go.uber.org/mock/mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type Usecase interface {
	GetNote(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error)
	GetNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error)
	CreateNote(ctx context.Context, userID uuid.UUID, title string, blocks []string) (models.Note, error)
	UpdateNote(ctx context.Context, userID, noteID uuid.UUID, title *string, blocks *[]string) (models.Note, error)
	DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error
}

type Repo interface {
	ReadNote(ctx context.Context, id uuid.UUID) (models.Note, error)
	ReadNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error)
	ReadBlocks(ctx context.Context, noteID uuid.UUID) ([]models.Block, error)

	CreateNote(ctx context.Context, note models.Note, blocks []models.Block) error
	UpdateNote(ctx context.Context, note models.Note, blocks []models.Block) error
	DeleteNote(ctx context.Context, noteID uuid.UUID) error
}

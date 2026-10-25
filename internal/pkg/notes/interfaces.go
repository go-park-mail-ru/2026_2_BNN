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
	CreateNote(ctx context.Context, userID uuid.UUID, req CreateNoteRequest) (models.Note, error)
	UpdateNote(ctx context.Context, userID, noteID uuid.UUID, req UpdateNoteRequest) (models.Note, error)
	DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error
}

type Repo interface {
	ReadNote(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error)
	ReadNoteWithBlocks(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error)
	ReadNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error)

	CreateNote(ctx context.Context, note models.Note) error
	UpdateNote(ctx context.Context, userID, noteID uuid.UUID, req UpdateNoteRequest) (models.Note, error)
	DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error
}

type CreateNoteRequest struct {
	Title        string
	ParentNoteID *uuid.UUID
	Header       *string
	Icon         *string
}

type UpdateNoteRequest struct {
	Title   string
	Header  *string
	Icon    *string
	Version int
	Blocks  []BlockInput
}

type BlockInput struct {
	ID      *uuid.UUID
	Content []byte
}

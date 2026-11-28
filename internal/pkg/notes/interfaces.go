package notes

import (
	"context"
	"time"

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
	AddFavorite(ctx context.Context, userID, noteID uuid.UUID) error
	RemoveFavorite(ctx context.Context, userID, noteID uuid.UUID) error
	GetFavorites(ctx context.Context, userID uuid.UUID) ([]models.Note, error)

	AddCollaborator(ctx context.Context, userID, noteID uuid.UUID, login string, role string) error
	RemoveCollaborator(ctx context.Context, userID, noteID, targetUserID uuid.UUID) error
	GetNoteUpdatedAt(ctx context.Context, noteID uuid.UUID) (time.Time, error)
	CheckAccess(ctx context.Context, userID, noteID uuid.UUID) (string, error)
	SetChecklistItem(ctx context.Context, userID, noteID, blockID uuid.UUID, checked bool) error
	GetNoteUpdates(ctx context.Context, noteID uuid.UUID, since *time.Time) (time.Time, []models.Block, error)
}

type Repo interface {
	ReadNote(ctx context.Context, id uuid.UUID) (models.Note, error)
	ReadNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error)
	ReadBlocks(ctx context.Context, noteID uuid.UUID) ([]models.Block, error)
	ReadNoteTags(ctx context.Context, noteID uuid.UUID) ([]models.Tag, error)
	ReadUserTags(ctx context.Context, userID uuid.UUID, tagIDs []uuid.UUID) ([]models.Tag, error)

	CreateNote(ctx context.Context, note models.Note, blocks []models.Block) error
	UpdateNote(ctx context.Context, note models.Note, blocks []models.Block, deletedBlockIDs []uuid.UUID) error
	DeleteNote(ctx context.Context, noteID uuid.UUID) error

	AddFavorite(ctx context.Context, userID, noteID uuid.UUID) error
	RemoveFavorite(ctx context.Context, userID, noteID uuid.UUID) error
	ReadFavorites(ctx context.Context, userID uuid.UUID) ([]models.Note, error)

	GetNoteUpdatedAt(ctx context.Context, noteID uuid.UUID) (time.Time, error)

	AddCollaborator(ctx context.Context, noteID, userID uuid.UUID, role string) error
	RemoveCollaborator(ctx context.Context, noteID, userID uuid.UUID) error
	ReadCollaboratorRole(ctx context.Context, noteID, userID uuid.UUID) (string, error)

	UpdateBlock(ctx context.Context, noteID uuid.UUID, block models.Block) error
}

type CreateNoteRequest struct {
	Title        string
	ParentNoteID *uuid.UUID
	Header       *string
	Icon         *string
}

type UpdateNoteRequest struct {
	Title  string
	Header *string
	Icon   *string
	Blocks []BlockInput
	TagIDs []uuid.UUID
}

type BlockInput struct {
	ID      *uuid.UUID
	Content []byte
}

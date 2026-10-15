package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"encoding/json"

	"bnn/internal/models"
	"bnn/internal/pkg/notes"

	uuid "github.com/satori/go.uuid"
)

const (
	minTitleLength = 1
	maxTitleLength = 200
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

func (u *Usecase) CreateNote(ctx context.Context, userID uuid.UUID, title string, blocks []string) (models.Note, error) {
	title = strings.TrimSpace(title)
	if len(title) < minTitleLength || len(title) > maxTitleLength {
		return models.Note{}, models.ErrValidation
	}

	now := time.Now().UTC()

	note := models.Note{
		ID:        uuid.NewV4(),
		CreatedBy: userID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}

	modelBlocks := make([]models.Block, 0, len(blocks))
	for _, content := range blocks {
		wrapped, _ := json.Marshal(map[string]string{"text": content})
		modelBlocks = append(modelBlocks, models.Block{
			ID:        uuid.NewV4(),
			Content:   wrapped,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	note.Blocks = modelBlocks

	if err := u.repo.CreateNote(ctx, note, modelBlocks); err != nil {
		return models.Note{}, err
	}

	return note, nil
}

func (u *Usecase) UpdateNote(ctx context.Context, userID, noteID uuid.UUID, title *string, blocks *[]string) (models.Note, error) {
	note, err := u.repo.ReadNote(ctx, noteID)
	if err != nil {
		return models.Note{}, err
	}

	if note.CreatedBy != userID {
		return models.Note{}, models.ErrNoteNotFound
	}

	if title != nil {
		trimmed := strings.TrimSpace(*title)
		if len(trimmed) < minTitleLength || len(trimmed) > maxTitleLength {
			return models.Note{}, models.ErrValidation
		}
		note.Title = trimmed
	}

	var modelBlocks []models.Block
	if blocks != nil {
		now := time.Now().UTC()
		modelBlocks = make([]models.Block, 0, len(*blocks))
		for _, content := range *blocks {
			wrapped, _ := json.Marshal(map[string]string{"text": content})
			modelBlocks = append(modelBlocks, models.Block{
				ID:        uuid.NewV4(),
				Content:   wrapped,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
		note.Blocks = modelBlocks
	}

	note.UpdatedAt = time.Now().UTC()

	if err := u.repo.UpdateNote(ctx, note, modelBlocks); err != nil {
		return models.Note{}, err
	}

	return note, nil
}

func (u *Usecase) DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error {
	note, err := u.repo.ReadNote(ctx, noteID)
	if err != nil {
		return err
	}

	if note.CreatedBy != userID {
		return models.ErrNoteNotFound
	}

	return u.repo.DeleteNote(ctx, noteID)
}

var _ = errors.New // временно, чтобы не было unused

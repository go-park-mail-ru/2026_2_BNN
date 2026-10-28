package usecase

import (
	"context"
	"errors"
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
	tags, err := u.repo.ReadNoteTags(ctx, noteID)
	if err != nil {
		return models.Note{}, err
	}
	note.Blocks = blocks
	note.Tags = tags
	return note, nil
}

func (u *Usecase) GetNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error) {
	return u.repo.ReadNotes(ctx, userID, limit, offset)
}

func (u *Usecase) CreateNote(ctx context.Context, userID uuid.UUID, req notes.CreateNoteRequest) (models.Note, error) {
	if req.ParentNoteID != nil {
		parentNote, err := u.repo.ReadNote(ctx, *req.ParentNoteID)
		if err != nil {
			if errors.Is(err, models.ErrNoteNotFound) {
				return models.Note{}, models.ErrNoteNotFound
			}
			return models.Note{}, err
		}
		if parentNote.CreatedBy != userID {
			return models.Note{}, models.ErrNoteNotFound
		}
	}

	now := time.Now().UTC()
	note := models.Note{
		ID:           uuid.NewV4(),
		ParentNoteID: req.ParentNoteID,
		CreatedBy:    userID,
		Title:        req.Title,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.Header != nil {
		note.Header = *req.Header
	}
	if req.Icon != nil {
		note.Icon = *req.Icon
	}

	if err := u.repo.CreateNote(ctx, note, nil); err != nil {
		return models.Note{}, err
	}
	return note, nil
}

func (u *Usecase) UpdateNote(ctx context.Context, userID, noteID uuid.UUID, req notes.UpdateNoteRequest) (models.Note, error) {
	note, err := u.repo.ReadNote(ctx, noteID)
	if err != nil {
		return models.Note{}, err
	}
	if note.CreatedBy != userID {
		return models.Note{}, models.ErrNoteNotFound
	}

	note.Title = req.Title
	note.UpdatedAt = time.Now().UTC()
	if req.Header != nil {
		note.Header = *req.Header
	}
	if req.Icon != nil {
		note.Icon = *req.Icon
	}

	oldBlocks, err := u.repo.ReadBlocks(ctx, noteID)
	if err != nil {
		return models.Note{}, err
	}

	oldIDs := make(map[uuid.UUID]bool, len(oldBlocks))
	for _, b := range oldBlocks {
		oldIDs[b.ID] = true
	}

	var blocksToSave []models.Block
	var finalBlockIDs []uuid.UUID

	for _, b := range req.Blocks {
		id := uuid.NewV4()
		if b.ID != nil {
			if !oldIDs[*b.ID] {
				return models.Note{}, models.ErrBlockNotFound
			}
			id = *b.ID
		}

		blocksToSave = append(blocksToSave, models.Block{
			ID:        id,
			Content:   b.Content,
			CreatedAt: note.UpdatedAt,
			UpdatedAt: note.UpdatedAt,
		})
		finalBlockIDs = append(finalBlockIDs, id)
	}

	var deletedIDs []uuid.UUID
	for _, oldB := range oldBlocks {
		found := false
		for _, newID := range finalBlockIDs {
			if oldB.ID == newID {
				found = true
				break
			}
		}
		if !found {
			deletedIDs = append(deletedIDs, oldB.ID)
		}
	}

	tags, err := u.noteTags(ctx, userID, noteID, req.TagIDs)
	if err != nil {
		return models.Note{}, err
	}

	note.Blocks = blocksToSave
	note.Tags = tags
	if err := u.repo.UpdateNote(ctx, note, blocksToSave, deletedIDs); err != nil {
		return models.Note{}, err
	}

	return note, nil
}

func (u *Usecase) noteTags(ctx context.Context, userID, noteID uuid.UUID, tagIDs []uuid.UUID) ([]models.Tag, error) {
	if tagIDs == nil {
		return u.repo.ReadNoteTags(ctx, noteID)
	}

	tags, err := u.repo.ReadUserTags(ctx, userID, tagIDs)
	if err != nil {
		return nil, err
	}
	if len(tags) != len(tagIDs) {
		return nil, models.ErrTagNotFound
	}

	return tags, nil
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

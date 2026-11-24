package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"bnn/internal/models"
	"bnn/internal/pkg/notes"

	uuid "github.com/satori/go.uuid"
)

const (
	maxTitleLength  = 255
	maxHeaderLength = 500
	maxIconLength   = 16

	zeroWidthJoiner   = '\u200d'
	variationSelector = '\ufe0f'
	keycapMark        = '\u20e3'
	tagFirst          = '\U000e0020'
	tagLast           = '\U000e007f'
)

const (
	RoleOwner  = "owner"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

type Usecase struct {
	repo       notes.Repo
	authClient AuthClient
}

func NewUsecase(repo notes.Repo, authClient AuthClient) *Usecase {
	return &Usecase{
		repo:       repo,
		authClient: authClient,
	}
}

func (u *Usecase) GetNote(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error) {
	if _, err := u.CheckAccess(ctx, userID, noteID); err != nil {
		return models.Note{}, err
	}
	note, err := u.repo.ReadNote(ctx, noteID)
	if err != nil {
		return models.Note{}, err
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
	if err := validateTitle(req.Title, req.Header, req.Icon); err != nil {
		return models.Note{}, err
	}
	if req.ParentNoteID != nil {
		role, err := u.CheckAccess(ctx, userID, *req.ParentNoteID)
		if err != nil {
			return models.Note{}, err
		}
		if role != RoleOwner && role != RoleEditor {
			return models.Note{}, models.ErrForbidden
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
	if err := validateTitle(req.Title, req.Header, req.Icon); err != nil {
		return models.Note{}, err
	}
	role, err := u.CheckAccess(ctx, userID, noteID)
	if err != nil {
		return models.Note{}, err
	}
	if role != RoleOwner && role != RoleEditor {
		return models.Note{}, models.ErrForbidden
	}

	note, err := u.repo.ReadNote(ctx, noteID)
	if err != nil {
		return models.Note{}, err
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

	blocksToSave := make([]models.Block, 0, len(req.Blocks))
	finalBlockIDs := make([]uuid.UUID, 0, len(req.Blocks))

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

	tags, err := u.noteTags(ctx, note.CreatedBy, noteID, req.TagIDs)
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
	role, err := u.CheckAccess(ctx, userID, noteID)
	if err != nil {
		return err
	}
	if role != RoleOwner {
		return models.ErrForbidden
	}

	return u.repo.DeleteNote(ctx, noteID)
}

func (u *Usecase) AddFavorite(ctx context.Context, userID, noteID uuid.UUID) error {
	if _, err := u.CheckAccess(ctx, userID, noteID); err != nil {
		return err
	}
	return u.repo.AddFavorite(ctx, userID, noteID)
}

func (u *Usecase) RemoveFavorite(ctx context.Context, userID, noteID uuid.UUID) error {
	if _, err := u.CheckAccess(ctx, userID, noteID); err != nil {
		return err
	}
	return u.repo.RemoveFavorite(ctx, userID, noteID)
}

func (u *Usecase) GetFavorites(ctx context.Context, userID uuid.UUID) ([]models.Note, error) {
	return u.repo.ReadFavorites(ctx, userID)
}

func validateTitle(title string, header, icon *string) error {
	if utf8.RuneCountInString(title) > maxTitleLength {
		return fmt.Errorf("%w: title too long", models.ErrValidation)
	}
	if header != nil {
		if err := validateHeader(*header); err != nil {
			return err
		}
	}
	if icon != nil {
		if err := validateIcon(*icon); err != nil {
			return err
		}
	}
	return nil
}

func validateHeader(header string) error {
	if utf8.RuneCountInString(header) > maxHeaderLength {
		return fmt.Errorf("%w: header too long", models.ErrValidation)
	}
	if strings.ContainsFunc(header, unicode.IsControl) {
		return fmt.Errorf("%w: header has control characters", models.ErrValidation)
	}
	return nil
}

func validateIcon(icon string) error {
	if utf8.RuneCountInString(icon) > maxIconLength {
		return fmt.Errorf("%w: icon too long", models.ErrValidation)
	}
	for _, r := range icon {
		if !isEmojiRune(r) {
			return fmt.Errorf("%w: icon must be emoji", models.ErrValidation)
		}
	}
	return nil
}

func isEmojiRune(r rune) bool {
	if r > unicode.MaxASCII && (unicode.Is(unicode.So, r) || unicode.Is(unicode.Sk, r)) {
		return true
	}
	return r == zeroWidthJoiner || r == variationSelector || r == keycapMark || (r >= tagFirst && r <= tagLast)
}

func (u *Usecase) GetNoteUpdatedAt(ctx context.Context, noteID uuid.UUID) (time.Time, error) {
	return u.repo.GetNoteUpdatedAt(ctx, noteID)
}

func (u *Usecase) CheckAccess(ctx context.Context, userID, noteID uuid.UUID) (string, error) {
	note, err := u.repo.ReadNote(ctx, noteID)
	if err != nil {
		return "", err
	}

	if note.CreatedBy == userID {
		return RoleOwner, nil
	}

	return u.repo.ReadCollaboratorRole(ctx, noteID, userID)
}

func (u *Usecase) AddCollaborator(ctx context.Context, userID, noteID uuid.UUID, login string, role string) error {
	if role != RoleEditor && role != RoleViewer {
		return models.ErrValidation
	}
	if login == "" {
		return models.ErrValidation
	}

	note, err := u.repo.ReadNote(ctx, noteID)
	if err != nil {
		return err
	}
	if note.CreatedBy != userID {
		return models.ErrForbidden
	}

	targetUserID, err := u.authClient.GetUserByLogin(ctx, login)
	if err != nil {
		return err
	}

	if userID == targetUserID {
		return models.ErrValidation
	}

	return u.repo.AddCollaborator(ctx, noteID, targetUserID, role)
}

func (u *Usecase) RemoveCollaborator(ctx context.Context, userID, noteID, targetUserID uuid.UUID) error {
	note, err := u.repo.ReadNote(ctx, noteID)
	if err != nil {
		return err
	}
	if note.CreatedBy != userID {
		return models.ErrForbidden
	}

	return u.repo.RemoveCollaborator(ctx, noteID, targetUserID)
}

type AuthClient interface {
	CheckUserExists(ctx context.Context, userID uuid.UUID) (bool, error)
	GetUserByLogin(ctx context.Context, login string) (uuid.UUID, error)
}

func (u *Usecase) GetNoteUpdates(ctx context.Context, noteID uuid.UUID, since *time.Time) (time.Time, []models.Block, error) {
	updatedAt, err := u.repo.GetNoteUpdatedAt(ctx, noteID)
	if err != nil {
		return time.Time{}, nil, err
	}

	if since != nil && !updatedAt.After(*since) {
		return updatedAt, nil, nil
	}

	blocks, err := u.repo.ReadBlocks(ctx, noteID)
	if err != nil {
		return time.Time{}, nil, err
	}

	return updatedAt, blocks, nil
}

package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"bnn/internal/models"
	"bnn/internal/pkg/tags"

	uuid "github.com/satori/go.uuid"
)

const (
	minNameLength = 1
	maxNameLength = 30
)

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{8}$`)

type Usecase struct {
	repo tags.Repo
}

func NewUsecase(repo tags.Repo) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) ListTags(ctx context.Context, userID uuid.UUID) ([]models.Tag, error) {
	return u.repo.ReadTags(ctx, userID)
}

func (u *Usecase) CreateTag(ctx context.Context, userID uuid.UUID, name, color string) (models.Tag, error) {
	name = strings.TrimSpace(name)
	if err := validateTag(name, color); err != nil {
		return models.Tag{}, err
	}

	now := time.Now().UTC()
	tag := models.Tag{
		ID:        uuid.NewV4(),
		CreatedBy: userID,
		Name:      name,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.repo.CreateTag(ctx, tag); err != nil {
		return models.Tag{}, err
	}

	return tag, nil
}

func (u *Usecase) UpdateTag(ctx context.Context, userID, tagID uuid.UUID, name, color string) (models.Tag, error) {
	name = strings.TrimSpace(name)
	if err := validateTag(name, color); err != nil {
		return models.Tag{}, err
	}

	tag, err := u.readOwnTag(ctx, userID, tagID)
	if err != nil {
		return models.Tag{}, err
	}

	tag.Name = name
	tag.Color = color
	tag.UpdatedAt = time.Now().UTC()

	if err := u.repo.UpdateTag(ctx, tag); err != nil {
		return models.Tag{}, err
	}

	return tag, nil
}

func (u *Usecase) DeleteTag(ctx context.Context, userID, tagID uuid.UUID) error {
	if _, err := u.readOwnTag(ctx, userID, tagID); err != nil {
		return err
	}

	return u.repo.DeleteTag(ctx, tagID)
}

func (u *Usecase) readOwnTag(ctx context.Context, userID, tagID uuid.UUID) (models.Tag, error) {
	tag, err := u.repo.ReadTag(ctx, tagID)
	if err != nil {
		return models.Tag{}, err
	}

	if tag.CreatedBy != userID {
		return models.Tag{}, models.ErrTagNotFound
	}

	return tag, nil
}

func validateTag(name, color string) error {
	nameLength := utf8.RuneCountInString(name)
	if nameLength < minNameLength || nameLength > maxNameLength {
		return fmt.Errorf("%w: invalid tag name length", models.ErrValidation)
	}

	if !colorPattern.MatchString(color) {
		return fmt.Errorf("%w: invalid tag color", models.ErrValidation)
	}

	return nil
}

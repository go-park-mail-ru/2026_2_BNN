package usecase

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"bnn/internal/models"
)

const searchUsersLimit = 10

func (u *Usecase) ListAdmins(ctx context.Context) ([]models.User, error) {
	return u.repo.ReadAdmins(ctx)
}

func (u *Usecase) SearchUsers(ctx context.Context, query string) ([]models.User, error) {
	query = strings.TrimSpace(query)
	queryLength := utf8.RuneCountInString(query)
	if queryLength == 0 || queryLength > maxLoginLength {
		return nil, fmt.Errorf("%w: invalid search query", models.ErrValidation)
	}

	return u.repo.SearchUsers(ctx, query, searchUsersLimit)
}

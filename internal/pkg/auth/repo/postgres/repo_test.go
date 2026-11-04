package postgres

import (
	"context"
	"testing"
	"time"

	"bnn/internal/models"

	"github.com/pashagolub/pgxmock/v4"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadUserColumnsOrder(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	repo := NewRepo(mock)
	now := time.Now().UTC()
	id := uuid.NewV4()

	columns := []string{"id", "login", "password_hash", "avatar", "email", "is_external", "totp_secret", "is_2fa_enabled", "created_at", "updated_at"}
	mock.ExpectQuery(readUserByLoginQuery).WithArgs("a@b.ru").
		WillReturnRows(pgxmock.NewRows(columns).AddRow(id.String(), "a@b.ru", []byte(nil), "", "a@b.ru", true, "SECRET", false, now, now))

	user, err := repo.ReadUserByLogin(context.Background(), "a@b.ru")

	require.NoError(t, err)
	assert.Equal(t, models.User{ID: id, Login: "a@b.ru", Email: "a@b.ru", IsExternal: true, TOTPSecret: "SECRET", CreatedAt: now, UpdatedAt: now}, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

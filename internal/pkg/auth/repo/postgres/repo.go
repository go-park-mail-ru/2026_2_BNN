package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"bnn/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	uuid "github.com/satori/go.uuid"
)

const uniqueViolationCode = "23505"

//go:embed queries/create_user.sql
var createUserQuery string

//go:embed queries/read_user_by_login.sql
var readUserByLoginQuery string

//go:embed queries/read_user_by_id.sql
var readUserByIDQuery string

//go:embed queries/create_refresh_token.sql
var createRefreshTokenQuery string

//go:embed queries/delete_refresh_token.sql
var deleteRefreshTokenQuery string

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateUser(ctx context.Context, user models.User) error {
	_, err := r.db.Exec(ctx, createUserQuery,
		user.ID,
		user.Login,
		user.PasswordHash,
		user.Avatar,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return models.ErrUserExists
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *Repo) ReadUserByLogin(ctx context.Context, login string) (models.User, error) {
	return r.readUser(ctx, readUserByLoginQuery, login)
}

func (r *Repo) ReadUserByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return r.readUser(ctx, readUserByIDQuery, id)
}

func (r *Repo) CreateRefreshToken(ctx context.Context, token models.RefreshToken) error {
	_, err := r.db.Exec(ctx, createRefreshTokenQuery,
		token.TokenHash,
		token.UserID,
		token.ExpiresAt,
		token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

func (r *Repo) DeleteRefreshToken(ctx context.Context, tokenHash []byte) (models.RefreshToken, error) {
	var token models.RefreshToken

	err := r.db.QueryRow(ctx, deleteRefreshTokenQuery, tokenHash).Scan(
		&token.TokenHash,
		&token.UserID,
		&token.ExpiresAt,
		&token.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.RefreshToken{}, models.ErrRefreshTokenNotFound
	}
	if err != nil {
		return models.RefreshToken{}, fmt.Errorf("delete refresh token: %w", err)
	}

	return token, nil
}

func (r *Repo) readUser(ctx context.Context, query string, arg any) (models.User, error) {
	var user models.User

	err := r.db.QueryRow(ctx, query, arg).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.Avatar,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, models.ErrUserNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("read user: %w", err)
	}

	return user, nil
}

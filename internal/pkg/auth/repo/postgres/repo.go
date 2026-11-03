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

//go:embed queries/update_password.sql
var updatePasswordQuery string

//go:embed queries/delete_user_refresh_tokens.sql
var deleteUserRefreshTokensQuery string

//go:embed queries/update_user_avatar.sql
var updateUserAvatarQuery string

//go:embed queries/update_totp_secret.sql
var updateTOTPSecretQuery string

//go:embed queries/update_2fa_status.sql
var update2FAStatusQuery string

//go:embed queries/read_user_by_email.sql
var readUserByEmailQuery string

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
		user.Email,
		user.IsExternal,
		user.TOTPSecret,
		user.Is2FAEnabled,
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

func (r *Repo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash []byte) error {
	tag, err := r.db.Exec(ctx, updatePasswordQuery, passwordHash, id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return models.ErrUserNotFound
	}

	return nil
}

func (r *Repo) DeleteUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, deleteUserRefreshTokensQuery, userID)
	if err != nil {
		return fmt.Errorf("delete user refresh tokens: %w", err)
	}
	return nil
}

func (r *Repo) readUser(ctx context.Context, query string, arg any) (models.User, error) {
	var user models.User

	err := r.db.QueryRow(ctx, query, arg).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.Avatar,
		&user.TOTPSecret,
		&user.Is2FAEnabled,
		&user.Email,
		&user.IsExternal,
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

func (r *Repo) UpdateAvatar(ctx context.Context, user models.User) error {
	_, err := r.db.Exec(ctx, updateUserAvatarQuery, user.Avatar, user.UpdatedAt, user.ID)
	if err != nil {
		return fmt.Errorf("update avatar: %w", err)
	}
	return nil
}

func (r *Repo) UpdateTOTPSecret(ctx context.Context, userID uuid.UUID, secret string) error {
	_, err := r.db.Exec(ctx, updateTOTPSecretQuery, secret, userID)
	if err != nil {
		return fmt.Errorf("update totp secret: %w", err)
	}
	return nil
}

func (r *Repo) Set2FAStatus(ctx context.Context, userID uuid.UUID, enabled bool) error {
	_, err := r.db.Exec(ctx, update2FAStatusQuery, enabled, userID)
	if err != nil {
		return fmt.Errorf("update 2fa status: %w", err)
	}
	return nil
}

func (r *Repo) ReadUserByEmail(ctx context.Context, email string) (models.User, error) {
	return r.readUser(ctx, readUserByEmailQuery, email)
}

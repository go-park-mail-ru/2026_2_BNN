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

const (
	uniqueViolationCode = "23505"
	loginUniqueName     = "users_login_key"
)

//go:embed queries/create_user.sql
var createUserQuery string

//go:embed queries/read_user_by_login.sql
var readUserByLoginQuery string

//go:embed queries/read_user_by_id.sql
var readUserByIDQuery string

//go:embed queries/update_password.sql
var updatePasswordQuery string

//go:embed queries/update_user_avatar.sql
var updateUserAvatarQuery string

//go:embed queries/create_session.sql
var createSessionQuery string

//go:embed queries/revoke_session.sql
var revokeSessionQuery string

//go:embed queries/revoke_user_sessions.sql
var revokeUserSessionsQuery string


var txOptions = pgx.TxOptions{IsoLevel: pgx.ReadCommitted}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateUser(ctx context.Context, user models.User, session models.RefreshToken) error {
	err := pgx.BeginTxFunc(ctx, r.db, txOptions, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, createUserQuery,
			user.ID,
			user.Login,
			user.PasswordHash,
			user.CreatedAt,
			user.UpdatedAt,
		)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, createSessionQuery, user.ID, session.TokenHash, session.ExpiresAt)
		return err
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode && pgErr.ConstraintName == loginUniqueName {
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

func (r *Repo) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatar string) error {
	tag, err := r.db.Exec(ctx, updateUserAvatarQuery, avatar, userID)
	if err != nil {
		return fmt.Errorf("update avatar: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return models.ErrUserNotFound
	}

	return nil
}

func (r *Repo) UpdatePassword(ctx context.Context, userID uuid.UUID, oldHash, newHash []byte, session models.RefreshToken) error {
	err := pgx.BeginTxFunc(ctx, r.db, txOptions, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, updatePasswordQuery, newHash, userID, oldHash)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return models.ErrWrongPassword
		}

		if _, err := tx.Exec(ctx, revokeUserSessionsQuery, userID); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, createSessionQuery, userID, session.TokenHash, session.ExpiresAt)
		return err
	})
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}

func (r *Repo) CreateSession(ctx context.Context, session models.RefreshToken) error {
	_, err := r.db.Exec(ctx, createSessionQuery, session.UserID, session.TokenHash, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (r *Repo) RotateSession(ctx context.Context, oldTokenHash []byte, session models.RefreshToken) (uuid.UUID, error) {
	var userID uuid.UUID

	err := pgx.BeginTxFunc(ctx, r.db, txOptions, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, revokeSessionQuery, oldTokenHash).Scan(&userID); err != nil {
			return err
		}

		_, err := tx.Exec(ctx, createSessionQuery, userID, session.TokenHash, session.ExpiresAt)
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, models.ErrRefreshTokenNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("rotate session: %w", err)
	}

	return userID, nil
}

func (r *Repo) RevokeSession(ctx context.Context, tokenHash []byte) error {
	if _, err := r.db.Exec(ctx, revokeSessionQuery, tokenHash); err != nil {
		return fmt.Errorf("revoke session: %w", err)
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
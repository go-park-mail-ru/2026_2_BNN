package models

import "errors"

var (
	ErrValidation           = errors.New("validation failed")
	ErrUserExists           = errors.New("user already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidCredentials   = errors.New("invalid login or password")
	ErrInvalidToken         = errors.New("invalid token")
	ErrNoteNotFound         = errors.New("note not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrUnsupportedMediaType = errors.New("unsupported media type")
)

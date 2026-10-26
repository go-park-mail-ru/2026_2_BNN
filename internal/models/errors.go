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
	ErrWrongPassword        = errors.New("wrong password")
	ErrBlockNotFound        = errors.New("block not found")
	ErrAttachmentNotFound   = errors.New("attachment not found")
	ErrUnsupportedFileType  = errors.New("unsupported file type")
	ErrTagNotFound          = errors.New("tag not found")
	ErrTagExists            = errors.New("tag already exists")
)

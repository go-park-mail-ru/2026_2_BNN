package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type SignUpRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type SignInRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type User struct {
	ID           uuid.UUID `json:"id"`
	Login        string    `json:"login"`
	PasswordHash []byte    `json:"-"`
	Avatar       string    `json:"avatar"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

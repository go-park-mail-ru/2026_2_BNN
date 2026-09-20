package models

import "time"

type SignUpRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type User struct {
	ID           string    `json:"ID"`
	Login        string    `json:"login"`
	PasswordHash []byte    `json:"-"`
	Avatar       string    `json:"avatar"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

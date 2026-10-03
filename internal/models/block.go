package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Block struct {
	ID        uuid.UUID `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

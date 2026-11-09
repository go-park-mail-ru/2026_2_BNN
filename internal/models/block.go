package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Block struct {
	ID        uuid.UUID
	Content   []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

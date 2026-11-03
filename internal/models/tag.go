package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Tag struct {
	ID        uuid.UUID
	CreatedBy uuid.UUID
	Name      string
	Color     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

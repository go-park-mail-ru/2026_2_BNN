package models

import (
	"encoding/json"
	"time"

	uuid "github.com/satori/go.uuid"
)

type Block struct {
	ID        uuid.UUID
	Content   json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

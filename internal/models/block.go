package models

import (
	"encoding/json"
	"time"

	uuid "github.com/satori/go.uuid"
)

type Block struct {
	ID        uuid.UUID       `json:"id"`
	Content   json.RawMessage `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Note struct {
	ID           uuid.UUID
	ParentNoteID *uuid.UUID
	Blocks       []Block
	Tags         []Tag
	CreatedBy    uuid.UUID
	Title        string
	Header       string
	Icon         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

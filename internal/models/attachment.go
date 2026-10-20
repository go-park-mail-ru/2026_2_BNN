package models

import (
	"io"

	uuid "github.com/satori/go.uuid"
)

type Attachment struct {
	ID      uuid.UUID
	BlockID uuid.UUID
	Path    string
}

type File struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

package grpcconv

import (
	"fmt"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
)

func ParseUUID(raw string) (uuid.UUID, error) {
	id, err := uuid.FromString(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid id", models.ErrValidation)
	}

	return id, nil
}

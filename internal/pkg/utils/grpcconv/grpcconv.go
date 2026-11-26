package grpcconv

import (
	"fmt"
	"time"

	"bnn/internal/models"

	uuid "github.com/satori/go.uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ParseUUID(raw string) (uuid.UUID, error) {
	id, err := uuid.FromString(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid id", models.ErrValidation)
	}

	return id, nil
}

func ParseOptionalUUID(raw *string) (*uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}

	id, err := ParseUUID(*raw)
	if err != nil {
		return nil, err
	}

	return &id, nil
}

func OptionalUUIDToString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}

	s := id.String()
	return &s
}

func TimeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}

	return timestamppb.New(t)
}

func TimeFromProto(t *timestamppb.Timestamp) time.Time {
	if t == nil {
		return time.Time{}
	}

	return t.AsTime()
}

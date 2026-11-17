package grpc

import (
	"bnn/internal/models"
	"bnn/internal/pkg/grpcconv"
	"bnn/internal/pkg/tags/delivery/grpc/gen"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TagToProto(t models.Tag) *gen.Tag {
	return &gen.Tag{
		Id:        t.ID.String(),
		CreatedBy: t.CreatedBy.String(),
		Name:      t.Name,
		Color:     t.Color,
		CreatedAt: timestamppb.New(t.CreatedAt),
		UpdatedAt: timestamppb.New(t.UpdatedAt),
	}
}

func TagFromProto(t *gen.Tag) (models.Tag, error) {
	id, err := grpcconv.ParseUUID(t.GetId())
	if err != nil {
		return models.Tag{}, err
	}

	createdBy, err := grpcconv.ParseUUID(t.GetCreatedBy())
	if err != nil {
		return models.Tag{}, err
	}

	return models.Tag{
		ID:        id,
		CreatedBy: createdBy,
		Name:      t.GetName(),
		Color:     t.GetColor(),
		CreatedAt: t.GetCreatedAt().AsTime(),
		UpdatedAt: t.GetUpdatedAt().AsTime(),
	}, nil
}

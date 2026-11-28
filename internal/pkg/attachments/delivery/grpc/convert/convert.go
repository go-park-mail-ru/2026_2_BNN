package convert

import (
	"bnn/internal/models"
	"bnn/internal/pkg/attachments/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"
)

func AttachmentToProto(a models.Attachment) *gen.Attachment {
	return &gen.Attachment{
		Id:      a.ID.String(),
		BlockId: a.BlockID.String(),
		Path:    a.Path,
	}
}

func AttachmentFromProto(a *gen.Attachment) (models.Attachment, error) {
	id, err := grpcconv.ParseUUID(a.GetId())
	if err != nil {
		return models.Attachment{}, err
	}

	blockID, err := grpcconv.ParseUUID(a.GetBlockId())
	if err != nil {
		return models.Attachment{}, err
	}

	return models.Attachment{
		ID:      id,
		BlockID: blockID,
		Path:    a.GetPath(),
	}, nil
}

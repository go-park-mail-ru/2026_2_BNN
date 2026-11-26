package convert

import (
	"bnn/internal/models"
	"bnn/internal/pkg/notes/delivery/grpc/gen"
	tagsconvert "bnn/internal/pkg/tags/delivery/grpc/convert"
	tagsgen "bnn/internal/pkg/tags/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"

	"github.com/samber/lo"
)

func NoteToProto(n models.Note) *gen.Note {
	return &gen.Note{
		Id:           n.ID.String(),
		ParentNoteId: grpcconv.OptionalUUIDToString(n.ParentNoteID),
		Blocks:       lo.Map(n.Blocks, func(b models.Block, _ int) *gen.Block { return BlockToProto(b) }),
		Tags:         lo.Map(n.Tags, func(t models.Tag, _ int) *tagsgen.Tag { return tagsconvert.TagToProto(t) }),
		CreatedBy:    n.CreatedBy.String(),
		Title:        n.Title,
		Header:       n.Header,
		Icon:         n.Icon,
		CreatedAt:    grpcconv.TimeToProto(n.CreatedAt),
		UpdatedAt:    grpcconv.TimeToProto(n.UpdatedAt),
	}
}

func NoteFromProto(n *gen.Note) (models.Note, error) {
	id, err := grpcconv.ParseUUID(n.GetId())
	if err != nil {
		return models.Note{}, err
	}

	createdBy, err := grpcconv.ParseUUID(n.GetCreatedBy())
	if err != nil {
		return models.Note{}, err
	}

	parentNoteID, err := grpcconv.ParseOptionalUUID(n.ParentNoteId)
	if err != nil {
		return models.Note{}, err
	}

	note := models.Note{
		ID:           id,
		ParentNoteID: parentNoteID,
		CreatedBy:    createdBy,
		Title:        n.GetTitle(),
		Header:       n.GetHeader(),
		Icon:         n.GetIcon(),
		CreatedAt:    grpcconv.TimeFromProto(n.GetCreatedAt()),
		UpdatedAt:    grpcconv.TimeFromProto(n.GetUpdatedAt()),
	}

	if len(n.GetBlocks()) > 0 {
		note.Blocks, err = BlocksFromProto(n.GetBlocks())
		if err != nil {
			return models.Note{}, err
		}
	}

	if len(n.GetTags()) > 0 {
		note.Tags, err = tagsconvert.TagsFromProto(n.GetTags())
		if err != nil {
			return models.Note{}, err
		}
	}

	return note, nil
}

func NotesFromProto(list []*gen.Note) ([]models.Note, error) {
	result := make([]models.Note, 0, len(list))
	for _, n := range list {
		note, err := NoteFromProto(n)
		if err != nil {
			return nil, err
		}
		result = append(result, note)
	}

	return result, nil
}

func BlockToProto(b models.Block) *gen.Block {
	return &gen.Block{
		Id:        b.ID.String(),
		Content:   b.Content,
		CreatedAt: grpcconv.TimeToProto(b.CreatedAt),
		UpdatedAt: grpcconv.TimeToProto(b.UpdatedAt),
	}
}

func BlocksFromProto(list []*gen.Block) ([]models.Block, error) {
	result := make([]models.Block, 0, len(list))
	for _, b := range list {
		id, err := grpcconv.ParseUUID(b.GetId())
		if err != nil {
			return nil, err
		}
		result = append(result, models.Block{
			ID:        id,
			Content:   b.GetContent(),
			CreatedAt: grpcconv.TimeFromProto(b.GetCreatedAt()),
			UpdatedAt: grpcconv.TimeFromProto(b.GetUpdatedAt()),
		})
	}

	return result, nil
}

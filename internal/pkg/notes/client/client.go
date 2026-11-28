package client

import (
	"context"
	"time"

	"bnn/internal/models"
	"bnn/internal/pkg/notes"
	"bnn/internal/pkg/notes/delivery/grpc/convert"
	"bnn/internal/pkg/notes/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"
	"bnn/internal/pkg/utils/grpcerr"

	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ notes.Usecase = (*Client)(nil)

type Client struct {
	client gen.NotesClient
}

func NewClient(conn grpc.ClientConnInterface) *Client {
	return &Client{client: gen.NewNotesClient(conn)}
}

func (c *Client) GetNote(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error) {
	resp, err := c.client.GetNote(ctx, &gen.NoteRequest{UserId: userID.String(), NoteId: noteID.String()})
	if err != nil {
		return models.Note{}, grpcerr.FromStatus(err)
	}
	return fullNoteFromProto(resp)
}

func (c *Client) GetNotes(ctx context.Context, userID uuid.UUID, limit, offset int64) ([]models.Note, error) {
	resp, err := c.client.GetNotes(ctx, &gen.GetNotesRequest{
		UserId: userID.String(),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}
	return convert.NotesFromProto(resp.GetNotes())
}

func (c *Client) CreateNote(ctx context.Context, userID uuid.UUID, req notes.CreateNoteRequest) (models.Note, error) {
	resp, err := c.client.CreateNote(ctx, &gen.CreateNoteRequest{
		UserId:       userID.String(),
		Title:        req.Title,
		ParentNoteId: grpcconv.OptionalUUIDToString(req.ParentNoteID),
		Header:       req.Header,
		Icon:         req.Icon,
	})
	if err != nil {
		return models.Note{}, grpcerr.FromStatus(err)
	}
	return convert.NoteFromProto(resp)
}

func (c *Client) UpdateNote(ctx context.Context, userID, noteID uuid.UUID, req notes.UpdateNoteRequest) (models.Note, error) {
	in := &gen.UpdateNoteRequest{
		UserId: userID.String(),
		NoteId: noteID.String(),
		Title:  req.Title,
		Header: req.Header,
		Icon:   req.Icon,
		Blocks: lo.Map(req.Blocks, func(b notes.BlockInput, _ int) *gen.BlockInput {
			return &gen.BlockInput{Id: grpcconv.OptionalUUIDToString(b.ID), Content: b.Content}
		}),
	}
	if req.TagIDs != nil {
		in.TagIds = &gen.TagIDs{
			Ids: lo.Map(req.TagIDs, func(id uuid.UUID, _ int) string { return id.String() }),
		}
	}

	resp, err := c.client.UpdateNote(ctx, in)
	if err != nil {
		return models.Note{}, grpcerr.FromStatus(err)
	}
	return fullNoteFromProto(resp)
}

func (c *Client) DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error {
	_, err := c.client.DeleteNote(ctx, &gen.NoteRequest{UserId: userID.String(), NoteId: noteID.String()})
	return grpcerr.FromStatus(err)
}

func (c *Client) AddFavorite(ctx context.Context, userID, noteID uuid.UUID) error {
	_, err := c.client.AddFavorite(ctx, &gen.NoteRequest{UserId: userID.String(), NoteId: noteID.String()})
	return grpcerr.FromStatus(err)
}

func (c *Client) RemoveFavorite(ctx context.Context, userID, noteID uuid.UUID) error {
	_, err := c.client.RemoveFavorite(ctx, &gen.NoteRequest{UserId: userID.String(), NoteId: noteID.String()})
	return grpcerr.FromStatus(err)
}

func (c *Client) GetFavorites(ctx context.Context, userID uuid.UUID) ([]models.Note, error) {
	resp, err := c.client.GetFavorites(ctx, &gen.UserRequest{UserId: userID.String()})
	if err != nil {
		return nil, grpcerr.FromStatus(err)
	}
	return convert.NotesFromProto(resp.GetNotes())
}

func (c *Client) GetNoteUpdatedAt(ctx context.Context, noteID uuid.UUID) (time.Time, error) {
	resp, err := c.client.GetNoteUpdatedAt(ctx, &gen.GetNoteUpdatedAtRequest{
		NoteId: noteID.String(),
	})
	if err != nil {
		return time.Time{}, grpcerr.FromStatus(err)
	}
	return resp.UpdatedAt.AsTime(), nil
}

func (c *Client) GetNoteUpdates(ctx context.Context, noteID uuid.UUID, since *time.Time) (time.Time, []models.Block, error) {
	req := &gen.GetNoteUpdatesRequest{
		NoteId: noteID.String(),
	}
	if since != nil {
		req.Since = timestamppb.New(*since)
	}

	resp, err := c.client.GetNoteUpdates(ctx, req)
	if err != nil {
		return time.Time{}, nil, grpcerr.FromStatus(err)
	}

	blocks, err := convert.BlocksFromProto(resp.GetBlocks())
	if err != nil {
		return time.Time{}, nil, err
	}

	var updatedAt time.Time
	if resp.GetUpdatedAt() != nil {
		updatedAt = resp.GetUpdatedAt().AsTime()
	}

	return updatedAt, blocks, nil
}

func (c *Client) CheckAccess(ctx context.Context, userID, noteID uuid.UUID) (string, error) {
	resp, err := c.client.CheckAccess(ctx, &gen.CheckAccessRequest{
		UserId: userID.String(),
		NoteId: noteID.String(),
	})
	if err != nil {
		return "", grpcerr.FromStatus(err)
	}
	return resp.Role, nil
}

func (c *Client) AddCollaborator(ctx context.Context, userID, noteID uuid.UUID, login string, role string) error {
	_, err := c.client.AddCollaborator(ctx, &gen.AddCollaboratorRequest{
		UserId: userID.String(),
		NoteId: noteID.String(),
		Login:  login,
		Role:   role,
	})
	return grpcerr.FromStatus(err)
}

func (c *Client) RemoveCollaborator(ctx context.Context, userID, noteID, targetUserID uuid.UUID) error {
	_, err := c.client.RemoveCollaborator(ctx, &gen.RemoveCollaboratorRequest{
		UserId:       userID.String(),
		NoteId:       noteID.String(),
		TargetUserId: targetUserID.String(),
	})
	return grpcerr.FromStatus(err)
}

func (c *Client) SetChecklistItem(ctx context.Context, userID, noteID, blockID uuid.UUID, checked bool) error {
	_, err := c.client.SetChecklistItem(ctx, &gen.SetChecklistItemRequest{
		UserId:  userID.String(),
		NoteId:  noteID.String(),
		BlockId: blockID.String(),
		Checked: checked,
	})
	return grpcerr.FromStatus(err)
}

func fullNoteFromProto(n *gen.Note) (models.Note, error) {
	note, err := convert.NoteFromProto(n)
	if err != nil {
		return models.Note{}, err
	}

	if note.Blocks == nil {
		note.Blocks = []models.Block{}
	}
	if note.Tags == nil {
		note.Tags = []models.Tag{}
	}

	return note, nil
}

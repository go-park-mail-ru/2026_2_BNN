package grpc

import (
	"context"

	"bnn/internal/models"
	"bnn/internal/pkg/grpcconv"
	"bnn/internal/pkg/grpcerr"
	"bnn/internal/pkg/tags"
	"bnn/internal/pkg/tags/delivery/grpc/gen"

	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	gen.UnimplementedTagsServer
	uc tags.Usecase
}

func NewHandler(uc tags.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ListTags(ctx context.Context, in *gen.ListTagsRequest) (*gen.ListTagsResponse, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	list, err := h.uc.ListTags(ctx, userID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &gen.ListTagsResponse{
		Tags: lo.Map(list, func(t models.Tag, _ int) *gen.Tag { return TagToProto(t) }),
	}, nil
}

func (h *Handler) CreateTag(ctx context.Context, in *gen.CreateTagRequest) (*gen.Tag, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	tag, err := h.uc.CreateTag(ctx, userID, in.GetName(), in.GetColor())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return TagToProto(tag), nil
}

func (h *Handler) UpdateTag(ctx context.Context, in *gen.UpdateTagRequest) (*gen.Tag, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	tagID, err := grpcconv.ParseUUID(in.GetTagId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	tag, err := h.uc.UpdateTag(ctx, userID, tagID, in.GetName(), in.GetColor())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return TagToProto(tag), nil
}

func (h *Handler) DeleteTag(ctx context.Context, in *gen.DeleteTagRequest) (*emptypb.Empty, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	tagID, err := grpcconv.ParseUUID(in.GetTagId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	if err := h.uc.DeleteTag(ctx, userID, tagID); err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

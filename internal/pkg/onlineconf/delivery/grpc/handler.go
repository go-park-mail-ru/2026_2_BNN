package grpc

import (
	"context"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/grpc/convert"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"
	"bnn/internal/pkg/utils/grpcerr"

	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	gen.UnimplementedOnlineConfServer
	uc onlineconf.Usecase
}

func NewHandler(uc onlineconf.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) GetSnapshot(
	ctx context.Context,
	in *gen.GetSnapshotRequest,
) (*gen.GetSnapshotResponse, error) {
	if in == nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}

	snapshot, changed, err := h.uc.GetSnapshot(ctx, in.Revision)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	response := &gen.GetSnapshotResponse{Changed: changed}
	if changed {
		response.Snapshot = convert.SnapshotToProto(snapshot)
	}

	return response, nil
}

func (h *Handler) ListKeys(
	ctx context.Context,
	in *gen.ListKeysRequest,
) (*gen.ListKeysResponse, error) {
	if in == nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}

	keys, err := h.uc.ListKeys(
		ctx, in.GetQuery(), in.GetScope(), in.GetKind(),
	)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	result := convert.KeysToProto(keys)

	for _, key := range result {
		if flag := key.GetFlag(); flag != nil {
			flag.IncludeUsers = nil
			flag.ExcludeUsers = nil
		}
	}

	return &gen.ListKeysResponse{Keys: result}, nil
}

func (h *Handler) GetKey(
	ctx context.Context,
	in *gen.GetKeyRequest,
) (*gen.Key, error) {
	id, err := grpcconv.ParseUUID(in.GetKeyId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	key, err := h.uc.GetKey(ctx, id)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return convert.KeyToProto(key), nil
}

func (h *Handler) CreateKey(
	ctx context.Context,
	in *gen.CreateKeyRequest,
) (*gen.Key, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	input, err := convert.InputFromProto(in.GetKey())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	key, err := h.uc.CreateKey(ctx, userID, input)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return convert.KeyToProto(key), nil
}

func (h *Handler) UpdateKey(
	ctx context.Context,
	in *gen.UpdateKeyRequest,
) (*gen.Key, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	id, err := grpcconv.ParseUUID(in.GetKeyId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	input, err := convert.InputFromProto(in.GetKey())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	key, err := h.uc.UpdateKey(
		ctx, userID, id, in.GetVersion(), input,
	)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return convert.KeyToProto(key), nil
}

func (h *Handler) DeleteKey(
	ctx context.Context,
	in *gen.DeleteKeyRequest,
) (*emptypb.Empty, error) {
	userID, err := grpcconv.ParseUUID(in.GetUserId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	id, err := grpcconv.ParseUUID(in.GetKeyId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	if err := h.uc.DeleteKey(
		ctx, userID, id, in.GetVersion(),
	); err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListHistory(
	ctx context.Context,
	in *gen.ListHistoryRequest,
) (*gen.ListHistoryResponse, error) {
	if in == nil {
		return nil, grpcerr.ToStatus(models.ErrValidation)
	}

	keyID, err := grpcconv.ParseUUID(in.GetKeyId())
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	history, err := h.uc.ListHistory(ctx, keyID)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}

	entries := make([]*gen.HistoryEntry, 0, len(history))
	for _, entry := range history {
		entries = append(entries, convert.HistoryToProto(entry))
	}

	return &gen.ListHistoryResponse{Entries: entries}, nil
}

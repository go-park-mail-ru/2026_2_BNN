package convert

import (
	"fmt"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"
)

func HistoryToProto(entry models.OnlineConfHistory) *gen.HistoryEntry {
	result := &gen.HistoryEntry{
		Id:     entry.ID.String(),
		KeyId:  entry.KeyID.String(),
		Action: entry.Action,
		User: &gen.UserRef{
			Id:    entry.User.ID.String(),
			Login: entry.User.Login,
		},
		CreatedAt: grpcconv.TimeToProto(entry.CreatedAt),
	}

	if entry.OldValue != nil {
		result.OldValue = KeyToProto(*entry.OldValue)
	}
	if entry.NewValue != nil {
		result.NewValue = KeyToProto(*entry.NewValue)
	}

	return result
}

func HistoryFromProto(
	entry *gen.HistoryEntry,
) (models.OnlineConfHistory, error) {
	if entry == nil {
		return models.OnlineConfHistory{}, fmt.Errorf(
			"%w: history entry is required", models.ErrValidation,
		)
	}

	id, err := grpcconv.ParseUUID(entry.GetId())
	if err != nil {
		return models.OnlineConfHistory{}, err
	}

	keyID, err := grpcconv.ParseUUID(entry.GetKeyId())
	if err != nil {
		return models.OnlineConfHistory{}, err
	}

	user, err := userFromProto(entry.GetUser())
	if err != nil {
		return models.OnlineConfHistory{}, err
	}

	if entry.CreatedAt != nil {
		if err := entry.CreatedAt.CheckValid(); err != nil {
			return models.OnlineConfHistory{}, fmt.Errorf(
				"%w: invalid created_at", models.ErrValidation,
			)
		}
	}

	result := models.OnlineConfHistory{
		ID:        id,
		KeyID:     keyID,
		Action:    entry.GetAction(),
		User:      user,
		CreatedAt: grpcconv.TimeFromProto(entry.GetCreatedAt()),
	}

	if entry.OldValue != nil {
		key, err := KeyFromProto(entry.OldValue)
		if err != nil {
			return models.OnlineConfHistory{}, err
		}
		result.OldValue = &key
	}

	if entry.NewValue != nil {
		key, err := KeyFromProto(entry.NewValue)
		if err != nil {
			return models.OnlineConfHistory{}, err
		}
		result.NewValue = &key
	}

	return result, nil
}

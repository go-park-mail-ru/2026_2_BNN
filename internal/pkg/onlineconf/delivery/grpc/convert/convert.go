package convert

import (
	"bytes"
	"encoding/json"
	"fmt"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/grpc/gen"
	"bnn/internal/pkg/utils/grpcconv"

	uuid "github.com/satori/go.uuid"
)

func KeyToProto(key models.OnlineConfKey) *gen.Key {
	result := &gen.Key{
		Id:          key.ID.String(),
		Scope:       key.Scope,
		Key:         key.Key,
		Kind:        key.Kind,
		Description: key.Description,
		IsPublic:    key.IsPublic,
		Version:     key.Version,
		UpdatedBy: &gen.UserRef{
			Id:    key.UpdatedBy.ID.String(),
			Login: key.UpdatedBy.Login,
		},
		UpdatedAt: grpcconv.TimeToProto(key.UpdatedAt),
	}

	if key.Config != nil {
		result.Settings = &gen.Key_Config{
			Config: configToProto(key.Config),
		}
	} else if key.Flag != nil {
		result.Settings = &gen.Key_Flag{
			Flag: &gen.Flag{
				Enabled:        key.Flag.Enabled,
				RolloutPercent: key.Flag.RolloutPercent,
				IncludeUsers:   usersToProto(key.Flag.IncludeUsers),
				ExcludeUsers:   usersToProto(key.Flag.ExcludeUsers),
			},
		}
	}

	return result
}

func KeyFromProto(key *gen.Key) (models.OnlineConfKey, error) {
	if key == nil {
		return models.OnlineConfKey{}, fmt.Errorf(
			"%w: key is required", models.ErrValidation,
		)
	}

	id, err := grpcconv.ParseUUID(key.GetId())
	if err != nil {
		return models.OnlineConfKey{}, err
	}

	updatedBy, err := userFromProto(key.GetUpdatedBy())
	if err != nil {
		return models.OnlineConfKey{}, err
	}

	if key.UpdatedAt != nil {
		if err := key.UpdatedAt.CheckValid(); err != nil {
			return models.OnlineConfKey{}, fmt.Errorf(
				"%w: invalid updated_at", models.ErrValidation,
			)
		}
	}

	result := models.OnlineConfKey{
		ID:          id,
		Scope:       key.GetScope(),
		Key:         key.GetKey(),
		Kind:        key.GetKind(),
		Description: key.GetDescription(),
		IsPublic:    key.GetIsPublic(),
		Version:     key.GetVersion(),
		UpdatedBy:   updatedBy,
		UpdatedAt:   grpcconv.TimeFromProto(key.GetUpdatedAt()),
	}

	switch result.Kind {
	case onlineconf.KindConfig:
		result.Config, err = configFromProto(key.GetConfig())
		if err != nil {
			return models.OnlineConfKey{}, err
		}

	case onlineconf.KindFlag:
		flag := key.GetFlag()
		if flag == nil {
			return models.OnlineConfKey{}, fmt.Errorf(
				"%w: flag settings are required", models.ErrValidation,
			)
		}

		included, err := usersFromProto(flag.GetIncludeUsers())
		if err != nil {
			return models.OnlineConfKey{}, err
		}

		excluded, err := usersFromProto(flag.GetExcludeUsers())
		if err != nil {
			return models.OnlineConfKey{}, err
		}

		result.Flag = &models.OnlineConfFlag{
			Enabled:        flag.GetEnabled(),
			RolloutPercent: flag.GetRolloutPercent(),
			IncludeUsers:   included,
			ExcludeUsers:   excluded,
		}

	default:
		return models.OnlineConfKey{}, fmt.Errorf(
			"%w: unknown kind", models.ErrValidation,
		)
	}

	return result, nil
}

func KeysToProto(keys []models.OnlineConfKey) []*gen.Key {
	result := make([]*gen.Key, len(keys))
	for i, key := range keys {
		result[i] = KeyToProto(key)
	}
	return result
}

func KeysFromProto(keys []*gen.Key) ([]models.OnlineConfKey, error) {
	result := make([]models.OnlineConfKey, 0, len(keys))

	for _, key := range keys {
		converted, err := KeyFromProto(key)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}

	return result, nil
}

func InputToProto(input onlineconf.KeyInput) (*gen.KeyInput, error) {
	result := &gen.KeyInput{
		Scope:       input.Scope,
		Key:         input.Key,
		Kind:        input.Kind,
		Description: input.Description,
		IsPublic:    input.IsPublic,
	}

	switch input.Kind {
	case onlineconf.KindConfig:
		if input.Config == nil || input.Flag != nil {
			return nil, fmt.Errorf(
				"%w: invalid config settings", models.ErrValidation,
			)
		}

		result.Settings = &gen.KeyInput_Config{
			Config: configToProto(input.Config),
		}

	case onlineconf.KindFlag:
		if input.Flag == nil || input.Config != nil {
			return nil, fmt.Errorf(
				"%w: invalid flag settings", models.ErrValidation,
			)
		}

		result.Settings = &gen.KeyInput_Flag{
			Flag: &gen.FlagInput{
				Enabled:        input.Flag.Enabled,
				RolloutPercent: input.Flag.RolloutPercent,
				IncludeUserIds: idsToProto(input.Flag.IncludeUserIDs),
				ExcludeUserIds: idsToProto(input.Flag.ExcludeUserIDs),
			},
		}

	default:
		return nil, fmt.Errorf(
			"%w: unknown kind", models.ErrValidation,
		)
	}

	return result, nil
}

func InputFromProto(input *gen.KeyInput) (onlineconf.KeyInput, error) {
	if input == nil {
		return onlineconf.KeyInput{}, fmt.Errorf(
			"%w: key input is required", models.ErrValidation,
		)
	}

	result := onlineconf.KeyInput{
		Scope:       input.GetScope(),
		Key:         input.GetKey(),
		Kind:        input.GetKind(),
		Description: input.GetDescription(),
		IsPublic:    input.GetIsPublic(),
	}

	switch result.Kind {
	case onlineconf.KindConfig:
		config, err := configFromProto(input.GetConfig())
		if err != nil {
			return onlineconf.KeyInput{}, err
		}
		result.Config = config

	case onlineconf.KindFlag:
		flag := input.GetFlag()
		if flag == nil {
			return onlineconf.KeyInput{}, fmt.Errorf(
				"%w: flag settings are required", models.ErrValidation,
			)
		}

		included, err := idsFromProto(flag.GetIncludeUserIds())
		if err != nil {
			return onlineconf.KeyInput{}, err
		}

		excluded, err := idsFromProto(flag.GetExcludeUserIds())
		if err != nil {
			return onlineconf.KeyInput{}, err
		}

		result.Flag = &onlineconf.FlagInput{
			Enabled:        flag.GetEnabled(),
			RolloutPercent: flag.GetRolloutPercent(),
			IncludeUserIDs: included,
			ExcludeUserIDs: excluded,
		}

	default:
		return onlineconf.KeyInput{}, fmt.Errorf(
			"%w: unknown kind", models.ErrValidation,
		)
	}

	return result, nil
}

func SnapshotToProto(snapshot models.Snapshot) *gen.Snapshot {
	return &gen.Snapshot{
		Revision: snapshot.Revision,
		Keys:     KeysToProto(snapshot.Keys),
	}
}

func SnapshotFromProto(snapshot *gen.Snapshot) (models.Snapshot, error) {
	if snapshot == nil || snapshot.GetRevision() < 0 {
		return models.Snapshot{}, fmt.Errorf(
			"%w: invalid snapshot", models.ErrValidation,
		)
	}

	keys, err := KeysFromProto(snapshot.GetKeys())
	if err != nil {
		return models.Snapshot{}, err
	}

	return models.Snapshot{
		Revision: snapshot.GetRevision(),
		Keys:     keys,
	}, nil
}

func configToProto(config *models.OnlineConfConfig) *gen.Config {
	return &gen.Config{
		ValueType: config.ValueType,
		ValueJson: bytes.Clone(config.Value),
	}
}

func configFromProto(config *gen.Config) (*models.OnlineConfConfig, error) {
	if config == nil || !json.Valid(config.GetValueJson()) {
		return nil, fmt.Errorf(
			"%w: config value must be valid JSON", models.ErrValidation,
		)
	}

	return &models.OnlineConfConfig{
		ValueType: config.GetValueType(),
		Value:     bytes.Clone(config.GetValueJson()),
	}, nil
}

func usersToProto(users []models.OnlineConfUser) []*gen.UserRef {
	result := make([]*gen.UserRef, len(users))
	for i, user := range users {
		result[i] = &gen.UserRef{
			Id:    user.ID.String(),
			Login: user.Login,
		}
	}
	return result
}

func userFromProto(user *gen.UserRef) (models.OnlineConfUser, error) {
	id, err := grpcconv.ParseUUID(user.GetId())
	if err != nil {
		return models.OnlineConfUser{}, err
	}

	return models.OnlineConfUser{
		ID:    id,
		Login: user.GetLogin(),
	}, nil
}

func usersFromProto(users []*gen.UserRef) ([]models.OnlineConfUser, error) {
	result := make([]models.OnlineConfUser, 0, len(users))

	for _, user := range users {
		converted, err := userFromProto(user)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}

	return result, nil
}

func idsToProto(ids []uuid.UUID) []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.String()
	}
	return result
}

func idsFromProto(ids []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(ids))

	for _, raw := range ids {
		id, err := grpcconv.ParseUUID(raw)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}

	return result, nil
}

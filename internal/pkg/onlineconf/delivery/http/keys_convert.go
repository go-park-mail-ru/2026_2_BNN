package delivery

import (
	"bytes"
	"fmt"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
	uuid "github.com/satori/go.uuid"
)

func toKeyInput(body api.OnlineConfKeyInput) (onlineconf.KeyInput, error) {
	input := onlineconf.KeyInput{
		Scope:       string(body.Scope),
		Key:         body.Key,
		Kind:        string(body.Kind),
		Description: body.Description,
		IsPublic:    body.IsPublic,
	}

	switch input.Kind {
	case onlineconf.KindConfig:
		if body.Config == nil || body.Flag != nil {
			return onlineconf.KeyInput{}, fmt.Errorf(
				"%w: config settings are required without flag settings",
				models.ErrValidation,
			)
		}

		input.Config = &models.OnlineConfConfig{
			ValueType: string(body.Config.ValueType),
			Value:     bytes.Clone(body.Config.Value),
		}

	case onlineconf.KindFlag:
		if body.Flag == nil || body.Config != nil {
			return onlineconf.KeyInput{}, fmt.Errorf(
				"%w: flag settings are required without config settings",
				models.ErrValidation,
			)
		}

		input.Flag = &onlineconf.FlagInput{
			Enabled:        body.Flag.Enabled,
			RolloutPercent: body.Flag.RolloutPercent,
			IncludeUserIDs: idsFromAPI(body.Flag.IncludeUserIDs),
			ExcludeUserIDs: idsFromAPI(body.Flag.ExcludeUserIDs),
		}

	default:
		return onlineconf.KeyInput{}, fmt.Errorf(
			"%w: unknown kind", models.ErrValidation,
		)
	}

	return input, nil
}

func idsFromAPI(ids []openapi_types.UUID) []uuid.UUID {
	result := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		result[i] = uuid.UUID(id)
	}
	return result
}

func toAPIKey(key models.OnlineConfKey, withOverrides bool) api.OnlineConfKey {
	result := api.OnlineConfKey{
		Id:          openapi_types.UUID(key.ID),
		Scope:       api.OnlineConfScope(key.Scope),
		Key:         key.Key,
		Kind:        api.OnlineConfKind(key.Kind),
		Description: key.Description,
		IsPublic:    key.IsPublic,
		Version:     key.Version,
		UpdatedBy: toAPIUser(models.User{
			ID: key.UpdatedBy.ID, Login: key.UpdatedBy.Login,
		}),
		UpdatedAt: key.UpdatedAt,
	}

	if key.Config != nil {
		result.Config = &api.OnlineConfConfig{
			ValueType: api.OnlineConfValueType(key.Config.ValueType),
			Value:     bytes.Clone(key.Config.Value),
		}
	}

	if key.Flag != nil {
		result.Flag = &api.OnlineConfFlag{
			Enabled:        key.Flag.Enabled,
			RolloutPercent: key.Flag.RolloutPercent,
		}

		if withOverrides {
			included := keyUsersToAPI(key.Flag.IncludeUsers)
			excluded := keyUsersToAPI(key.Flag.ExcludeUsers)

			result.Flag.IncludeUsers = &included
			result.Flag.ExcludeUsers = &excluded
		}
	}

	return result
}

func keyUsersToAPI(users []models.OnlineConfUser) []api.OnlineConfUser {
	result := make([]api.OnlineConfUser, 0, len(users))

	for _, user := range users {
		result = append(result, toAPIUser(models.User{
			ID: user.ID, Login: user.Login,
		}))
	}

	return result
}

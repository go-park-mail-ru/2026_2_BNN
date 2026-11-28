package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sync/atomic"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"

	uuid "github.com/satori/go.uuid"
)

const maxOverrideUsers = 100

var keyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.-]{0,99}$`)

var _ onlineconf.Usecase = (*Usecase)(nil)

type Usecase struct {
	repo        onlineconf.Repo
	authClient  onlineconf.AuthClient
	snapshot    atomic.Pointer[models.Snapshot]
	refreshGate chan struct{}
}

func NewUsecase(repo onlineconf.Repo, authClient onlineconf.AuthClient) *Usecase {
	return &Usecase{repo: repo, authClient: authClient, refreshGate: make(chan struct{}, 1)}
}

func (u *Usecase) ListScopes() []string {
	return []string{
		onlineconf.ScopeFrontend,
		onlineconf.ScopeBackend,
		onlineconf.ScopeAuth,
		onlineconf.ScopeNotes,
	}
}

func (u *Usecase) ListKeys(ctx context.Context, query, scope, kind string) ([]models.OnlineConfKey, error) {
	if scope != "" && !validScope(scope) {
		return nil, fmt.Errorf("%w: unknown scope", models.ErrValidation)
	}
	if kind != "" && kind != onlineconf.KindConfig && kind != onlineconf.KindFlag {
		return nil, fmt.Errorf("%w: unknown kind", models.ErrValidation)
	}
	return u.repo.ReadKeys(ctx, query, scope, kind)
}

func (u *Usecase) GetKey(ctx context.Context, id uuid.UUID) (models.OnlineConfKey, error) {
	if id == uuid.Nil {
		return models.OnlineConfKey{}, fmt.Errorf("%w: key id is required", models.ErrValidation)
	}
	return u.repo.ReadKey(ctx, id)
}

func (u *Usecase) CreateKey(ctx context.Context, userID uuid.UUID, input onlineconf.KeyInput) (models.OnlineConfKey, error) {
	if userID == uuid.Nil {
		return models.OnlineConfKey{}, fmt.Errorf("%w: user id is required", models.ErrValidation)
	}
	if err := validateInput(input); err != nil {
		return models.OnlineConfKey{}, err
	}
	key, err := u.buildKey(ctx, userID, input)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	key.ID = uuid.NewV4()
	saved, err := u.repo.CreateKey(ctx, key)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	u.refreshSnapshot(ctx)
	return saved, nil
}

func (u *Usecase) UpdateKey(ctx context.Context, userID, id uuid.UUID, version int32, input onlineconf.KeyInput) (models.OnlineConfKey, error) {
	if err := validateWrite(userID, id, version); err != nil {
		return models.OnlineConfKey{}, err
	}
	if err := validateInput(input); err != nil {
		return models.OnlineConfKey{}, err
	}
	current, err := u.repo.ReadKey(ctx, id)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	if current.Version != version {
		return models.OnlineConfKey{}, models.ErrOnlineConfVersionConflict
	}
	if current.Kind != input.Kind {
		return models.OnlineConfKey{}, fmt.Errorf("%w: kind cannot be changed", models.ErrValidation)
	}
	key, err := u.buildKey(ctx, userID, input)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	key.ID = id
	key.Version = version
	saved, err := u.repo.UpdateKey(ctx, key)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	u.refreshSnapshot(ctx)
	return saved, nil
}

func (u *Usecase) DeleteKey(ctx context.Context, userID, id uuid.UUID, version int32) error {
	if err := validateWrite(userID, id, version); err != nil {
		return err
	}
	if err := u.repo.DeleteKey(ctx, id, version, userID); err != nil {
		return err
	}
	u.refreshSnapshot(ctx)
	return nil
}

func (u *Usecase) ListHistory(ctx context.Context, keyID uuid.UUID) ([]models.OnlineConfHistory, error) {
	if keyID == uuid.Nil {
		return nil, fmt.Errorf("%w: key id is required", models.ErrValidation)
	}
	return u.repo.ReadHistory(ctx, keyID)
}

func (u *Usecase) buildKey(ctx context.Context, userID uuid.UUID, input onlineconf.KeyInput) (models.OnlineConfKey, error) {
	key := models.OnlineConfKey{
		Scope: input.Scope, Key: input.Key, Kind: input.Kind,
		Description: input.Description, IsPublic: input.IsPublic,
		UpdatedBy: models.OnlineConfUser{ID: userID},
	}
	if input.Config != nil {
		key.Config = &models.OnlineConfConfig{
			ValueType: input.Config.ValueType,
			Value:     bytes.Clone(input.Config.Value),
		}
		return key, nil
	}

	included, err := u.resolveUsers(ctx, input.Flag.IncludeUserIDs)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	excluded, err := u.resolveUsers(ctx, input.Flag.ExcludeUserIDs)
	if err != nil {
		return models.OnlineConfKey{}, err
	}
	key.Flag = &models.OnlineConfFlag{
		Enabled: input.Flag.Enabled, RolloutPercent: input.Flag.RolloutPercent,
		IncludeUsers: included, ExcludeUsers: excluded,
	}
	return key, nil
}

func (u *Usecase) resolveUsers(ctx context.Context, ids []uuid.UUID) ([]models.OnlineConfUser, error) {
	users := make([]models.OnlineConfUser, 0, len(ids))
	for _, id := range ids {
		user, err := u.authClient.GetUser(ctx, id)
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, fmt.Errorf("%w: override user %s does not exist", models.ErrValidation, id)
		}
		if err != nil {
			return nil, fmt.Errorf("read override user: %w", err)
		}
		if user.ID != id {
			return nil, fmt.Errorf("auth returned an unexpected user id")
		}
		users = append(users, models.OnlineConfUser{ID: user.ID, Login: user.Login})
	}
	return users, nil
}

func validateWrite(userID, id uuid.UUID, version int32) error {
	if userID == uuid.Nil || id == uuid.Nil {
		return fmt.Errorf("%w: user id and key id are required", models.ErrValidation)
	}
	if version < 1 {
		return fmt.Errorf("%w: version must be positive", models.ErrValidation)
	}
	return nil
}

func validateInput(input onlineconf.KeyInput) error {
	if !validScope(input.Scope) {
		return fmt.Errorf("%w: unknown scope", models.ErrValidation)
	}
	if !keyPattern.MatchString(input.Key) {
		return fmt.Errorf("%w: invalid key name", models.ErrValidation)
	}
	switch input.Kind {
	case onlineconf.KindConfig:
		if input.Config == nil || input.Flag != nil {
			return fmt.Errorf("%w: config settings are required without flag settings", models.ErrValidation)
		}
		if err := validateConfig(*input.Config); err != nil {
			return err
		}
	case onlineconf.KindFlag:
		if input.Flag == nil || input.Config != nil {
			return fmt.Errorf("%w: flag settings are required without config settings", models.ErrValidation)
		}
		if err := validateFlag(*input.Flag); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unknown kind", models.ErrValidation)
	}
	return validateKnownKey(input)
}

func validScope(scope string) bool {
	switch scope {
	case onlineconf.ScopeFrontend, onlineconf.ScopeBackend,
		onlineconf.ScopeAuth, onlineconf.ScopeNotes:
		return true
	default:
		return false
	}
}

func validateConfig(config models.OnlineConfConfig) error {
	if !json.Valid(config.Value) {
		return fmt.Errorf("%w: value must be valid JSON", models.ErrValidation)
	}
	decoder := json.NewDecoder(bytes.NewReader(config.Value))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("%w: cannot decode config value", models.ErrValidation)
	}
	var matches bool
	switch config.ValueType {
	case onlineconf.ValueString:
		_, matches = value.(string)
	case onlineconf.ValueNumber:
		_, matches = value.(json.Number)
	case onlineconf.ValueBoolean:
		_, matches = value.(bool)
	case onlineconf.ValueObject:
		_, matches = value.(map[string]any)
	case onlineconf.ValueArray:
		_, matches = value.([]any)
	default:
		return fmt.Errorf("%w: unknown value type", models.ErrValidation)
	}
	if !matches {
		return fmt.Errorf("%w: value does not match value type", models.ErrValidation)
	}
	return nil
}

func validateFlag(flag onlineconf.FlagInput) error {
	if flag.RolloutPercent < 0 || flag.RolloutPercent > 100 {
		return fmt.Errorf("%w: rollout percent must be between 0 and 100", models.ErrValidation)
	}
	if len(flag.IncludeUserIDs) > maxOverrideUsers || len(flag.ExcludeUserIDs) > maxOverrideUsers {
		return fmt.Errorf("%w: at most 100 users are allowed in each override list", models.ErrValidation)
	}
	seen := make(map[uuid.UUID]struct{}, len(flag.IncludeUserIDs)+len(flag.ExcludeUserIDs))
	for _, ids := range [][]uuid.UUID{flag.IncludeUserIDs, flag.ExcludeUserIDs} {
		for _, id := range ids {
			if id == uuid.Nil {
				return fmt.Errorf("%w: override user id is required", models.ErrValidation)
			}
			if _, exists := seen[id]; exists {
				return fmt.Errorf("%w: duplicate or overlapping override user", models.ErrValidation)
			}
			seen[id] = struct{}{}
		}
	}
	return nil
}

func validateKnownKey(input onlineconf.KeyInput) error {
	if input.Scope != onlineconf.ScopeBackend {
		return nil
	}

	switch input.Key {
	case onlineconf.KeyAttachmentsMaxFileSize:
		if input.Kind != onlineconf.KindConfig ||
			input.Config == nil ||
			input.Config.ValueType != onlineconf.ValueNumber {
			return fmt.Errorf(
				"%w: attachment size must be a number config",
				models.ErrValidation,
			)
		}

		var size int64
		if json.Unmarshal(input.Config.Value, &size) != nil ||
			size < onlineconf.MinAttachmentSizeBytes ||
			size > onlineconf.MaxAttachmentSizeBytes {
			return fmt.Errorf(
				"%w: attachment size must be an integer between %d and %d bytes",
				models.ErrValidation,
				onlineconf.MinAttachmentSizeBytes,
				onlineconf.MaxAttachmentSizeBytes,
			)
		}

	case onlineconf.KeyAttachmentsUploadEnabled:
		if input.Kind != onlineconf.KindFlag {
			return fmt.Errorf(
				"%w: attachment upload setting must be a flag",
				models.ErrValidation,
			)
		}
	}

	return nil
}

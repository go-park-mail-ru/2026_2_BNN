package onlineconf

import (
	"context"
	"log/slog"
	"sort"
	"sync/atomic"

	"bnn/internal/models"

	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
)

type AdminCache struct {
	source AdminSource
	admins atomic.Pointer[map[uuid.UUID]models.User]
}

func NewAdminCache(source AdminSource) *AdminCache {
	return &AdminCache{source: source}
}

func (c *AdminCache) Run(ctx context.Context) {
	runEvery(ctx, refreshInterval, c.Refresh)
}

func (c *AdminCache) Get(userID uuid.UUID) (models.User, bool) {
	admins := c.admins.Load()
	if admins == nil {
		return models.User{}, false
	}

	user, ok := (*admins)[userID]
	return user, ok
}

// List отдаёт админов по алфавиту логинов.
func (c *AdminCache) List() []models.User {
	admins := c.admins.Load()
	if admins == nil {
		return []models.User{}
	}

	users := lo.Values(*admins)
	sort.Slice(users, func(i, j int) bool { return users[i].Login < users[j].Login })
	return users
}

// Refresh перечитывает админов из auth; вызывается по таймеру и сразу после выдачи или снятия прав.
func (c *AdminCache) Refresh(ctx context.Context) {
	users, err := c.source.ListAdmins(ctx)
	if err != nil {
		slog.WarnContext(ctx, "Failed to refresh admins", "error", err)
		return
	}

	admins := lo.SliceToMap(users, func(u models.User) (uuid.UUID, models.User) { return u.ID, u })
	c.admins.Store(&admins)
}

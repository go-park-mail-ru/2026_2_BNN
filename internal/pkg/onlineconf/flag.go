package onlineconf

import (
	"hash/fnv"

	"bnn/internal/models"

	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
)

const (
	ReasonDisabled = "disabled"
	ReasonExcluded = "excluded"
	ReasonIncluded = "included"
	ReasonRollout  = "rollout"
)

// EvaluateFlag решает, включён ли флаг для пользователя, и говорит почему.
func EvaluateFlag(key models.OnlineConfKey, userID uuid.UUID) (bool, string) {
	flag := key.Flag
	if flag == nil || !flag.Enabled {
		return false, ReasonDisabled
	}

	if hasUser(flag.ExcludeUsers, userID) {
		return false, ReasonExcluded
	}

	if hasUser(flag.IncludeUsers, userID) {
		return true, ReasonIncluded
	}

	return bucket(key.Key, userID) < int(flag.RolloutPercent), ReasonRollout
}

func hasUser(users []models.OnlineConfUser, userID uuid.UUID) bool {
	return lo.ContainsBy(users, func(u models.OnlineConfUser) bool { return u.ID == userID })
}

// bucket — номер «корзины» пользователя от 0 до 99, всегда один и тот же для пары ключ + пользователь.
func bucket(key string, userID uuid.UUID) int {
	h := fnv.New32a()
	h.Write([]byte(key + ":" + userID.String()))
	return int(h.Sum32() % 100)
}

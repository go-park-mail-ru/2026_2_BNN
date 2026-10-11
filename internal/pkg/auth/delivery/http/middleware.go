package delivery

import (
	"context"
	"log/slog"
	"net/http"

	uuid "github.com/satori/go.uuid"
)

type contextKey string

const userIDKey contextKey = "userID"

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(accessCookieName)
		if err != nil {
			slog.Warn("Access cookie missing")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		userID, err := h.uc.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			writeError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

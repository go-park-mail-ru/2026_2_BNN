package delivery

import (
	"log/slog"
	"net/http"

	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/httpjson"
)

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
			httpjson.WriteError(w, err)
			return
		}

		ctx := ctxutil.WithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

package delivery

import (
	"net/http"

	"bnn/internal/pkg/ctxutil"
)

var publicPaths = map[string]bool{
	"/api/auth/signup":  true,
	"/api/auth/signin":  true,
	"/api/auth/refresh": true,
	"/api/auth/logout":  true,
}

func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(accessCookieName)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		userID, err := h.uc.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctxutil.WithUserID(r.Context(), userID)))
	})
}

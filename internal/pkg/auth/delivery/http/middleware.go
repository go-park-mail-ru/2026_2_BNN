package delivery

import (
	"crypto/subtle"
	"net/http"

	"bnn/internal/pkg/ctxutil"
)

const csrfHeaderName = "X-CSRF-Token"

var publicPaths = map[string]bool{
	"/api/auth/signup":  true,
	"/api/auth/signin":  true,
	"/api/auth/refresh": true,
	"/api/auth/logout":  true,

	"/api/auth/2fa/verify":      true,
	"/api/auth/github/login":    true,
	"/api/auth/github/callback": true,
}

var safeMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
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

		if !safeMethods[r.Method] && !validCSRF(r) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctxutil.WithUserID(r.Context(), userID)))
	})
}

func validCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	header := r.Header.Get(csrfHeaderName)
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) == 1
}

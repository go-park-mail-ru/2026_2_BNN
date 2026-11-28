package delivery

import (
	"net/http"

	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/utils/ctxutil"
	"bnn/internal/pkg/utils/httpjson"
)

const messageForbidden = "admin access required"

func AdminOnly(admins *onlineconf.AdminCache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := ctxutil.GetUserID(r.Context())
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if _, ok := admins.Get(userID); !ok {
				httpjson.WriteError(w, http.StatusForbidden, onlineconf.CodeForbidden, messageForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

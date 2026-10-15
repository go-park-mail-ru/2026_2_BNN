package delivery

import (
	"context"
	"errors"
	"net/http"

	"bnn/internal/pkg/api"
	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/ctxutil"
)

var ErrUnauthorized = errors.New("unauthorized")

var publicOperations = map[string]bool{
	"SignUp":  true,
	"SignIn":  true,
	"Refresh": true,
	"Logout":  true,
}

func StrictAuthMiddleware(uc auth.Usecase) api.StrictMiddlewareFunc {
	return func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error) {
			if publicOperations[operationID] {
				return next(ctx, w, r, req)
			}

			cookie, err := r.Cookie(accessCookieName)
			if err != nil {
				return nil, ErrUnauthorized
			}

			userID, err := uc.Authenticate(ctx, cookie.Value)
			if err != nil {
				return nil, ErrUnauthorized
			}

			ctx = ctxutil.WithUserID(ctx, userID)
			return next(ctx, w, r, req)
		}
	}
}

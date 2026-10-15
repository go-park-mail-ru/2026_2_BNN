package api

import (
	"net/http"
)

func CookieMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := withCookieStore(r.Context())
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)

		store := cookieStoreFromContext(r.Context())
		if store == nil {
			return
		}

		for _, c := range store.all() {
			http.SetCookie(w, &http.Cookie{
				Name:     c.Name,
				Value:    c.Value,
				Path:     c.Path,
				Domain:   c.Domain,
				Expires:  c.Expires,
				MaxAge:   c.MaxAge,
				Secure:   c.Secure,
				HttpOnly: c.HttpOnly,
				SameSite: c.SameSite,
			})
		}
	})
}

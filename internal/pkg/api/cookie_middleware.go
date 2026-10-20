package api;

// import (
// 	"net/http"
// )

// // cookieResponseWriter — обёртка над http.ResponseWriter, которая
// // перед первой записью заголовков добавляет Set-Cookie из store.
// type cookieResponseWriter struct {
// 	http.ResponseWriter
// 	store       *cookieStore
// 	wroteHeader bool
// }

// func (w *cookieResponseWriter) WriteHeader(status int) {
// 	if w.wroteHeader {
// 		return
// 	}
// 	w.wroteHeader = true

// 	if w.store != nil {
// 		for _, c := range w.store.all() {
// 			http.SetCookie(w.ResponseWriter, &http.Cookie{
// 				Name:     c.Name,
// 				Value:    c.Value,
// 				Path:     c.Path,
// 				Domain:   c.Domain,
// 				Expires:  c.Expires,
// 				MaxAge:   c.MaxAge,
// 				Secure:   c.Secure,
// 				HttpOnly: c.HttpOnly,
// 				SameSite: c.SameSite,
// 			})
// 		}
// 	}

// 	w.ResponseWriter.WriteHeader(status)
// }

// func (w *cookieResponseWriter) Write(b []byte) (int, error) {
// 	if !w.wroteHeader {
// 		w.WriteHeader(http.StatusOK)
// 	}
// 	return w.ResponseWriter.Write(b)
// }

// // CookieMiddleware кладёт в контекст хранилище кук и *http.Request,
// // а перед записью ответа добавляет все накопленные куки в Set-Cookie.
// func CookieMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		ctx, store := withCookieStore(r.Context())
// 		ctx = WithRequest(ctx, r)
// 		r = r.WithContext(ctx)

// 		wrapped := &cookieResponseWriter{
// 			ResponseWriter: w,
// 			store:          store,
// 		}

// 		next.ServeHTTP(wrapped, r)
// 	})
// }

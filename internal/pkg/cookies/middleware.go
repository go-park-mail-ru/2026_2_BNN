package cookies

import (
	"context"
	"net/http"
)

type writer struct {
	http.ResponseWriter
	store       *store
	wroteHeader bool
}

func (w *writer) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	for _, c := range w.store.cookies {
		http.SetCookie(w.ResponseWriter, c)
	}

	w.ResponseWriter.WriteHeader(status)
}

func (w *writer) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(b)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := &store{}

		ctx := context.WithValue(r.Context(), storeKey{}, s)
		ctx = context.WithValue(ctx, requestKey{}, r)

		next.ServeHTTP(&writer{ResponseWriter: w, store: s}, r.WithContext(ctx))
	})
}

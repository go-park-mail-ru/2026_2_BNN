package cookies

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
)

type writer struct {
	http.ResponseWriter
	store       *store
	wroteHeader bool
}

func (w *writer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("underlying ResponseWriter does not implement http.Hijacker")
	}

	if !w.wroteHeader {
		w.wroteHeader = true
		for _, c := range w.store.cookies {
			http.SetCookie(w.ResponseWriter, c)
		}
	}

	return hijacker.Hijack()
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

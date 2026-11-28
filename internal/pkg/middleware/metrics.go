package middleware

import (
	"net/http"
	"strconv"
	"time"

	"bnn/internal/pkg/metrics"

	"github.com/gorilla/mux"
)

const unknownPath = "unknown"

func Metrics(m metrics.HTTP) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			path := routePath(r)
			m.IncreaseHits(path, r.Method)
			m.ObserveDuration(path, r.Method, time.Since(start))

			if rw.status >= http.StatusBadRequest {
				m.IncreaseErr(strconv.Itoa(rw.status), path, r.Method)
			}
		})
	}
}

func routePath(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return unknownPath
	}

	template, err := route.GetPathTemplate()
	if err != nil {
		return unknownPath
	}

	return template
}

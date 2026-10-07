package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

const (
	corsAllowMethods = "GET, POST, OPTIONS"
	corsAllowHeaders = "Content-Type"
	corsMaxAge       = "600"
)

var ErrNoOrigins = errors.New("no valid CORS origins")

func parseOrigins(rawOrigins string) []string {
	origins := make([]string, 0, 2)

	for _, origin := range strings.Split(rawOrigins, ",") {
		origin = strings.TrimSpace(origin)
		origin = strings.TrimSuffix(origin, "/")

		if origin != "" {
			origins = append(origins, origin)
		}
	}

	return origins
}

func NewCORS(rawOrigins string) (func(http.Handler) http.Handler, error) {
	origins := parseOrigins(rawOrigins)
	if len(origins) == 0 {
		return nil, ErrNoOrigins
	}

	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}

	slog.Info("CORS configured", "allowed_origins", origins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			w.Header().Add("Vary", "Origin")

			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Add("Vary", "Access-Control-Request-Method")
				w.Header().Add("Vary", "Access-Control-Request-Headers")
				w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
				w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
				w.Header().Set("Access-Control-Max-Age", corsMaxAge)
				w.WriteHeader(http.StatusNoContent)

				return
			}

			next.ServeHTTP(w, r)
		})
	}, nil
}

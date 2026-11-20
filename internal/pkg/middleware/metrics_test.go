package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bnn/internal/pkg/metrics"
	"bnn/internal/pkg/middleware"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewHTTP(registry)

	router := mux.NewRouter()
	router.Use(middleware.Metrics(m))
	router.HandleFunc("/api/notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		if mux.Vars(r)["id"] == "missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)

	for _, id := range []string{"1", "2", "missing"} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/notes/"+id, nil))
	}

	expected := `
# HELP http_requests_total Number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/api/notes/{id}"} 3
# HELP http_requests_errors_total Number of HTTP requests with status 4xx and 5xx
# TYPE http_requests_errors_total counter
http_requests_errors_total{method="GET",path="/api/notes/{id}",status_code="404"} 1
`
	err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "http_requests_total", "http_requests_errors_total")
	assert.NoError(t, err)
	assert.Equal(t, 1, testutil.CollectAndCount(registry, "http_request_duration_seconds"))
}

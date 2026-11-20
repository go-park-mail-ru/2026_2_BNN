package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type HTTP interface {
	IncreaseHits(path, method string)
	IncreaseErr(statusCode, path, method string)
	ObserveDuration(path, method string, duration time.Duration)
}

type HTTPMetrics struct {
	hits     *prometheus.CounterVec
	errors   *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func NewHTTP(registerer prometheus.Registerer) *HTTPMetrics {
	m := &HTTPMetrics{
		hits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of HTTP requests",
		}, []string{"path", "method"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_errors_total",
			Help: "Number of HTTP requests with status 4xx and 5xx",
		}, []string{"status_code", "path", "method"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: prometheus.DefBuckets,
		}, []string{"path", "method"}),
	}

	registerer.MustRegister(m.hits, m.errors, m.duration)
	return m
}

func (m *HTTPMetrics) IncreaseHits(path, method string) {
	m.hits.WithLabelValues(path, method).Inc()
}

func (m *HTTPMetrics) IncreaseErr(statusCode, path, method string) {
	m.errors.WithLabelValues(statusCode, path, method).Inc()
}

func (m *HTTPMetrics) ObserveDuration(path, method string, duration time.Duration) {
	m.duration.WithLabelValues(path, method).Observe(duration.Seconds())
}

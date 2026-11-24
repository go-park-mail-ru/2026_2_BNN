package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type GRPC interface {
	IncreaseHits(method string)
	IncreaseErr(method, code string)
	ObserveDuration(method string, duration time.Duration)
}

type GRPCMetrics struct {
	hits     *prometheus.CounterVec
	errors   *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func NewGRPC(registerer prometheus.Registerer) *GRPCMetrics {
	m := &GRPCMetrics{
		hits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Number of gRPC requests",
		}, []string{"method"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_requests_errors_total",
			Help: "Number of gRPC requests with non-OK status codes",
		}, []string{"method", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "gRPC request duration",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
	}

	registerer.MustRegister(m.hits, m.errors, m.duration)
	return m
}

func (m *GRPCMetrics) IncreaseHits(method string) {
	m.hits.WithLabelValues(method).Inc()
}

func (m *GRPCMetrics) IncreaseErr(method, code string) {
	m.errors.WithLabelValues(method, code).Inc()
}

func (m *GRPCMetrics) ObserveDuration(method string, duration time.Duration) {
	m.duration.WithLabelValues(method).Observe(duration.Seconds())
}

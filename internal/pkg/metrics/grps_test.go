package metrics_test

import (
	"testing"
	"time"

	"bnn/internal/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGRPCMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewGRPC(reg)

	method := "/notes.Notes/GetNote"
	m.IncreaseHits(method)
	m.IncreaseHits(method)
	m.IncreaseErr(method, "NotFound")
	m.ObserveDuration(method, 25*time.Millisecond)

	families, err := reg.Gather()
	assert.NoError(t, err)
	assert.Len(t, families, 3)

	count, err := testutil.GatherAndCount(reg,
		"grpc_requests_total",
		"grpc_requests_errors_total",
		"grpc_request_duration_seconds",
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const path = "/api/metrics"

func NewServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

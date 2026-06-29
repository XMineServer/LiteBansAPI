package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "litebans_api_http_requests_total",
		Help: "Total number of HTTP requests by route, method and status.",
	}, []string{"method", "path", "status"})

	HTTPRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "litebans_api_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds by route, method and status.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})
)

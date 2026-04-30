package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	BackendHTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_http_requests_total",
			Help: "Total number of HTTP requests handled by the backend",
		},
		[]string{"method", "route", "status_class"},
	)

	BackendHTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backend_http_request_duration_seconds",
			Help:    "HTTP request latency for backend routes",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status_class"},
	)

	BackendHTTPInFlightRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "backend_http_in_flight_requests",
			Help: "Current number of in-flight HTTP requests",
		},
	)

	BackendDependencyUp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backend_dependency_up",
			Help: "Dependency availability as seen by the backend process at runtime (1=available, 0=unavailable)",
		},
		[]string{"dependency"},
	)

	ElasticsearchRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_elasticsearch_requests_total",
			Help: "Total Elasticsearch client calls made by the backend",
		},
		[]string{"operation", "outcome"},
	)

	ElasticsearchRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backend_elasticsearch_request_duration_seconds",
			Help:    "Latency of Elasticsearch client calls made by the backend",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "outcome"},
	)

	TrinoRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_trino_requests_total",
			Help: "Total Trino client calls made by the backend",
		},
		[]string{"operation", "outcome"},
	)

	TrinoRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backend_trino_request_duration_seconds",
			Help:    "Latency of Trino client calls made by the backend",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "outcome"},
	)
)

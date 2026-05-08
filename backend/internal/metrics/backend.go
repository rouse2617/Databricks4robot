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

	BackendEvalWriteRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_eval_write_requests_total",
			Help: "Total eval result write requests handled by the backend",
		},
		[]string{"outcome"},
	)

	BackendEvalWriteDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backend_eval_write_duration_seconds",
			Help:    "Latency of eval result write requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"outcome"},
	)

	BackendEvalMetricKeysTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_eval_metric_keys_total",
			Help: "Count of metric keys processed from eval payloads",
		},
		[]string{"kind"}, // registered_queryable | unregistered_or_unqueryable
	)

	BackendEvalMetricUpsertsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_eval_metric_upserts_total",
			Help: "Total metric projection upserts into asset_metrics",
		},
		[]string{"outcome"}, // ok | error
	)

	BackendEvalMetricUpsertDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backend_eval_metric_upsert_duration_seconds",
			Help:    "Latency of single metric projection upsert operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"outcome"}, // ok | error
	)

	BackendLakehouseSyncCountDiffRatio = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "backend_lakehouse_sync_count_diff_ratio",
			Help: "Latest sync count diff ratio between PostgreSQL and Iceberg",
		},
	)

	BackendLakehouseSyncPGTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "backend_lakehouse_sync_pg_total_count",
			Help: "Latest sync check PostgreSQL total count",
		},
	)

	BackendLakehouseSyncIcebergTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "backend_lakehouse_sync_iceberg_total_count",
			Help: "Latest sync check Iceberg total count",
		},
	)

	BackendLakehouseSyncCheckedAtUnixSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "backend_lakehouse_sync_checked_at_unix_seconds",
			Help: "Unix timestamp of the latest successful sync check",
		},
	)

	BackendLakehouseSyncSourceUp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backend_lakehouse_sync_source_up",
			Help: "Whether a sync-status source is currently serving successful data",
		},
		[]string{"source"}, // realtime | sync_reconciliation
	)
)

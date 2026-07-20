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

	OutboxPendingEvents = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "outbox_pending_events",
			Help: "Current number of pending outbox events",
		},
		[]string{"transport"},
	)

	OutboxRelayPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "outbox_relay_published_total",
			Help: "Total outbox events marked published by relay",
		},
		[]string{"ordering_key_kind"},
	)

	OutboxRelayFlushDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "outbox_relay_flush_duration_seconds",
			Help:    "Duration of outbox relay flush cycles",
			Buckets: prometheus.DefBuckets,
		},
	)

	OutboxRelayParallelKeys = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "outbox_relay_parallel_keys",
			Help: "Configured parallel ordering key workers used by relay flush",
		},
	)

	OutboxInternalBusDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "outbox_internal_bus_depth",
			Help: "Current in-memory internal bus queue depth",
		},
	)

	OutboxSubscriberBatchSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "outbox_subscriber_batch_size",
			Help:    "Batch size observed by ES subscriber",
			Buckets: []float64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000},
		},
	)

	OutboxSubscriberHandleDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "outbox_subscriber_handle_duration_seconds",
			Help:    "Duration of ES subscriber handling operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"op"}, // index | delete
	)

	OutboxSubscriberESErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "outbox_subscriber_es_errors_total",
			Help: "Total Elasticsearch write errors observed by outbox subscriber",
		},
		[]string{"kind"}, // conflict | other
	)

	OutboxEventLagSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "outbox_event_lag_seconds",
			Help:    "Lag between event creation and successful publish",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 30, 60, 120, 300},
		},
	)

	// Outbox watermark gauges — refreshed every time GET /api/v1/search/sync-progress
	// runs (the Frontend dashboard polls it, and Cloud Monitoring uptime/sync jobs
	// can hit it on a fixed schedule). All five mirror the JSON fields of
	// SyncProgress so dashboards and alerts can pivot on the same numbers.
	OutboxPGMaxEventSeq = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "outbox_pg_max_event_seq",
			Help: "MAX(event_seq) across asset_events (PG side high-water mark).",
		},
	)

	OutboxPublishedMaxSeq = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "outbox_published_max_seq",
			Help: "MAX(event_seq) where publish_state='published' (handed off to MQ, not yet ES-applied).",
		},
	)

	OutboxSeqLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "outbox_seq_lag",
			Help: "outbox_pg_max_event_seq - outbox_published_max_seq. PG→MQ backlog; primary alerting signal.",
		},
	)

	OutboxESAppliedMinSeq = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "outbox_es_applied_min_seq",
			Help: "MIN(applied_seq) across es_sync_checkpoint shards. Conservative ES-applied watermark; 0 until all shards report.",
		},
	)

	OutboxConsumerLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "outbox_consumer_lag",
			Help: "outbox_published_max_seq - outbox_es_applied_min_seq when both non-zero. MQ→ES backlog.",
		},
	)

	// Lakehouse Bronze watermark gauges — refreshed every time
	// GET /api/v1/lakehouse/sync-progress runs (Frontend dashboard polls it).
	// Mirror the PG→Iceberg pipeline (Cloud Run Job bronze-incremental); see
	// docs/review/lakehouse-incremental-ingestion.md.
	LakehouseBronzeMaxEventSeq = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "lakehouse_bronze_max_event_seq",
			Help: "MAX(event_seq) in iceberg.robot.bronze_asset_events (Bronze high-water mark; cursor for the next ingest run).",
		},
	)

	LakehouseBronzeLagEvents = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "lakehouse_bronze_lag_events",
			Help: "outbox_published_max_seq - lakehouse_bronze_max_event_seq. PG-published events not yet in Bronze; primary alert signal.",
		},
	)

	LakehouseBronzeLastIngestedUnixSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "lakehouse_bronze_last_ingested_unix_seconds",
			Help: "MAX(_ingested_at) from bronze_asset_events as a Unix epoch. time()-this is Bronze data staleness; 0 when table is empty.",
		},
	)

	QueryRunRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_query_run_requests_total",
			Help: "Total number of /api/v1/queries/run requests by outcome",
		},
		[]string{"outcome"},
	)

	QueryRunDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backend_query_run_duration_seconds",
			Help:    "End-to-end latency of /api/v1/queries/run requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"outcome"},
	)

	QueryRunPhaseDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backend_query_run_phase_duration_seconds",
			Help:    "Latency of /api/v1/queries/run execution phases",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"phase", "outcome"}, // compile_validate | es_recall | pg_refine
	)

	QueryRunCandidateIDs = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "backend_query_run_candidate_ids",
			Help:    "Number of candidate asset IDs produced by query recall phase",
			Buckets: []float64{0, 1, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000, 500000},
		},
	)

	// DispatcherTemplateFallbackTotal counts batch submissions where the
	// pinned template version was missing from both the template_version
	// column and filter_json, forcing a fallback to the template's current
	// active version (CYB-3677). Non-zero on legacy rows only; growth on new
	// batches indicates the pinning write path regressed.
	// CYB-3678 per-cluster dispatch instrumentation.
	DispatcherSubmitDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "backend_dispatcher_submit_duration_seconds",
			Help:    "Per-item workflow submission latency (AIMD input)",
			Buckets: prometheus.DefBuckets,
		},
	)
	DispatcherEffectiveConcurrency = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backend_dispatcher_effective_concurrency",
			Help: "Governor-adjusted in-flight submit concurrency per cluster",
		},
		[]string{"cluster"},
	)
	DispatcherChannelPaused = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backend_dispatcher_channel_paused",
			Help: "1 when a cluster's dispatch channel is paused by dispatcher config (CYB-3679)",
		},
		[]string{"cluster"},
	)
	DispatcherChannelBreakerTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_dispatcher_channel_breaker_total",
			Help: "Cluster channels skipped after consecutive transient failures",
		},
		[]string{"cluster"},
	)
	DispatcherDLQTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_dispatcher_dlq_total",
			Help: "Batch items dead-lettered (failed), by reason",
		},
		[]string{"reason"},
	)

	DispatcherTemplateFallbackTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "backend_dispatcher_template_fallback_total",
			Help: "Batch submissions that fell back to the active template version (pin missing)",
		},
	)

	// DispatcherStaleItems gauges the watcher→reconciler gap: how many
	// pipeline_runs are terminal (Succeeded/Failed/Error) while their linked
	// backfill_item still shows pending or submitted. Without a webhook, the
	// reconciler is the only backstop; if this gauge stays >0 after a full
	// reconciler cycle (60s), there are items the reconciler cannot reach
	// (no pipeline_run_id, orphan, or bug).
	DispatcherStaleItems = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "backend_dispatcher_stale_items",
			Help: "Number of terminal pipeline_runs whose backfill_items have not yet been synced (watcher→reconciler gap)",
		},
	)
)

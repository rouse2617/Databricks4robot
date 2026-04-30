package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	OutboxWorkerBatches = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_worker_batches_total",
		Help: "Number of outbox worker batch iterations completed",
	})
	OutboxEventsPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_worker_events_published_total",
		Help: "Number of asset_events rows marked published after ES sink",
	})
	OutboxEventsFailedMark = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_worker_events_failed_mark_total",
		Help: "Number of asset_events rows marked failed (retry) after ES error",
	})
	OutboxBulkDocs = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_worker_bulk_docs_total",
		Help: "Number of ES documents successfully indexed by outbox bulk",
	})
	OutboxPendingGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_worker_pending_total",
		Help: "Current count of asset_events with publish_state=pending",
	})
	OutboxBulkPartialFailure = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_bulk_partial_failure_total",
		Help: "Number of ES bulk calls with at least one per-doc failure (partial failure)",
	})
	// Task 2.13: batch processing duration histogram (§11.1)
	OutboxBatchDurationMs = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "outbox_worker_batch_duration_ms",
		Help:    "Duration of a single processBatch call in milliseconds",
		Buckets: []float64{10, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
	})
	// Task 2.14: age of the oldest pending event in seconds (§11.1)
	OutboxOldestPendingAge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_oldest_pending_age_seconds",
		Help: "Age of the oldest pending asset_event in seconds (now - MIN(occurred_at) WHERE pending)",
	})
	// Task 2.15: transport-level ES bulk failure counter (§11.1)
	OutboxESBulkFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_es_bulk_failures_total",
		Help: "Number of ES bulk calls that failed at the transport/HTTP level (entire batch rejected)",
	})

	// ── §11.1 remaining metrics (Task 5.4) ──────────────────────────────────

	// outbox_worker_batch_size: histogram of events fetched per processBatch call
	OutboxBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "outbox_worker_batch_size",
		Help:    "Number of events fetched per processBatch call",
		Buckets: []float64{1, 10, 50, 100, 250, 500, 1000},
	})

	// outbox_worker_retry_max: gauge of MAX(retry_count) among pending events
	OutboxRetryMax = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_worker_retry_max",
		Help: "Maximum retry_count among pending asset_events (WARN > 5 sustained 10 min)",
	})

	// outbox_sink_lag_seq: gauge of MAX(event_seq) - last_published_seq
	OutboxSinkLagSeq = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_sink_lag_seq",
		Help: "Sequence lag: MAX(event_seq) - last_published_seq for es_assets sink",
	})

	// outbox_tombstone_failure_total: counter of DELETE doc failures
	OutboxTombstoneFailure = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_tombstone_failure_total",
		Help: "Number of ES DELETE doc (tombstone) failures",
	})

	// outbox_fetch_deadlock_total: counter of FetchPending deadlock/errors
	OutboxFetchDeadlock = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_fetch_deadlock_total",
		Help: "Number of FetchPending deadlock or connection errors",
	})

	// outbox_cursor_deadlock_total: counter of MarkPublishedAndAdvanceCursor deadlocks
	OutboxCursorDeadlock = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_cursor_deadlock_total",
		Help: "Number of MarkPublishedAndAdvanceCursor deadlock or transaction errors",
	})
)

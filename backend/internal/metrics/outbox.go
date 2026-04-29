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
		Name: "outbox_pending_events",
		Help: "Current count of asset_events with publish_state=pending",
	})
)

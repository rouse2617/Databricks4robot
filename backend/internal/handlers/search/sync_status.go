package search

import "time"

// SyncInfo describes how the search index is kept in sync with PostgreSQL.
type SyncInfo struct {
	ElasticsearchOK           bool `json:"elasticsearch_ok"`
	OutboxRelayEnabled        bool `json:"outbox_relay_enabled"`
	OutboxESSubscriberEnabled bool `json:"outbox_es_subscriber_enabled"`
	// SearchIndexMode is one of: unavailable | outbox_es_subscriber | local_reconcile | manual
	SearchIndexMode    string `json:"search_index_mode"`
	Env                string `json:"env,omitempty"`
	AdminSearchEnabled bool   `json:"admin_search_enabled"`
}

// SyncProgress exposes runtime PG→ES sync progress signals.
type SyncProgress struct {
	PostgresAssetsTotal    int64   `json:"postgres_assets_total"`
	ElasticsearchDocsTotal int64   `json:"elasticsearch_docs_total"`
	PGESGap                int64   `json:"pg_es_gap"`
	PGESSyncRatio          float64 `json:"pg_es_sync_ratio"`
	// OutboxPendingEvents is all rows with publish_state=pending (includes rows
	// younger than the relay safety lag that are not yet claimable).
	OutboxPendingEvents int64 `json:"outbox_pending_events"`
	// OutboxPendingClaimable is pending rows old enough for the relay to claim
	// (same visibility rule as ClaimPendingSafe for the pending branch).
	OutboxPendingClaimable int64 `json:"outbox_pending_claimable"`
	// OutboxProcessingEvents is rows currently claimed by a relay (processing).
	OutboxProcessingEvents int64 `json:"outbox_processing_events"`
	// OutboxRelaySafetyLagSec is the configured relay visibility delay for new pending rows.
	OutboxRelaySafetyLagSec float64 `json:"outbox_relay_safety_lag_sec"`
	OldestPendingAgeSec     float64 `json:"oldest_pending_age_sec"`
	// PGMaxEventSeq is MAX(event_seq) across asset_events. O(1) watermark used
	// to compute SeqLag without expensive COUNT(*) over assets/ES at 50B+ scale.
	PGMaxEventSeq int64 `json:"pg_max_event_seq"`
	// OutboxPublishedMaxSeq is MAX(event_seq) WHERE publish_state='published'.
	// This is "handed off to MQ" — NOT "consumed by ES". Conservative high-water
	// mark for the PG→relay→bus boundary.
	OutboxPublishedMaxSeq int64 `json:"outbox_published_max_seq"`
	// SeqLag = PGMaxEventSeq - OutboxPublishedMaxSeq. Number of events still in
	// pending/processing on the PG side. Primary alerting signal.
	SeqLag int64 `json:"seq_lag"`
	// ESAppliedMinSeq is MIN(applied_seq) across es_sync_checkpoint shards.
	// Conservative high-water mark: every event_seq <= this value has been
	// applied to ES on its shard. 0 when no checkpoint rows yet (fresh
	// deployment) OR when fewer shards have reported than configured (so
	// ConsumerLag stays 0 instead of being optimistically advanced).
	ESAppliedMinSeq int64 `json:"es_applied_min_seq"`
	// ConsumerLag = OutboxPublishedMaxSeq - ESAppliedMinSeq when both are
	// non-zero. Reflects the MQ → ES segment of the pipeline (events handed
	// to the bus but not yet ack'd by ES). 0 when ESAppliedMinSeq is 0.
	ConsumerLag int64     `json:"consumer_lag"`
	CheckedAt   time.Time `json:"checked_at"`
}

-- Per-shard checkpoint for the PG→Pub/Sub→ES subscriber.
--
-- The ES subscriber hashes each event's asset_id (or mcap routing key) to a
-- fixed number of logical shards and after every successful _bulk it upserts
-- MAX(event_seq) it has applied for that shard. The cross-shard MIN of those
-- values is a conservative high-water mark: "every event_seq <= MIN has been
-- consumed by ES somewhere", which is exactly what /api/v1/search/sync-progress
-- exposes as consumer_lag.
--
-- Row count = OUTBOX_ES_CHECKPOINT_SHARDS (default 16). Each successful bulk
-- emits at most `shards-in-batch` UPSERTs, so write amplification stays flat
-- regardless of asset_events volume.
CREATE TABLE IF NOT EXISTS es_sync_checkpoint (
    shard_id    INTEGER     PRIMARY KEY,
    applied_seq BIGINT      NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE  es_sync_checkpoint              IS 'ES subscriber per-shard high-water mark of applied event_seq (see consumer_lag in /search/sync-progress).';
COMMENT ON COLUMN es_sync_checkpoint.shard_id     IS 'FNV(routing_key) % OUTBOX_ES_CHECKPOINT_SHARDS; stable for the lifetime of the deployment.';
COMMENT ON COLUMN es_sync_checkpoint.applied_seq  IS 'GREATEST(applied_seq, batch_max_seq) — only advances on successful ES bulk.';

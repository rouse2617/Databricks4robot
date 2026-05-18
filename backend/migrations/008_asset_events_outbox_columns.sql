-- 008_asset_events_outbox_columns.sql
-- Add missing columns required by the Outbox Worker:
--   retry_count  — tracks delivery retry attempts
--   last_error   — stores the most recent failure message
--   aggregate_type — routing key for downstream sinks (default 'asset')

ALTER TABLE asset_events ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE asset_events ADD COLUMN IF NOT EXISTS last_error TEXT;
ALTER TABLE asset_events ADD COLUMN IF NOT EXISTS aggregate_type TEXT NOT NULL DEFAULT 'asset';

-- Index for monitoring high-retry events (P1 alert: retry_count > 10).
CREATE INDEX IF NOT EXISTS idx_asset_events_retry_high
    ON asset_events (retry_count) WHERE retry_count > 5 AND publish_state = 'pending';

-- Index for aggregate_type routing (future multi-sink fan-out).
CREATE INDEX IF NOT EXISTS idx_asset_events_aggregate_time
    ON asset_events (aggregate_type, occurred_at DESC);

-- 010_outbox_dlq.sql
-- Dead Letter Queue table for permanently failed outbox events.
-- Events that exceed the retry threshold are moved here for manual inspection.

CREATE TABLE IF NOT EXISTS outbox_dlq (
  dlq_id          BIGSERIAL PRIMARY KEY,
  event_id        UUID NOT NULL,
  event_seq       BIGINT NOT NULL,
  event_type      TEXT NOT NULL,
  aggregate_type  TEXT NOT NULL DEFAULT 'asset',
  asset_id        UUID,
  mcap_file_id    UUID,
  event_source    TEXT NOT NULL DEFAULT 'backend',
  event_payload   JSONB NOT NULL DEFAULT '{}',
  retry_count     INT NOT NULL DEFAULT 0,
  last_error      TEXT,
  original_created_at TIMESTAMPTZ NOT NULL,
  moved_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at     TIMESTAMPTZ,
  resolution      TEXT  -- 'retried', 'discarded', 'manual_fix'
);

CREATE INDEX IF NOT EXISTS idx_outbox_dlq_unresolved
  ON outbox_dlq (moved_at DESC) WHERE resolved_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_dlq_asset
  ON outbox_dlq (asset_id) WHERE resolved_at IS NULL;

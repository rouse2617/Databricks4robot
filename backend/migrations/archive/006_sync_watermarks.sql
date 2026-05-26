-- 006_sync_watermarks.sql — Watermark tracking for Dagster incremental sync.
-- Stores the last synced updated_at per source table for postgres_to_bronze pipeline.

CREATE TABLE IF NOT EXISTS sync_watermarks (
    table_name      TEXT PRIMARY KEY,
    watermark       TIMESTAMPTZ NOT NULL,
    dagster_run_id  TEXT,
    synced_rows     BIGINT DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

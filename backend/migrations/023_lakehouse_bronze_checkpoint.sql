-- Single-row checkpoint for the bronze-incremental Cloud Run Job.
-- The job upserts (applied_seq, ingested_at) at the end of each successful
-- run; backend reads it to power GET /api/v1/lakehouse/sync-progress and the
-- lakehouse_bronze_* Prometheus gauges.
--
-- Why PG instead of Trino: backend's Trino catalog cannot see the
-- BigLake-managed Iceberg tables (different metastore), so we cannot rely
-- on Trino to read the Bronze high-water mark. The job itself knows
-- exactly what it just committed and is a strictly more accurate source.
CREATE TABLE IF NOT EXISTS lakehouse_bronze_checkpoint (
    id          INTEGER     PRIMARY KEY,
    applied_seq BIGINT      NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL,
    run_id      TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT lakehouse_bronze_checkpoint_singleton CHECK (id = 1)
);

COMMENT ON TABLE  lakehouse_bronze_checkpoint              IS 'PG→Iceberg Bronze high-water mark; one row (id=1) updated by Cloud Run Job bronze-incremental after each successful run.';
COMMENT ON COLUMN lakehouse_bronze_checkpoint.applied_seq  IS 'MAX(event_seq) the job confirmed committed to Bronze on its latest run.';
COMMENT ON COLUMN lakehouse_bronze_checkpoint.ingested_at  IS 'Wall-clock time of the job that produced applied_seq (matches Bronze _ingested_at for that batch).';
COMMENT ON COLUMN lakehouse_bronze_checkpoint.run_id       IS 'Cloud Run Job execution name (e.g. bronze-incremental-jvzkr); useful for cross-referencing logs.';

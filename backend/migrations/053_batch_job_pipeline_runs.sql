-- Link pipeline runs to batch (backfill) jobs so list UIs can paginate and filter.

ALTER TABLE pipeline_runs
    ADD COLUMN IF NOT EXISTS batch_job_id TEXT;

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_batch_job_id
    ON pipeline_runs (batch_job_id)
    WHERE batch_job_id IS NOT NULL;

ALTER TABLE backfill_items
    ADD COLUMN IF NOT EXISTS pipeline_run_id TEXT;

CREATE INDEX IF NOT EXISTS idx_backfill_items_pipeline_run_id
    ON backfill_items (pipeline_run_id)
    WHERE pipeline_run_id IS NOT NULL;

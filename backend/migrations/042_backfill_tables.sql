-- Backfill jobs: batch re-processing of assets through a pipeline template.
-- Each job tracks overall progress across all items.

CREATE TABLE IF NOT EXISTS backfill_jobs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template_id TEXT NOT NULL REFERENCES pipeline_templates(id),
    filter_json JSONB,                    -- asset filter criteria
    total_count INT NOT NULL DEFAULT 0,
    completed_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'running',  -- running | paused | completed | failed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backfill_jobs_status ON backfill_jobs(status);
CREATE INDEX IF NOT EXISTS idx_backfill_jobs_created_at ON backfill_jobs(created_at DESC);

-- Individual backfill items: one per asset to be processed.
CREATE TABLE IF NOT EXISTS backfill_items (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES backfill_jobs(id) ON DELETE CASCADE,
    asset_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending | running | completed | failed | cancelled
    workflow_name TEXT,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backfill_items_job_id ON backfill_items(job_id);
CREATE INDEX IF NOT EXISTS idx_backfill_items_status ON backfill_items(status);

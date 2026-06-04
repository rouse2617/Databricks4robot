-- 052: Link fan-out batch runs (one template × N assets → N workflows).

ALTER TABLE pipeline_runs
  ADD COLUMN IF NOT EXISTS batch_run_id UUID NULL;

ALTER TABLE pipeline_deployments
  ADD COLUMN IF NOT EXISTS batch_run_id UUID NULL;

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_batch_run_id
  ON pipeline_runs (batch_run_id)
  WHERE batch_run_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pipeline_deployments_batch_run_id
  ON pipeline_deployments (batch_run_id)
  WHERE batch_run_id IS NOT NULL;

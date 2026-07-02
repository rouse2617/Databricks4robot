-- M1 batch operability fields: lock template version and pilot gate state.

ALTER TABLE backfill_jobs
  ADD COLUMN IF NOT EXISTS template_version INT,
  ADD COLUMN IF NOT EXISTS pilot_count INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS pilot_phase TEXT NOT NULL DEFAULT 'none';

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_batch_job_id
  ON pipeline_runs(batch_job_id)
  WHERE batch_job_id IS NOT NULL;

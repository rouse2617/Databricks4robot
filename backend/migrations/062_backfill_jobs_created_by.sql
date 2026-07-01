-- CYB-2835: add created_by to backfill_jobs so the batch job list can show the owner.
-- finished_at already exists (migration 042); this migration only adds created_by.
ALTER TABLE backfill_jobs
  ADD COLUMN IF NOT EXISTS created_by TEXT;

CREATE INDEX IF NOT EXISTS idx_backfill_jobs_created_by ON backfill_jobs(created_by);

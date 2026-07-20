-- CYB-3681 (load-test fix): the run-event watcher loads active-status runs
-- oldest-first (FindActiveRunSummaries) instead of "most recent N". This
-- composite index serves `status = ANY(active) ORDER BY created_at ASC LIMIT n`
-- straight off the index — active runs are a small fraction of pipeline_runs
-- (~2.5k of ~60k), so the scan stays cheap even as history grows.
CREATE INDEX IF NOT EXISTS "idx_pipeline_runs_status_created_at"
  ON "pipeline_runs" ("status", "created_at");

-- 050: Add ledger_state to pipeline_runs
-- Tracks whether a run's events have been synced by the watcher.
-- Managed by the watcher scan loop; read by the frontend to determine
-- whether to show "DataBrew 运行" or "外部 Workflow" and whether to
-- suppress DataBrew-only UI modules.

ALTER TABLE pipeline_runs
  ADD COLUMN ledger_state TEXT NOT NULL DEFAULT 'pending';

COMMENT ON COLUMN pipeline_runs.ledger_state IS '
  pending    — never scanned by watcher (default for new runs)
  has_ledger — watcher confirmed events exist (COUNT(events) > 0)
  no_ledger  — watcher scanned but found no events
  backfilling— watcher actively backfilling events from Argo
';

-- Partial index for the watcher health query; only non-has_ledger rows
-- need to be reported.
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_ledger_state
  ON pipeline_runs(ledger_state)
  WHERE ledger_state != 'has_ledger';

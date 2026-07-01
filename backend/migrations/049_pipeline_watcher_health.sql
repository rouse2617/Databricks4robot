-- Extend pipeline watcher state with durable health diagnostics.
--
-- Backward-compatible: existing rows keep their ID/last_synced_at/limit/error
-- and gain zero/default health counters.

ALTER TABLE pipeline_run_watcher_state
  ADD COLUMN IF NOT EXISTS last_scan_started_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_scan_finished_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_error_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_synced_run_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS total_scans BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS total_errors BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS scan_lag_seconds BIGINT;

CREATE INDEX IF NOT EXISTS idx_pipeline_run_events_run_filters
  ON pipeline_run_events (run_id, event_type, subject_type, status, occurred_at);

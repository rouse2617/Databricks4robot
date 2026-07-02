-- Add durable pipeline run event ledger.
--
-- Events are append-only facts used by the execution detail timeline and
-- future audit/notification flows. run_id is intentionally not a foreign key:
-- delete operations can still leave an auditable deletion event even if the
-- run row is removed.

CREATE TABLE IF NOT EXISTS pipeline_run_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id TEXT NOT NULL,
  workflow_name TEXT,
  event_type TEXT NOT NULL,
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  status TEXT,
  message TEXT,
  reason TEXT,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  idempotency_key TEXT NOT NULL,
  sequence BIGSERIAL NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT pipeline_run_events_event_type_not_blank CHECK (btrim(event_type) <> ''),
  CONSTRAINT pipeline_run_events_subject_type_not_blank CHECK (btrim(subject_type) <> ''),
  CONSTRAINT pipeline_run_events_subject_id_not_blank CHECK (btrim(subject_id) <> ''),
  CONSTRAINT pipeline_run_events_idempotency_key_not_blank CHECK (btrim(idempotency_key) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_run_events_run_id_idempotency
  ON pipeline_run_events (run_id, idempotency_key);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_events_run_timeline
  ON pipeline_run_events (run_id, sequence);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_events_workflow_time
  ON pipeline_run_events (workflow_name, occurred_at);

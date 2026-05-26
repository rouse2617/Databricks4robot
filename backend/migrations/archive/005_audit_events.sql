-- 005_audit_events.sql — Audit events table for tracking core write operations.

CREATE TABLE IF NOT EXISTS audit_events (
    event_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor           TEXT NOT NULL,
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_ids    TEXT[] NOT NULL,
    request_summary JSONB DEFAULT '{}',
    request_id      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_action_created
  ON audit_events (action, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_events_actor_created
  ON audit_events (actor, created_at DESC);

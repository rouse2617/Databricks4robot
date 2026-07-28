-- CYB-3778: Replace scheduled_tasks (REST polling) with subscription_tasks (Pub/Sub).
-- Drop the old table entirely — dev has one test row, prod has none.

DROP TABLE IF EXISTS scheduled_tasks;

CREATE TABLE subscription_tasks (
    id                    TEXT PRIMARY KEY,
    name                  TEXT NOT NULL,
    enabled               BOOLEAN NOT NULL DEFAULT true,

    template_id           TEXT NOT NULL,
    template_version      INTEGER,
    target_id             TEXT NOT NULL,
    scheduling            JSONB NOT NULL DEFAULT '{}',

    -- Pub/Sub config
    project_id            TEXT NOT NULL,
    subscription_id       TEXT NOT NULL,
    pull_interval_seconds INTEGER NOT NULL DEFAULT 10,
    max_messages_per_pull INTEGER NOT NULL DEFAULT 1000,

    -- Observation (written by scheduler, not by CRUD)
    last_run_at           TIMESTAMPTZ,
    last_run_status       TEXT,
    last_batch_id         TEXT,
    last_error            TEXT,
    last_success_at       TIMESTAMPTZ,

    created_by            TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_pull_interval CHECK (pull_interval_seconds > 0),
    CONSTRAINT chk_max_messages  CHECK (max_messages_per_pull > 0 AND max_messages_per_pull <= 10000)
);

CREATE INDEX idx_subscription_tasks_enabled ON subscription_tasks (enabled) WHERE enabled = true;

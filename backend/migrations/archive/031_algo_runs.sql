-- CYB-1018: algo_runs execution-event entity + run_id FK wiring.

BEGIN;

CREATE TABLE IF NOT EXISTS algo_runs (
    run_id           TEXT PRIMARY KEY CHECK (run_id ~ '^[0-9A-Za-z]{16}$'),

    algo_name        TEXT NOT NULL,
    algo_version     TEXT NOT NULL,
    algo_kind        TEXT NOT NULL CHECK (algo_kind IN ('processing', 'split', 'qa', 'enrichment')),

    triggered_by     TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'running', 'ok', 'failed', 'cancelled')),

    started_at       TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    duration_ns      BIGINT GENERATED ALWAYS AS (
        CASE
            WHEN finished_at IS NOT NULL AND started_at IS NOT NULL
            THEN (EXTRACT(EPOCH FROM (finished_at - started_at))::BIGINT * 1000000000)
            ELSE NULL
        END
    ) STORED,

    input_filter     JSONB NOT NULL DEFAULT '{}'::jsonb,
    input_asset_ids  TEXT[],
    params           JSONB NOT NULL DEFAULT '{}'::jsonb,

    code_commit      TEXT,
    image_digest     TEXT,
    pipeline_name    TEXT,
    pipeline_version TEXT,

    assets_processed INT,
    assets_succeeded INT,
    assets_failed    INT,
    actions_created  INT,
    metrics_written  INT,

    outputs          JSONB NOT NULL DEFAULT '{}'::jsonb,

    cpu_seconds      BIGINT,
    gpu_seconds      BIGINT,
    cost_usd_micros  BIGINT,

    error_class      TEXT,
    error_message    TEXT,

    tenant_id        TEXT,
    project_id       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    row_version      BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_algo_runs_algo
    ON algo_runs (algo_name, algo_version, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_algo_runs_status
    ON algo_runs (status) WHERE status IN ('pending', 'running', 'failed');
CREATE INDEX IF NOT EXISTS idx_algo_runs_started
    ON algo_runs (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_algo_runs_triggered
    ON algo_runs (triggered_by) WHERE triggered_by LIKE 'manual:%';

-- Legacy run_id values cannot FK until registered in algo_runs.
UPDATE asset_algo_latest SET run_id = NULL WHERE run_id IS NOT NULL;
UPDATE actions SET run_id = NULL WHERE run_id IS NOT NULL;
UPDATE asset_eval_results SET run_id = NULL WHERE run_id IS NOT NULL;
UPDATE asset_tags SET run_id = NULL WHERE run_id IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_aal_run') THEN
        ALTER TABLE asset_algo_latest
            ADD CONSTRAINT fk_aal_run FOREIGN KEY (run_id)
            REFERENCES algo_runs (run_id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_act_run') THEN
        ALTER TABLE actions
            ADD CONSTRAINT fk_act_run FOREIGN KEY (run_id)
            REFERENCES algo_runs (run_id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_eval_run') THEN
        ALTER TABLE asset_eval_results
            ADD CONSTRAINT fk_eval_run FOREIGN KEY (run_id)
            REFERENCES algo_runs (run_id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_atags_run') THEN
        ALTER TABLE asset_tags
            ADD CONSTRAINT fk_atags_run FOREIGN KEY (run_id)
            REFERENCES algo_runs (run_id) ON DELETE SET NULL;
    END IF;
END $$;

COMMIT;

-- Phase 1.5: Eval / Metrics tables
-- Plain SQL — no migration framework needed.

CREATE TABLE IF NOT EXISTS asset_eval_results (
    eval_result_id      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Keep as plain text for cross-environment compatibility. In some local
    -- DB snapshots, assets.asset_id may be UUID; in others it's TEXT.
    asset_id            TEXT         NOT NULL,
    mcap_file_id        TEXT,

    target_type         TEXT         NOT NULL DEFAULT 'segment',
    target_id           TEXT         NOT NULL DEFAULT '',

    eval_name           TEXT         NOT NULL,
    eval_version        TEXT         NOT NULL,
    parameter_version   TEXT,
    run_id              TEXT,
    status              TEXT         NOT NULL DEFAULT 'ok',

    result_payload      JSONB        NOT NULL DEFAULT '{}',
    output_uri          TEXT,
    summary_uri         TEXT,

    source_type         TEXT         NOT NULL DEFAULT 'algo',
    source_name         TEXT,
    source_version      TEXT,

    started_at          TIMESTAMPTZ,
    finished_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_eval_results_asset
    ON asset_eval_results (asset_id, target_type, target_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_eval_results_name_ver
    ON asset_eval_results (eval_name, eval_version, parameter_version, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_eval_results_run_id
    ON asset_eval_results (run_id) WHERE run_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS asset_metrics (
    -- Keep as plain text for cross-environment compatibility.
    asset_id            TEXT         NOT NULL,
    target_type         TEXT         NOT NULL DEFAULT 'segment',
    target_id           TEXT         NOT NULL DEFAULT '',

    metric_key          TEXT         NOT NULL,
    metric_type         TEXT         NOT NULL DEFAULT 'float',
    metric_unit         TEXT,

    metric_value        DOUBLE PRECISION,
    metric_value_int    BIGINT,
    metric_value_text   TEXT,
    metric_value_bool   BOOLEAN,

    eval_name           TEXT         NOT NULL,
    eval_version        TEXT         NOT NULL,
    parameter_version   TEXT,
    run_id              TEXT,

    source_type         TEXT         NOT NULL DEFAULT 'algo',
    source_name         TEXT,
    confidence          DOUBLE PRECISION,

    recorded_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),

    PRIMARY KEY (asset_id, target_type, target_id, metric_key, eval_name, eval_version)
);

CREATE INDEX IF NOT EXISTS idx_asset_metrics_key_value
    ON asset_metrics (metric_key, metric_value);
CREATE INDEX IF NOT EXISTS idx_asset_metrics_asset
    ON asset_metrics (asset_id, target_type, target_id);

-- updated_at triggers
CREATE OR REPLACE TRIGGER trg_asset_eval_results_updated_at
    BEFORE UPDATE ON asset_eval_results
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE TRIGGER trg_asset_metrics_updated_at
    BEFORE UPDATE ON asset_metrics
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

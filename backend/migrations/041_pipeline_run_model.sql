-- 041_pipeline_run_model.sql — First-class pipeline runs and execution targets.

CREATE TABLE IF NOT EXISTS execution_targets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    cluster TEXT NOT NULL,
    namespace TEXT NOT NULL,
    service_account TEXT NOT NULL DEFAULT '',
    argo_server_url TEXT NOT NULL DEFAULT '',
    argo_auth_secret_ref TEXT NOT NULL DEFAULT '',
    argo_insecure_skip_verify BOOLEAN NOT NULL DEFAULT FALSE,
    argo_ca_cert_ref TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    status TEXT NOT NULL DEFAULT 'available',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    resource_defaults JSONB NOT NULL DEFAULT '{}'::jsonb,
    quota_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_execution_targets_one_default
    ON execution_targets (is_default)
    WHERE is_default;

CREATE TABLE IF NOT EXISTS pipeline_runs (
    id TEXT PRIMARY KEY,
    template_id TEXT REFERENCES pipeline_templates(id) ON DELETE SET NULL,
    pipeline_name TEXT NOT NULL,
    template_version INT,
    workflow_name TEXT NOT NULL UNIQUE,
    execution_target_id TEXT NOT NULL REFERENCES execution_targets(id),
    target_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'Pending',
    node_count INT NOT NULL DEFAULT 0,
    asset_ids TEXT[] NOT NULL DEFAULT '{}'::text[],
    asset_count INT NOT NULL DEFAULT 0,
    no_asset_run BOOLEAN NOT NULL DEFAULT FALSE,
    manifest TEXT,
    pipeline_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    argo_namespace TEXT NOT NULL,
    argo_workflow_uid TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_created_at
    ON pipeline_runs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status_updated_at
    ON pipeline_runs (status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_template_created_at
    ON pipeline_runs (template_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_asset_ids
    ON pipeline_runs USING GIN (asset_ids);

CREATE TABLE IF NOT EXISTS pipeline_run_nodes (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    pipeline_node_id TEXT NOT NULL,
    argo_node_id TEXT NOT NULL DEFAULT '',
    argo_node_name TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    template_name TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    phase TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    pod_name TEXT NOT NULL DEFAULT '',
    host_node_name TEXT NOT NULL DEFAULT '',
    children TEXT[] NOT NULL DEFAULT '{}'::text[],
    inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    outputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    resources_duration JSONB NOT NULL DEFAULT '{}'::jsonb,
    resource_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    log_ref TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_run_nodes_run_argo_node
    ON pipeline_run_nodes (run_id, argo_node_id)
    WHERE argo_node_id <> '';
CREATE INDEX IF NOT EXISTS idx_pipeline_run_nodes_run_phase
    ON pipeline_run_nodes (run_id, phase);

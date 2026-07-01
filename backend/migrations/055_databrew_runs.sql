-- 054: Unified DataBrew Run core table and build extension tables.

CREATE TABLE IF NOT EXISTS databrew_runs (
    id                      TEXT PRIMARY KEY,
    type                    TEXT NOT NULL,
    name                    TEXT NOT NULL,
    status                  TEXT NOT NULL DEFAULT 'Pending',
    runtime                 TEXT NOT NULL DEFAULT 'argo',
    runtime_namespace       TEXT NOT NULL DEFAULT '',
    runtime_resource_name   TEXT NOT NULL,
    runtime_uid             TEXT NOT NULL DEFAULT '',
    owner                   TEXT NOT NULL DEFAULT '',
    created_by              TEXT NOT NULL DEFAULT '',
    message                 TEXT NOT NULL DEFAULT '',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at              TIMESTAMPTZ,
    finished_at             TIMESTAMPTZ,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_databrew_runs_runtime_resource
    ON databrew_runs (runtime_namespace, runtime_resource_name);

CREATE INDEX IF NOT EXISTS idx_databrew_runs_type_created_at
    ON databrew_runs (type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_databrew_runs_status_updated_at
    ON databrew_runs (status, updated_at DESC);

INSERT INTO databrew_runs (
    id, type, name, status, runtime, runtime_namespace, runtime_resource_name,
    runtime_uid, owner, created_by, message, created_at, started_at, finished_at, updated_at
)
SELECT
    id,
    'pipeline',
    pipeline_name,
    status,
    'argo',
    argo_namespace,
    workflow_name,
    argo_workflow_uid,
    COALESCE(owner, ''),
    '',
    COALESCE(message, ''),
    created_at,
    started_at,
    finished_at,
    updated_at
FROM pipeline_runs
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS component_build_runs (
    run_id            TEXT PRIMARY KEY REFERENCES databrew_runs(id) ON DELETE CASCADE,
    component_id      TEXT NOT NULL DEFAULT '',
    repo_url          TEXT NOT NULL DEFAULT '',
    git_ref           TEXT NOT NULL DEFAULT '',
    commit_sha        TEXT NOT NULL DEFAULT '',
    dockerfile        TEXT NOT NULL DEFAULT 'Dockerfile',
    build_context     TEXT NOT NULL DEFAULT '.',
    image_repository  TEXT NOT NULL DEFAULT '',
    image_tag         TEXT NOT NULL DEFAULT '',
    image_digest      TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS rag_build_runs (
    run_id              TEXT PRIMARY KEY REFERENCES databrew_runs(id) ON DELETE CASCADE,
    knowledge_base_id   TEXT NOT NULL DEFAULT '',
    datasource_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    embedding_model     TEXT NOT NULL DEFAULT '',
    vector_index_name   TEXT NOT NULL DEFAULT '',
    release_version     TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS component_releases (
    id              TEXT PRIMARY KEY,
    component_id    TEXT NOT NULL,
    source_commit   TEXT NOT NULL DEFAULT '',
    image           TEXT NOT NULL DEFAULT '',
    image_tag       TEXT NOT NULL DEFAULT '',
    image_digest    TEXT NOT NULL DEFAULT '',
    release_label   TEXT NOT NULL DEFAULT '',
    build_run_id    TEXT REFERENCES databrew_runs(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_component_releases_component_id
    ON component_releases (component_id, created_at DESC);

CREATE OR REPLACE FUNCTION sync_databrew_run_from_pipeline()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO databrew_runs (
        id, type, name, status, runtime, runtime_namespace, runtime_resource_name,
        runtime_uid, owner, created_by, message, created_at, started_at, finished_at, updated_at
    ) VALUES (
        NEW.id,
        'pipeline',
        NEW.pipeline_name,
        NEW.status,
        'argo',
        NEW.argo_namespace,
        NEW.workflow_name,
        NEW.argo_workflow_uid,
        COALESCE(NEW.owner, ''),
        '',
        COALESCE(NEW.message, ''),
        NEW.created_at,
        NEW.started_at,
        NEW.finished_at,
        NEW.updated_at
    )
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        status = EXCLUDED.status,
        runtime_namespace = EXCLUDED.runtime_namespace,
        runtime_resource_name = EXCLUDED.runtime_resource_name,
        runtime_uid = EXCLUDED.runtime_uid,
        owner = EXCLUDED.owner,
        message = EXCLUDED.message,
        started_at = EXCLUDED.started_at,
        finished_at = EXCLUDED.finished_at,
        updated_at = EXCLUDED.updated_at;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sync_databrew_run_from_pipeline ON pipeline_runs;
CREATE TRIGGER trg_sync_databrew_run_from_pipeline
    AFTER INSERT OR UPDATE ON pipeline_runs
    FOR EACH ROW
    EXECUTE FUNCTION sync_databrew_run_from_pipeline();

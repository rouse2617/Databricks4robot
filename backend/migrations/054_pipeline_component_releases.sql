CREATE TABLE IF NOT EXISTS pipeline_component_releases (
    id TEXT PRIMARY KEY,
    component_id TEXT NOT NULL,
    task_name TEXT NOT NULL,
    task_path TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    owner TEXT NOT NULL DEFAULT '',
    release_label TEXT NOT NULL,
    channel TEXT NOT NULL DEFAULT 'dev',
    source_repo TEXT NOT NULL DEFAULT '',
    source_ref TEXT NOT NULL DEFAULT '',
    source_commit TEXT NOT NULL DEFAULT '',
    build_id TEXT NOT NULL DEFAULT '',
    image_repo TEXT NOT NULL DEFAULT '',
    image_tag TEXT NOT NULL DEFAULT '',
    image_digest TEXT NOT NULL DEFAULT '',
    runtime_image TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'failed',
    selectable BOOLEAN NOT NULL DEFAULT FALSE,
    validation_status TEXT NOT NULL DEFAULT 'failed',
    validation_errors JSONB NOT NULL DEFAULT '[]'::jsonb,
    runtime_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    technical_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_synced_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_component_releases_component_label
    ON pipeline_component_releases(component_id, release_label);

CREATE INDEX IF NOT EXISTS idx_pipeline_component_releases_task_name
    ON pipeline_component_releases(task_name);

CREATE INDEX IF NOT EXISTS idx_pipeline_component_releases_status
    ON pipeline_component_releases(status);

CREATE INDEX IF NOT EXISTS idx_pipeline_component_releases_selectable
    ON pipeline_component_releases(selectable);

CREATE INDEX IF NOT EXISTS idx_pipeline_component_releases_source_commit
    ON pipeline_component_releases(source_commit);

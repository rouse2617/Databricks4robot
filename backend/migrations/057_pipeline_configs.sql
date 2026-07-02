-- 057: Standalone pipeline config files and immutable file versions.

CREATE TABLE IF NOT EXISTS pipeline_configs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    owner TEXT NOT NULL DEFAULT '',
    scope VARCHAR(16) NOT NULL DEFAULT 'dev',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    file_type TEXT NOT NULL,
    lifecycle TEXT NOT NULL DEFAULT 'draft',
    current_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_pipeline_configs_file_type CHECK (file_type IN ('yaml', 'json')),
    CONSTRAINT chk_pipeline_configs_lifecycle CHECK (lifecycle IN ('draft', 'ready', 'deprecated'))
);

CREATE TABLE IF NOT EXISTS pipeline_config_versions (
    id TEXT PRIMARY KEY,
    config_id TEXT NOT NULL REFERENCES pipeline_configs(id) ON DELETE RESTRICT,
    version INT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    content TEXT NOT NULL,
    content_sha256 TEXT NOT NULL,
    content_size_bytes INT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_pipeline_config_versions_status CHECK (status IN ('draft', 'ready', 'deprecated')),
    CONSTRAINT chk_pipeline_config_versions_size CHECK (content_size_bytes >= 0 AND content_size_bytes <= 1048576),
    CONSTRAINT uq_pipeline_config_versions_config_version UNIQUE (config_id, version)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_configs_owner_scope
    ON pipeline_configs(owner, scope);

CREATE INDEX IF NOT EXISTS idx_pipeline_configs_lifecycle
    ON pipeline_configs(lifecycle);

CREATE INDEX IF NOT EXISTS idx_pipeline_config_versions_config_id
    ON pipeline_config_versions(config_id, version DESC);

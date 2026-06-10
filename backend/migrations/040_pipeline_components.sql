CREATE TABLE IF NOT EXISTS pipeline_components (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    image TEXT NOT NULL,
    tag TEXT NOT NULL DEFAULT 'latest',
    source TEXT NOT NULL DEFAULT 'custom',
    input_ports JSONB NOT NULL DEFAULT '[]',
    output_ports JSONB NOT NULL DEFAULT '[]',
    resources JSONB,
    env_vars JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pipeline_components_name ON pipeline_components(name);
CREATE INDEX IF NOT EXISTS idx_pipeline_components_source ON pipeline_components(source);

-- 039_pipeline_tables.sql — Pipeline templates and deployments.
-- Adds tables for storing reusable pipeline templates and tracking pipeline deployments
-- triggered from pipeline-to-workflow orchestration.

CREATE TABLE IF NOT EXISTS pipeline_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    pipeline JSONB NOT NULL,
    node_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pipeline_deployments (
    id TEXT PRIMARY KEY,
    template_id TEXT REFERENCES pipeline_templates(id),
    pipeline_name TEXT NOT NULL,
    workflow_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'Pending',
    node_count INT NOT NULL DEFAULT 0,
    manifest TEXT,
    pipeline_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

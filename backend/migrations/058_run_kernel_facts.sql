-- Durable Run Kernel facts for parent/child lineage and immutable inputs.

UPDATE pipeline_runs
SET workflow_name = id
WHERE btrim(workflow_name) = '';

CREATE TABLE IF NOT EXISTS run_relations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_run_id TEXT NOT NULL,
  child_run_id TEXT NOT NULL,
  relation_type TEXT NOT NULL,
  asset_id TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT run_relations_parent_not_blank CHECK (btrim(parent_run_id) <> ''),
  CONSTRAINT run_relations_child_not_blank CHECK (btrim(child_run_id) <> ''),
  CONSTRAINT run_relations_type_not_blank CHECK (btrim(relation_type) <> ''),
  CONSTRAINT run_relations_no_self_edge CHECK (parent_run_id <> child_run_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_run_relations_parent_child_type
  ON run_relations (parent_run_id, child_run_id, relation_type);

CREATE INDEX IF NOT EXISTS idx_run_relations_parent_created
  ON run_relations (parent_run_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_run_relations_child_created
  ON run_relations (child_run_id, created_at DESC);

CREATE TABLE IF NOT EXISTS run_inputs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id TEXT NOT NULL,
  node_id TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL,
  ref_id TEXT NOT NULL DEFAULT '',
  ref_version TEXT NOT NULL DEFAULT '',
  file_name TEXT NOT NULL DEFAULT '',
  mount_path TEXT NOT NULL DEFAULT '',
  target_filename TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL DEFAULT '',
  projection_key TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT run_inputs_run_not_blank CHECK (btrim(run_id) <> ''),
  CONSTRAINT run_inputs_type_not_blank CHECK (btrim(type) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_run_inputs_unique_fact
  ON run_inputs (
    run_id,
    type,
    node_id,
    ref_id,
    ref_version,
    mount_path,
    target_filename,
    projection_key
  );

CREATE INDEX IF NOT EXISTS idx_run_inputs_run_created
  ON run_inputs (run_id, created_at ASC);

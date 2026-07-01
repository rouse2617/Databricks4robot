-- Add pipeline observability tables for asset-node snapshots, notification
-- candidates, and watcher state.

CREATE TABLE IF NOT EXISTS pipeline_run_asset_nodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id TEXT NOT NULL,
  asset_id TEXT NOT NULL,
  pipeline_node_id TEXT NOT NULL,
  argo_node_id TEXT,
  display_name TEXT,
  status TEXT,
  message TEXT,
  pod_name TEXT,
  log_ref TEXT,
  estimated_cost_usd DECIMAL(16,8),
  cost_source TEXT NOT NULL DEFAULT 'not_available',
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT pipeline_run_asset_nodes_asset_id_not_blank CHECK (btrim(asset_id) <> ''),
  CONSTRAINT pipeline_run_asset_nodes_node_id_not_blank CHECK (btrim(pipeline_node_id) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_run_asset_nodes_unique
  ON pipeline_run_asset_nodes (run_id, asset_id, pipeline_node_id);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_asset_nodes_run_status
  ON pipeline_run_asset_nodes (run_id, status);

CREATE TABLE IF NOT EXISTS pipeline_run_notification_candidates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id TEXT NOT NULL,
  event_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  status TEXT,
  message TEXT,
  sink_type TEXT NOT NULL DEFAULT 'candidate',
  delivery_status TEXT NOT NULL DEFAULT 'pending',
  idempotency_key TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT pipeline_run_notification_candidates_key_not_blank CHECK (btrim(idempotency_key) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_run_notification_candidates_key
  ON pipeline_run_notification_candidates (idempotency_key);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_notification_candidates_run
  ON pipeline_run_notification_candidates (run_id, created_at);

CREATE TABLE IF NOT EXISTS pipeline_run_watcher_state (
  id TEXT PRIMARY KEY,
  last_synced_at TIMESTAMPTZ,
  active_scan_limit INTEGER NOT NULL DEFAULT 100,
  last_error TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

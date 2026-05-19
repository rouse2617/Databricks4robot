-- 004_asset_query_indexes.sql - indexes for high-traffic asset discovery queries.

CREATE OR REPLACE FUNCTION asset_algo_statuses(algo JSONB)
RETURNS TEXT[]
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
  SELECT COALESCE(array_agg(value ORDER BY key), ARRAY[]::TEXT[])
  FROM jsonb_each_text(COALESCE(algo, '{}'::JSONB))
  WHERE key LIKE '%:status';
$$;

CREATE INDEX IF NOT EXISTS idx_assets_active_updated_at
  ON assets (updated_at DESC)
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_active_status_updated_at
  ON assets (status, updated_at DESC)
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_meta_owner_active
  ON assets ((cf_meta#>>'{owner}'))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_meta_reviewer_active
  ON assets ((cf_meta#>>'{reviewer}'))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_meta_env_active
  ON assets ((cf_meta#>>'{env}'))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_meta_task_active
  ON assets ((cf_meta#>>'{task}'))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_meta_duration_num_active
  ON assets (((cf_meta#>>'{duration_sec}')::NUMERIC))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_meta_delivery_count_active
  ON assets ((COALESCE((cf_meta->>'delivery_count')::INT, 0)))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_tag_priority_active
  ON assets ((cf_tag#>>'{priority}'))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_tag_quality_active
  ON assets ((cf_tag#>>'{quality}'))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_tag_scene_active
  ON assets ((cf_tag#>>'{scene}'))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_tag_task_active
  ON assets ((cf_tag#>>'{task}'))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_cf_tag_batch_active
  ON assets ((cf_tag#>>'{batch}'))
  WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_assets_algo_statuses_gin_active
  ON assets USING GIN (asset_algo_statuses(cf_algo))
  WHERE is_deleted = FALSE;

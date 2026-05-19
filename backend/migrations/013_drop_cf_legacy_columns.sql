-- 013_drop_cf_legacy_columns.sql
-- Remove cf_* legacy columns and indexes
--
-- This migration completes P2-9: Remove cf_* legacy layer.
-- After this migration, all data lives in typed columns and projection tables.
--
-- Change History:
--   - 2026-05-05: Initial version

-- ============================================================
-- Drop expression indexes on cf_meta
-- ============================================================
DROP INDEX IF EXISTS idx_assets_cf_meta_owner_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_reviewer_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_env_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_task_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_duration_num_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_delivery_count_active;

-- ============================================================
-- Drop expression indexes on cf_tag
-- ============================================================
DROP INDEX IF EXISTS idx_assets_cf_tag_priority_active;
DROP INDEX IF EXISTS idx_assets_cf_tag_quality_active;
DROP INDEX IF EXISTS idx_assets_cf_tag_scene_active;
DROP INDEX IF EXISTS idx_assets_cf_tag_task_active;
DROP INDEX IF EXISTS idx_assets_cf_tag_batch_active;

-- ============================================================
-- Drop GIN indexes on cf_* columns
-- ============================================================
DROP INDEX IF EXISTS idx_assets_cf_meta_gin;
DROP INDEX IF EXISTS idx_assets_cf_algo_gin;
DROP INDEX IF EXISTS idx_assets_cf_tag_gin;
DROP INDEX IF EXISTS idx_assets_cf_files_gin;
DROP INDEX IF EXISTS idx_assets_algo_statuses_gin_active;

-- ============================================================
-- Drop cf_* columns from assets
-- ============================================================
ALTER TABLE assets DROP COLUMN IF EXISTS cf_meta;
ALTER TABLE assets DROP COLUMN IF EXISTS cf_algo;
ALTER TABLE assets DROP COLUMN IF EXISTS cf_tag;
ALTER TABLE assets DROP COLUMN IF EXISTS cf_files;

-- ============================================================
-- Drop cf_* columns from mcap_files
-- ============================================================
ALTER TABLE mcap_files DROP COLUMN IF EXISTS cf_meta;
ALTER TABLE mcap_files DROP COLUMN IF EXISTS cf_process;

-- ============================================================
-- Drop cf_* columns from deliveries
-- ============================================================
ALTER TABLE deliveries DROP COLUMN IF EXISTS cf_meta;

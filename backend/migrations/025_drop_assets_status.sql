-- Drop legacy assets.status column. Replaced by assets.lifecycle_state.
DROP INDEX IF EXISTS idx_assets_active_status_updated_at;
ALTER TABLE assets DROP COLUMN IF EXISTS status;
CREATE INDEX IF NOT EXISTS idx_assets_active_lifecycle_updated_at
  ON assets (lifecycle_state, updated_at DESC)
  WHERE is_deleted = FALSE;

-- Drop legacy assets.status column. Replaced by assets.lifecycle_state.
ALTER TABLE assets DROP COLUMN IF EXISTS status;

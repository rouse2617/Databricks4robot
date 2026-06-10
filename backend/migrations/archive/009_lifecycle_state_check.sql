-- 009_lifecycle_state_check.sql
-- Add CHECK constraint on lifecycle_state to enforce valid values.

ALTER TABLE assets
  ADD CONSTRAINT chk_lifecycle_state
  CHECK (lifecycle_state IN (
    'created', 'processing', 'ready', 'delivered',
    'archived', 'superseded', 'failed', 'rejected'
  ));

-- CYB-2774: Track retry attempts for batch items to support
-- lease-based crash recovery. On worker crash, the stale reaper
-- resets the item to pending and increments attempts. Items
-- exceeding max_attempts (3) are marked as failed to prevent
-- infinite crash loops.

ALTER TABLE backfill_items
  ADD COLUMN attempts INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN backfill_items.attempts IS
  'Number of times this item has been claimed for processing. Reset to 0 on user-initiated rerun.';

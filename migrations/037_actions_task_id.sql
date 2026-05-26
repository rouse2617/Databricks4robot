-- CYB-1219: Add task_id placeholder column to actions table
--
-- P1.5 annotation_tasks placeholder. Nullable; FK constraint will be added
-- when annotation_tasks table is created.
-- See docs/review/unified-asset-catalog/schema.md §13, §17.2 B4

ALTER TABLE actions ADD COLUMN task_id TEXT;

COMMENT ON COLUMN actions.task_id IS
    'P1.5: FK → annotation_tasks(task_id), nullable placeholder, not yet enforced';

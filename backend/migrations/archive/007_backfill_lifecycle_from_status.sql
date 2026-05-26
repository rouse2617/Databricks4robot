-- 007_backfill_lifecycle_from_status.sql
-- P0-2 minimal: align lifecycle_state with legacy status for rows still at default 'created'
-- while status already carries review semantics (see internal/postgres/lifecycle.go).

UPDATE assets
SET lifecycle_state = CASE status::text
    WHEN 'approved' THEN 'ready'
    WHEN 'rejected' THEN 'rejected'
    WHEN 'archived' THEN 'archived'
    WHEN 'superseded' THEN 'superseded'
    ELSE lifecycle_state
END
WHERE is_deleted = FALSE
  AND lifecycle_state = 'created'
  AND status::text IN ('approved', 'rejected', 'archived', 'superseded');

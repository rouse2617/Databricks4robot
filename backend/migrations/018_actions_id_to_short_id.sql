-- 018_actions_id_to_short_id.sql
-- Migrate actions.action_id from UUID to 8-char alnum short id.

-- 1) Convert UUID/text action_id to deterministic short ids first.
--    NOTE: `left(md5(...), 8)` yields [0-9a-f]{8}, which satisfies the
--    repository-wide short-id contract [0-9A-Za-z]{8}.
UPDATE actions
SET action_id = left(md5(action_id::text), 8)
WHERE length(action_id::text) <> 8
   OR action_id::text !~ '^[0-9A-Za-z]{8}$';

-- 2) Switch column type/constraints to short-id shape.
ALTER TABLE actions
    ALTER COLUMN action_id TYPE TEXT USING action_id::text,
    ALTER COLUMN action_id DROP DEFAULT;

ALTER TABLE actions
    DROP CONSTRAINT IF EXISTS actions_id_chk;

ALTER TABLE actions
    ADD CONSTRAINT actions_id_chk CHECK (action_id ~ '^[0-9A-Za-z]{8}$');

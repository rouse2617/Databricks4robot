-- CYB-1015: asset_tags multi-source PK refactor.
--
-- Goal: allow multiple rows on (asset_id, tag_key) when they come from
-- different sources/versions. Identity becomes
--   (asset_id, tag_key, tag_value, source_type, source_version_norm)
-- where source_version_norm is a STORED generated column (Postgres does not
-- allow COALESCE expressions inside a PRIMARY KEY directly).
--
-- This migration is destructive in the sense that it drops and replaces the
-- table's primary key. No row data is rewritten beyond the column additions.
--
-- Pre-flight (read-only) is documented in
-- openspec/changes/CYB-1015-asset-tags-multi-source-pk/migration-plan.md §3.

BEGIN;

-- 1. Drop the old single-row-per-key primary key.
ALTER TABLE asset_tags DROP CONSTRAINT IF EXISTS asset_tags_pkey;

-- 2. Add a surrogate primary key for stable single-column reference.
ALTER TABLE asset_tags ADD COLUMN IF NOT EXISTS id BIGSERIAL PRIMARY KEY;

-- 3. applied_at — the moment the assertion was made by the source. created_at
--    stays as the row's first-insert timestamp. For legacy rows we backfill
--    applied_at from created_at so query semantics ("most recent assertion")
--    work immediately.
ALTER TABLE asset_tags
    ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE asset_tags SET applied_at = created_at WHERE applied_at = created_at;
-- (no-op if no legacy rows; safe to run repeatedly)

-- 4. Generated column to normalize NULL source_version into '' so it can
--    participate in a UNIQUE constraint deterministically.
ALTER TABLE asset_tags
    ADD COLUMN IF NOT EXISTS source_version_norm TEXT
        GENERATED ALWAYS AS (COALESCE(source_version, '')) STORED;

-- 5. New identity constraint — same source/version cannot duplicate the same
--    (key, value) on the same asset; different sources can coexist.
ALTER TABLE asset_tags
    ADD CONSTRAINT uq_asset_tags_identity
        UNIQUE (asset_id, tag_key, tag_value, source_type, source_version_norm);

-- 6. Indexes for the read paths described in
--    docs/review/unified-asset-catalog/design/asset-tagging.md §3.1.
CREATE INDEX IF NOT EXISTS idx_atags_lookup
    ON asset_tags (tag_key, tag_value, asset_id);
CREATE INDEX IF NOT EXISTS idx_atags_source
    ON asset_tags (source_type, source_name, source_version);
CREATE INDEX IF NOT EXISTS idx_atags_propagation
    ON asset_tags (tag_key) WHERE tag_key LIKE 'compliance.%';
CREATE INDEX IF NOT EXISTS idx_atags_run
    ON asset_tags (run_id) WHERE run_id IS NOT NULL;

-- 7. FK on run_id deferred until CYB-1018 lands algo_runs:
-- ALTER TABLE asset_tags
--     ADD CONSTRAINT fk_atags_run FOREIGN KEY (run_id)
--     REFERENCES algo_runs(run_id) ON DELETE SET NULL;

COMMIT;

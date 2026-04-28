-- 013_schema_evolution_backfill_tags.sql — Backfill asset_tags from assets.cf_tag JSONB.
-- Phase B (Backfill): expands cf_tag key-value pairs into asset_tags rows.
-- Idempotent: uses INSERT ... ON CONFLICT (asset_id, tag_key) DO UPDATE.
-- Processes in batches of 1000 assets using FOR UPDATE SKIP LOCKED.

DO $
DECLARE
    batch_size INT := 1000;
    rows_affected INT;
BEGIN
    LOOP
        -- Select a batch of assets that have non-empty cf_tag and haven't
        -- been fully backfilled yet. We mark processed assets by checking
        -- whether at least one tag row exists; the ON CONFLICT handles
        -- partial backfills idempotently.
        WITH batch AS (
            SELECT asset_id, cf_tag
            FROM assets
            WHERE cf_tag IS NOT NULL
              AND cf_tag != '{}'::jsonb
              AND NOT EXISTS (
                  SELECT 1 FROM asset_tags at
                  WHERE at.asset_id = assets.asset_id
                    AND at.source_type = 'system'
              )
            LIMIT batch_size
            FOR UPDATE SKIP LOCKED
        )
        INSERT INTO asset_tags (asset_id, tag_key, tag_value, tag_type, source_type)
        SELECT
            b.asset_id,
            kv.key,
            kv.value,
            'string',
            'system'
        FROM batch b,
             jsonb_each_text(b.cf_tag) AS kv(key, value)
        ON CONFLICT (asset_id, tag_key) DO UPDATE SET
            tag_value  = EXCLUDED.tag_value,
            updated_at = now();

        GET DIAGNOSTICS rows_affected = ROW_COUNT;
        EXIT WHEN rows_affected = 0;
    END LOOP;
END $;

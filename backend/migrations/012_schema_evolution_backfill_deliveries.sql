-- 012_schema_evolution_backfill_deliveries.sql — Backfill deliveries real columns from cf_meta JSONB.
-- Phase B (Backfill): populates promoted columns for historical rows.
-- Idempotent: only updates rows where real columns are still at default/NULL.
-- Processes in batches of 1000 using FOR UPDATE SKIP LOCKED.

DO $
DECLARE
    batch_size INT := 1000;
    rows_updated INT;
BEGIN
    LOOP
        WITH batch AS (
            SELECT delivery_id
            FROM deliveries
            WHERE
                -- Only rows that haven't been backfilled yet:
                -- nullable columns still NULL, or NOT NULL columns still at default
                (manifest_uri IS NULL AND cf_meta->>'manifest_uri' IS NOT NULL)
                OR (contract_id IS NULL AND cf_meta->>'contract_id' IS NOT NULL)
                OR (item_count = 0 AND cf_meta->>'asset_count' IS NOT NULL AND (cf_meta->>'asset_count')::BIGINT != 0)
                OR (delivered_by IS NULL AND cf_meta->>'owner' IS NOT NULL)
            LIMIT batch_size
            FOR UPDATE SKIP LOCKED
        )
        UPDATE deliveries d SET
            manifest_uri = COALESCE(d.manifest_uri, d.cf_meta->>'manifest_uri'),
            contract_id  = COALESCE(d.contract_id, d.cf_meta->>'contract_id'),
            item_count   = CASE
                               WHEN d.item_count = 0 THEN
                                   COALESCE((d.cf_meta->>'asset_count')::BIGINT, 0)
                               ELSE d.item_count
                           END,
            delivered_by = COALESCE(d.delivered_by, d.cf_meta->>'owner')
        FROM batch
        WHERE d.delivery_id = batch.delivery_id;

        GET DIAGNOSTICS rows_updated = ROW_COUNT;
        EXIT WHEN rows_updated = 0;
    END LOOP;
END $;

-- 010_schema_evolution_backfill_assets.sql — Backfill assets real columns from cf_meta JSONB.
-- Phase B (Backfill): populates promoted columns for historical rows.
-- Idempotent: only updates rows where real columns are still at default/NULL.
-- Processes in batches of 1000 using FOR UPDATE SKIP LOCKED.

DO $$
DECLARE
    batch_size INT := 1000;
    rows_updated INT;
BEGIN
    LOOP
        WITH batch AS (
            SELECT asset_id
            FROM assets
            WHERE
                -- Only rows that haven't been backfilled yet:
                -- nullable columns still NULL, or NOT NULL columns still at default
                (owner IS NULL AND cf_meta->>'owner' IS NOT NULL)
                OR (reviewer IS NULL AND cf_meta->>'reviewer' IS NOT NULL)
                OR (duration_ms IS NULL AND cf_meta->>'duration_sec' IS NOT NULL)
                OR (lifecycle_state = 'created' AND status IS DISTINCT FROM 'approved')
                OR (asset_type = 'segment' AND cf_meta->>'type' IS NOT NULL AND cf_meta->>'type' != 'segment')
                OR (delivery_count = 0 AND cf_meta->>'delivery_count' IS NOT NULL AND (cf_meta->>'delivery_count')::INT != 0)
                OR (last_delivered_to IS NULL AND cf_meta->>'last_delivered_to' IS NOT NULL)
                OR (last_delivered_at IS NULL AND cf_meta->>'last_delivered_at' IS NOT NULL)
                OR (retention_tier IS NULL AND cf_meta->>'retention_tier' IS NOT NULL)
            LIMIT batch_size
            FOR UPDATE SKIP LOCKED
        )
        UPDATE assets a SET
            owner            = COALESCE(a.owner, a.cf_meta->>'owner'),
            reviewer         = COALESCE(a.reviewer, a.cf_meta->>'reviewer'),
            duration_ms      = COALESCE(a.duration_ms, (a.cf_meta->>'duration_sec')::NUMERIC * 1000),
            lifecycle_state  = CASE
                                   WHEN a.lifecycle_state = 'created' THEN
                                       CASE a.status
                                           WHEN 'approved'   THEN 'ready'
                                           WHEN 'rejected'   THEN 'rejected'
                                           WHEN 'archived'   THEN 'archived'
                                           WHEN 'superseded' THEN 'superseded'
                                           ELSE 'created'
                                       END
                                   ELSE a.lifecycle_state
                               END,
            asset_type       = CASE
                                   WHEN a.asset_type = 'segment' THEN
                                       COALESCE(a.cf_meta->>'type', 'segment')
                                   ELSE a.asset_type
                               END,
            delivery_count   = CASE
                                   WHEN a.delivery_count = 0 THEN
                                       COALESCE((a.cf_meta->>'delivery_count')::INT, 0)
                                   ELSE a.delivery_count
                               END,
            last_delivered_to = COALESCE(a.last_delivered_to, a.cf_meta->>'last_delivered_to'),
            last_delivered_at = COALESCE(a.last_delivered_at, (a.cf_meta->>'last_delivered_at')::TIMESTAMPTZ),
            retention_tier   = COALESCE(a.retention_tier, a.cf_meta->>'retention_tier')
        FROM batch
        WHERE a.asset_id = batch.asset_id;

        GET DIAGNOSTICS rows_updated = ROW_COUNT;
        EXIT WHEN rows_updated = 0;
    END LOOP;
END $$;

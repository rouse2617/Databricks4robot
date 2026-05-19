-- 012_sync_delivery_item_count.sql — Align deliveries.item_count with delivery_items
-- and promote legacy cf_meta fields into typed columns where the latter are empty.
--
-- Run automatically on fresh Docker Postgres (initdb). For existing volumes, apply once:
--   psql "$DATABASE_URL" -f backend/migrations/012_sync_delivery_item_count.sql

BEGIN;

-- Authoritative counts from the junction table.
UPDATE deliveries d
SET item_count = s.cnt
FROM (
  SELECT d0.delivery_id, COUNT(di.asset_id)::bigint AS cnt
  FROM deliveries d0
  LEFT JOIN delivery_items di ON di.delivery_id = d0.delivery_id
  WHERE d0.is_deleted = FALSE
  GROUP BY d0.delivery_id
) s
WHERE d.delivery_id = s.delivery_id;

-- Promote cf_meta → typed columns (seed / old rows often only had JSONB cf_meta).
UPDATE deliveries
SET
  manifest_uri = COALESCE(NULLIF(TRIM(manifest_uri), ''), NULLIF(cf_meta->>'manifest_uri', '')),
  contract_id = COALESCE(NULLIF(TRIM(contract_id), ''), NULLIF(cf_meta->>'contract_id', '')),
  delivered_by = COALESCE(NULLIF(TRIM(delivered_by), ''), NULLIF(cf_meta->>'owner', ''))
WHERE is_deleted = FALSE
  AND cf_meta IS NOT NULL
  AND cf_meta <> '{}'::jsonb;

-- Note lives in metadata JSON for the current API/repo layer.
UPDATE deliveries
SET metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('note', cf_meta->>'note')
WHERE is_deleted = FALSE
  AND cf_meta ? 'note'
  AND NULLIF(TRIM(cf_meta->>'note'), '') IS NOT NULL
  AND (
    metadata IS NULL
    OR metadata = '{}'::jsonb
    OR NOT (metadata ? 'note')
    OR NULLIF(TRIM(metadata->>'note'), '') IS NULL
  );

COMMIT;

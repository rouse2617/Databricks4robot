-- 014_schema_evolution_backfill_algo.sql — Backfill asset_algo_latest from assets.cf_algo JSONB.
-- Phase B (Backfill): parses cf_algo keys (format: <algo>@<ver>:<field>) into
-- asset_algo_latest rows — one row per unique (asset_id, algo_name).
-- Idempotent: uses INSERT ... ON CONFLICT (asset_id, algo_name) DO UPDATE.
-- Processes in batches of 1000 assets using FOR UPDATE SKIP LOCKED.

DO $
DECLARE
    batch_size INT := 1000;
    rows_affected INT;
BEGIN
    LOOP
        WITH batch AS (
            SELECT asset_id, cf_algo
            FROM assets
            WHERE cf_algo IS NOT NULL
              AND cf_algo != '{}'::jsonb
              AND NOT EXISTS (
                  SELECT 1 FROM asset_algo_latest aal
                  WHERE aal.asset_id = assets.asset_id
              )
            LIMIT batch_size
            FOR UPDATE SKIP LOCKED
        ),
        parsed AS (
            -- Extract algo_name, algo_version, field, and value from each key.
            -- Key format: <algo_name>@<version>:<field>
            -- Only keys matching the pattern are included; malformed keys are skipped.
            SELECT
                b.asset_id,
                (regexp_match(kv.key, '^([^@]+)@([^:]+):(.+)$'))[1] AS algo_name,
                (regexp_match(kv.key, '^([^@]+)@([^:]+):(.+)$'))[2] AS algo_version,
                (regexp_match(kv.key, '^([^@]+)@([^:]+):(.+)$'))[3] AS field,
                kv.value AS field_value
            FROM batch b,
                 jsonb_each_text(b.cf_algo) AS kv(key, value)
            WHERE kv.key ~ '^[^@]+@[^:]+:.+$'
        ),
        status_rows AS (
            -- Pick the status value for each (asset_id, algo_name).
            -- We prefer the :status field; use DISTINCT ON to get one row per combo.
            SELECT DISTINCT ON (p.asset_id, p.algo_name)
                p.asset_id,
                p.algo_name,
                p.algo_version,
                p.field_value AS status
            FROM parsed p
            WHERE p.field = 'status'
            ORDER BY p.asset_id, p.algo_name, p.algo_version DESC
        )
        INSERT INTO asset_algo_latest (asset_id, algo_name, algo_version, status)
        SELECT
            s.asset_id,
            s.algo_name,
            s.algo_version,
            s.status
        FROM status_rows s
        ON CONFLICT (asset_id, algo_name) DO UPDATE SET
            algo_version = EXCLUDED.algo_version,
            status       = EXCLUDED.status,
            updated_at   = now();

        GET DIAGNOSTICS rows_affected = ROW_COUNT;
        EXIT WHEN rows_affected = 0;
    END LOOP;
END $;

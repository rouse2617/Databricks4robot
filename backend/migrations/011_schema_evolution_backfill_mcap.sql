-- 011_schema_evolution_backfill_mcap.sql — Backfill mcap_files real columns from cf_meta JSONB.
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
            SELECT mcap_file_id
            FROM mcap_files
            WHERE
                -- Only rows that haven't been backfilled yet:
                -- nullable columns still NULL, or NOT NULL columns still at default
                (mcap_uri = '' AND cf_meta->>'gcs_path' IS NOT NULL)
                OR (size_bytes IS NULL AND cf_meta->>'size_bytes' IS NOT NULL)
                OR (ingest_state = 'pending' AND cf_meta->>'ingest_state' IS NOT NULL AND cf_meta->>'ingest_state' != 'pending')
                OR (start_timestamp_ns IS NULL AND cf_meta->>'start_timestamp_ns' IS NOT NULL)
                OR (end_timestamp_ns IS NULL AND cf_meta->>'end_timestamp_ns' IS NOT NULL)
                OR (channel_count IS NULL AND cf_meta->>'channel_count' IS NOT NULL)
                OR (chunk_count IS NULL AND cf_meta->>'chunk_count' IS NOT NULL)
                OR (owner IS NULL AND cf_meta->>'owner' IS NOT NULL)
            LIMIT batch_size
            FOR UPDATE SKIP LOCKED
        )
        UPDATE mcap_files m SET
            mcap_uri         = CASE
                                   WHEN m.mcap_uri = '' THEN
                                       COALESCE(m.cf_meta->>'gcs_path', '')
                                   ELSE m.mcap_uri
                               END,
            size_bytes       = COALESCE(m.size_bytes, (m.cf_meta->>'size_bytes')::BIGINT),
            ingest_state     = CASE
                                   WHEN m.ingest_state = 'pending' THEN
                                       COALESCE(m.cf_meta->>'ingest_state', 'pending')
                                   ELSE m.ingest_state
                               END,
            start_timestamp_ns = COALESCE(m.start_timestamp_ns, (m.cf_meta->>'start_timestamp_ns')::BIGINT),
            end_timestamp_ns   = COALESCE(m.end_timestamp_ns, (m.cf_meta->>'end_timestamp_ns')::BIGINT),
            channel_count    = COALESCE(m.channel_count, (m.cf_meta->>'channel_count')::INT),
            chunk_count      = COALESCE(m.chunk_count, (m.cf_meta->>'chunk_count')::INT),
            owner            = COALESCE(m.owner, m.cf_meta->>'owner')
        FROM batch
        WHERE m.mcap_file_id = batch.mcap_file_id;

        GET DIAGNOSTICS rows_updated = ROW_COUNT;
        EXIT WHEN rows_updated = 0;
    END LOOP;
END $;

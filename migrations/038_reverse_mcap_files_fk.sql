-- CYB-1217: Reverse mcap_files PK→FK direction (1:1 extension of assets)
--
-- schema.md §3.1, §17.2 B2 requires mcap_files.mcap_file_id to reference
-- assets.asset_id (mcap_files is a 1:1 extension table for raw_mcap assets).
-- Currently assets.mcap_file_id → mcap_files(mcap_file_id), which is reversed.
--
-- Step 1: Backfill placeholder raw_mcap assets for mcap_files rows that lack
-- a corresponding assets row (asset_id must match mcap_file_id).
-- Step 2: Add FK mcap_files → assets as DEFERRABLE for backward compat with
-- existing callers that may create mcap_files before their asset row.

-- ═══════════════════════════════════════════════════════════════════
-- Step 1: Backfill orphan mcap_files
-- ═══════════════════════════════════════════════════════════════════

INSERT INTO assets (
  asset_id, mcap_file_id,
  start_timestamp_ns, end_timestamp_ns, segment_locator,
  lifecycle_state, asset_type, duration_ms,
  owner, reviewer, storage_uri,
  retention_tier, expire_at,
  metadata, files,
  tenant_id, project_id,
  asset_level, delivery_count,
  is_deleted, logical_asset_id, revision, is_current,
  created_at, updated_at, version
)
SELECT
  m.mcap_file_id, m.mcap_file_id,
  m.start_timestamp_ns, m.end_timestamp_ns, '',
  'created', 'raw_mcap', m.file_duration_ms,
  COALESCE(m.owner, ''), '', COALESCE(m.mcap_uri, ''),
  COALESCE(m.retention_tier, ''), m.expire_at,
  COALESCE(m.metadata, '{}'::jsonb), '{}'::jsonb,
  m.tenant_id, m.project_id,
  0, 0,
  FALSE, NULL, 0, FALSE,
  m.created_at, m.created_at, 1
FROM mcap_files m
WHERE NOT EXISTS (
  SELECT 1 FROM assets a WHERE a.asset_id = m.mcap_file_id
);

-- ═══════════════════════════════════════════════════════════════════
-- Step 2: Add FK constraint (DEFERRABLE for transactional compatibility)
-- ═══════════════════════════════════════════════════════════════════

ALTER TABLE mcap_files ADD CONSTRAINT fk_mcap_asset
  FOREIGN KEY (mcap_file_id) REFERENCES assets(asset_id)
  DEFERRABLE INITIALLY DEFERRED;

-- CYB-1218: Allow NULL mcap_file_id for derived_asset
--
-- derived_asset across-mcap aggregation has no single MCAP file source,
-- so mcap_file_id must be nullable for this asset type.
-- Other asset types (segment, clip, frame, action, task, raw_mcap) still
-- require non-NULL mcap_file_id.
--
-- The existing assets_mcap_file_id_chk (regex pattern) automatically permits
-- NULL in Postgres CHECK semantics.
--
-- ⚠️ Off-limits zone — approved by user.
-- See openspec/changes/CYB-1218-mcap-file-id-nullable/

ALTER TABLE assets ALTER COLUMN mcap_file_id DROP NOT NULL;

ALTER TABLE assets
  DROP CONSTRAINT IF EXISTS chk_mcap_file_required,
  ADD CONSTRAINT chk_mcap_file_required CHECK (
    asset_type = 'derived_asset' OR mcap_file_id IS NOT NULL
  );

-- Allow UUID-format asset IDs for grace_video and future schema-registered asset types.
-- The 8-char alphanumeric format is still accepted; UUIDs (e.g. Grace segmentation_id) are now also valid.

-- Drop and recreate assets.logical_asset_id constraint (nullable column)

ALTER TABLE assets
    DROP CONSTRAINT IF EXISTS assets_logical_asset_id_check;

ALTER TABLE assets
    ADD CONSTRAINT assets_logical_asset_id_check
    CHECK (logical_asset_id IS NULL OR logical_asset_id ~ '^[0-9A-Za-z]{8}$' OR logical_asset_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$');

-- Drop and recreate logical_assets.logical_asset_id constraint (NOT NULL PK)

ALTER TABLE logical_assets
    DROP CONSTRAINT IF EXISTS logical_assets_logical_asset_id_check;

ALTER TABLE logical_assets
    ADD CONSTRAINT logical_assets_logical_asset_id_check
    CHECK (logical_asset_id ~ '^[0-9A-Za-z]{8}$' OR logical_asset_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$');

-- CYB-1013: logical_assets + per-revision columns on assets (no backfill).

CREATE TABLE IF NOT EXISTS logical_assets (
    logical_asset_id    TEXT PRIMARY KEY CHECK (logical_asset_id ~ '^[0-9A-Za-z]{8}$'),
    asset_type          TEXT NOT NULL,
    display_name        TEXT,
    description         TEXT,
    owner               TEXT,
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    current_revision    BIGINT NOT NULL DEFAULT 1 CHECK (current_revision >= 1),
    total_revisions     BIGINT NOT NULL DEFAULT 1 CHECK (total_revisions >= current_revision),
    metadata            JSONB NOT NULL DEFAULT '{}',
    extra               JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    row_version         BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_lassets_owner ON logical_assets(owner) WHERE owner IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_lassets_type ON logical_assets(asset_type, status);
CREATE INDEX IF NOT EXISTS idx_lassets_status ON logical_assets(status) WHERE status <> 'active';
CREATE INDEX IF NOT EXISTS idx_lassets_updated ON logical_assets(updated_at DESC);

CREATE OR REPLACE FUNCTION trg_logical_assets_type_immutable() RETURNS TRIGGER AS $$
BEGIN
  IF NEW.asset_type IS DISTINCT FROM OLD.asset_type THEN
    RAISE EXCEPTION 'logical_assets.asset_type is immutable (LA1)';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS logical_assets_type_immutable ON logical_assets;
CREATE TRIGGER logical_assets_type_immutable
  BEFORE UPDATE ON logical_assets
  FOR EACH ROW EXECUTE FUNCTION trg_logical_assets_type_immutable();

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS logical_asset_id TEXT
    CHECK (logical_asset_id IS NULL OR logical_asset_id ~ '^[0-9A-Za-z]{8}$'),
  ADD COLUMN IF NOT EXISTS revision BIGINT
    CHECK (revision IS NULL OR revision >= 1),
  ADD COLUMN IF NOT EXISTS is_current BOOLEAN;

CREATE INDEX IF NOT EXISTS idx_assets_logical ON assets(logical_asset_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_assets_current_per_logical
  ON assets(logical_asset_id)
  WHERE is_current = TRUE AND is_deleted = FALSE;

ALTER TABLE assets
  ADD CONSTRAINT fk_assets_logical_asset
  FOREIGN KEY (logical_asset_id) REFERENCES logical_assets(logical_asset_id)
  DEFERRABLE INITIALLY DEFERRED;

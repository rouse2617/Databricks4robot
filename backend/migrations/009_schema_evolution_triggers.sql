-- 009_schema_evolution_triggers.sql — Create updated_at trigger function and
-- attach BEFORE UPDATE triggers to asset_tags and asset_algo_latest.
-- All statements are idempotent.

-- ============================================================
-- Function: set_updated_at() — sets NEW.updated_at = now()
-- ============================================================
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- Trigger: trg_asset_tags_updated_at
-- ============================================================
DROP TRIGGER IF EXISTS trg_asset_tags_updated_at ON asset_tags;
CREATE TRIGGER trg_asset_tags_updated_at
    BEFORE UPDATE ON asset_tags
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================
-- Trigger: trg_asset_algo_latest_updated_at
-- ============================================================
DROP TRIGGER IF EXISTS trg_asset_algo_latest_updated_at ON asset_algo_latest;
CREATE TRIGGER trg_asset_algo_latest_updated_at
    BEFORE UPDATE ON asset_algo_latest
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

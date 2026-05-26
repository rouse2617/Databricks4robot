-- CYB-1094: asset_usage_stats tracks per-asset engagement counters.

CREATE TABLE IF NOT EXISTS asset_usage_stats (
    id bigserial PRIMARY KEY,
    asset_id text NOT NULL REFERENCES assets(asset_id),
    logical_asset_id text,
    view_count integer NOT NULL DEFAULT 0,
    last_viewed_at timestamptz,
    favorite_count integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(asset_id)
);
CREATE INDEX IF NOT EXISTS idx_asset_usage_stats_logical ON asset_usage_stats(logical_asset_id);

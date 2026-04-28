-- 007_schema_evolution_columns.sql — Add real columns to assets, mcap_files, deliveries.
-- Phase A (DDL only): promotes high-frequency fields from JSONB to real columns.
-- All statements are idempotent (ADD COLUMN IF NOT EXISTS / CREATE INDEX IF NOT EXISTS).
-- Existing cf_meta, cf_algo, cf_tag, cf_files, cf_process columns are retained unchanged.

-- ============================================================
-- assets: promoted fields from cf_meta
-- ============================================================
ALTER TABLE assets ADD COLUMN IF NOT EXISTS asset_type              TEXT        NOT NULL DEFAULT 'segment';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS lifecycle_state         TEXT        NOT NULL DEFAULT 'created';
ALTER TABLE assets ADD COLUMN IF NOT EXISTS duration_ms             BIGINT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS owner                   TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS reviewer                TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS storage_uri             TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS thumb_uri               TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS retention_tier          TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS expire_at               TIMESTAMPTZ;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS asset_level             INT         NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS parent_asset_id         UUID;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS root_asset_id           UUID;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS delivery_count          INT         NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS last_delivered_at       TIMESTAMPTZ;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS last_delivered_to       TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS segment_index           INT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS parent_start_offset_ms  BIGINT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS parent_end_offset_ms    BIGINT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS split_method            TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS split_algo_name         TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS split_algo_version      TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS split_run_id            TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS split_reason            TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS metadata                JSONB       NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS files                   JSONB       NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS tenant_id               TEXT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS project_id              TEXT;

-- assets: indexes on new columns
CREATE INDEX IF NOT EXISTS idx_assets_lifecycle
    ON assets (lifecycle_state) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_assets_asset_type
    ON assets (asset_type) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_assets_parent
    ON assets (parent_asset_id) WHERE parent_asset_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_assets_root
    ON assets (root_asset_id) WHERE root_asset_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_assets_tenant_project
    ON assets (tenant_id, project_id) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_assets_metadata_gin
    ON assets USING GIN (metadata);

-- ============================================================
-- mcap_files: promoted fields from cf_meta / cf_process
-- ============================================================
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS mcap_uri             TEXT        NOT NULL DEFAULT '';
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS size_bytes           BIGINT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS file_duration_ms     BIGINT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS start_timestamp_ns   BIGINT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS end_timestamp_ns     BIGINT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS channel_count        INT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS chunk_count          INT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS ingest_state         TEXT        NOT NULL DEFAULT 'pending';
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS vendor_id            TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS collector_id         TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS task_id              TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS device_id            TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS camera_model         TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS data_source          TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS location_id          TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS scene_id             TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS environment_id       TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS collection_method    TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS owner                TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS retention_tier       TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS expire_at            TIMESTAMPTZ;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS metadata             JSONB       NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS process_state        JSONB       NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS raw_hash_sha256      TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS tenant_id            TEXT;
ALTER TABLE mcap_files ADD COLUMN IF NOT EXISTS project_id           TEXT;

-- mcap_files: indexes
CREATE UNIQUE INDEX IF NOT EXISTS uq_mcap_files_hash_md5
    ON mcap_files (raw_hash_md5) WHERE raw_hash_md5 IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_mcap_files_tenant_project
    ON mcap_files (tenant_id, project_id) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_mcap_files_ingest_state
    ON mcap_files (ingest_state) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_mcap_files_metadata_gin
    ON mcap_files USING GIN (metadata);

-- ============================================================
-- deliveries: promoted fields from cf_meta
-- ============================================================
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS contract_id          TEXT;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS delivery_type        TEXT        NOT NULL DEFAULT 'asset_set';
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS requested_by         TEXT;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS approved_by          TEXT;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS delivered_by         TEXT;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS manifest_uri         TEXT;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS replay_manifest_uri  TEXT;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS item_count           BIGINT      NOT NULL DEFAULT 0;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS total_size_bytes     BIGINT;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS completed_at         TIMESTAMPTZ;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS metadata             JSONB       NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS tenant_id            TEXT;
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS project_id           TEXT;

-- deliveries: indexes
CREATE INDEX IF NOT EXISTS idx_deliveries_tenant_project
    ON deliveries (tenant_id, project_id) WHERE is_deleted = FALSE;

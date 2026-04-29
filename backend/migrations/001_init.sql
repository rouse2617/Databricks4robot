-- 001_init.sql — Target DDL for data4cyber PostgreSQL schema.
-- Single source of truth for fresh deployments. Executed automatically on first
-- docker-compose up via /docker-entrypoint-initdb.d mount.
--
-- Schema includes:
--   * Core tables: assets, mcap_files, deliveries, delivery_items
--   * Idempotency / audit infrastructure: idempotency_keys
--   * Schema-evolution projection tables: asset_tags, asset_algo_latest,
--     asset_relations
--   * Unified event/outbox table: asset_events + outbox_sink_cursors
--   * Triggers: set_updated_at on projection tables
--
-- Legacy cf_* JSONB columns are retained on assets/mcap_files/deliveries to
-- keep backward compatibility with existing read paths during the cutover.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================
-- Table 1: assets (core + promoted columns from cf_meta)
-- ============================================================
CREATE TABLE assets (
    asset_id            UUID            PRIMARY KEY,
    mcap_file_id        UUID            NOT NULL,
    start_timestamp_ns  BIGINT          NOT NULL,
    end_timestamp_ns    BIGINT          NOT NULL,
    status              VARCHAR(32)     NOT NULL DEFAULT 'approved',
    is_deleted          BOOLEAN         DEFAULT FALSE,
    segment_locator     CHAR(40),

    -- Legacy JSONB column families (kept for backward compatibility)
    cf_meta             JSONB           DEFAULT '{}',
    cf_algo             JSONB           DEFAULT '{}',
    cf_tag              JSONB           DEFAULT '{}',
    cf_files            JSONB           DEFAULT '{}',

    -- Promoted columns (formerly inside cf_meta)
    asset_type              TEXT        NOT NULL DEFAULT 'segment',
    lifecycle_state         TEXT        NOT NULL DEFAULT 'created',
    duration_ms             BIGINT      NOT NULL DEFAULT 0,
    owner                   TEXT        NOT NULL DEFAULT '',
    reviewer                TEXT        NOT NULL DEFAULT '',
    storage_uri             TEXT        NOT NULL DEFAULT '',
    thumb_uri               TEXT        NOT NULL DEFAULT '',
    retention_tier          TEXT        NOT NULL DEFAULT '',
    expire_at               TIMESTAMPTZ,
    asset_level             INT         NOT NULL DEFAULT 0,
    parent_asset_id         UUID,
    root_asset_id           UUID,
    delivery_count          INT         NOT NULL DEFAULT 0,
    last_delivered_at       TIMESTAMPTZ,
    last_delivered_to       TEXT        NOT NULL DEFAULT '',
    segment_index           INT,
    parent_start_offset_ms  BIGINT,
    parent_end_offset_ms    BIGINT,
    split_method            TEXT,
    split_algo_name         TEXT,
    split_algo_version      TEXT,
    split_run_id            TEXT,
    split_reason            TEXT,
    metadata                JSONB       NOT NULL DEFAULT '{}'::jsonb,
    files                   JSONB       NOT NULL DEFAULT '{}'::jsonb,
    tenant_id               TEXT,
    project_id              TEXT,

    created_at          TIMESTAMPTZ     NOT NULL,
    updated_at          TIMESTAMPTZ     NOT NULL,
    version             BIGINT          DEFAULT 1
);

-- ============================================================
-- Table 2: mcap_files (core + promoted columns from cf_meta / cf_process)
-- ============================================================
CREATE TABLE mcap_files (
    mcap_file_id    UUID            PRIMARY KEY,
    raw_hash_md5    VARCHAR(32),
    is_deleted      BOOLEAN         DEFAULT FALSE,

    -- Legacy JSONB column families (kept for backward compatibility)
    cf_meta         JSONB           DEFAULT '{}',
    cf_process      JSONB           DEFAULT '{}',

    -- Promoted columns (formerly inside cf_meta / cf_process)
    mcap_uri             TEXT        NOT NULL DEFAULT '',
    size_bytes           BIGINT      NOT NULL DEFAULT 0,
    file_duration_ms     BIGINT      NOT NULL DEFAULT 0,
    start_timestamp_ns   BIGINT      NOT NULL DEFAULT 0,
    end_timestamp_ns     BIGINT      NOT NULL DEFAULT 0,
    channel_count        INT         NOT NULL DEFAULT 0,
    chunk_count          INT         NOT NULL DEFAULT 0,
    ingest_state         TEXT        NOT NULL DEFAULT 'pending',
    vendor_id            TEXT,
    collector_id         TEXT,
    task_id              TEXT,
    device_id            TEXT,
    camera_model         TEXT,
    data_source          TEXT,
    location_id          TEXT,
    scene_id             TEXT,
    environment_id       TEXT,
    collection_method    TEXT,
    owner                TEXT        NOT NULL DEFAULT '',
    retention_tier       TEXT,
    expire_at            TIMESTAMPTZ,
    metadata             JSONB       NOT NULL DEFAULT '{}'::jsonb,
    process_state        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    raw_hash_sha256      TEXT,
    tenant_id            TEXT,
    project_id           TEXT,

    created_at      TIMESTAMPTZ     NOT NULL,
    updated_at      TIMESTAMPTZ     NOT NULL,
    version         BIGINT          DEFAULT 1
);

-- ============================================================
-- Table 3: deliveries (core + promoted columns from cf_meta)
-- ============================================================
CREATE TABLE deliveries (
    delivery_id     UUID            PRIMARY KEY,
    customer_id     VARCHAR(64)     NOT NULL,
    status          VARCHAR(16)     NOT NULL DEFAULT 'pending',
    delivered_at    TIMESTAMPTZ,
    is_deleted      BOOLEAN         DEFAULT FALSE,

    -- Legacy JSONB column families
    cf_meta         JSONB           DEFAULT '{}',

    -- Promoted columns (formerly inside cf_meta)
    contract_id          TEXT,
    delivery_type        TEXT        NOT NULL DEFAULT 'asset_set',
    requested_by         TEXT,
    approved_by          TEXT,
    delivered_by         TEXT,
    manifest_uri         TEXT,
    replay_manifest_uri  TEXT,
    item_count           BIGINT      NOT NULL DEFAULT 0,
    total_size_bytes     BIGINT,
    completed_at         TIMESTAMPTZ,
    metadata             JSONB       NOT NULL DEFAULT '{}'::jsonb,
    tenant_id            TEXT,
    project_id           TEXT,

    created_at      TIMESTAMPTZ     NOT NULL,
    updated_at      TIMESTAMPTZ     NOT NULL,
    version         BIGINT          DEFAULT 1
);

-- ============================================================
-- Table 4: delivery_items (junction)
-- ============================================================
CREATE TABLE delivery_items (
    delivery_id     UUID            NOT NULL,
    asset_id        UUID            NOT NULL,
    created_at      TIMESTAMPTZ     DEFAULT now(),
    PRIMARY KEY (delivery_id, asset_id)
);

-- ============================================================
-- Table 5: idempotency_keys
-- ============================================================
CREATE TABLE idempotency_keys (
    scope           VARCHAR(64),
    idem_key        VARCHAR(128),
    request_hash    VARCHAR(64),
    status_code     INT,
    response_json   JSONB,
    created_at      TIMESTAMPTZ,
    PRIMARY KEY (scope, idem_key)
);

-- ============================================================
-- Table 6: asset_tags — tag projection (PK: asset_id, tag_key)
-- ============================================================
CREATE TABLE asset_tags (
    asset_id        UUID                 NOT NULL REFERENCES assets(asset_id),
    tag_key         TEXT                 NOT NULL,
    tag_value       TEXT                 NOT NULL,
    tag_value_num   DOUBLE PRECISION,
    tag_value_bool  BOOLEAN,
    tag_type        TEXT                 NOT NULL DEFAULT 'string',
    source_type     TEXT                 NOT NULL DEFAULT 'human',
    source_name     TEXT,
    source_version  TEXT,
    run_id          TEXT,
    confidence      DOUBLE PRECISION,
    tenant_id       TEXT,
    project_id      TEXT,
    created_at      TIMESTAMPTZ          NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ          NOT NULL DEFAULT now(),
    PRIMARY KEY (asset_id, tag_key)
);

-- ============================================================
-- Table 7: asset_algo_latest — algo projection (PK: asset_id, algo_name)
-- ============================================================
CREATE TABLE asset_algo_latest (
    asset_id        UUID              NOT NULL REFERENCES assets(asset_id),
    algo_name       TEXT              NOT NULL,
    algo_version    TEXT              NOT NULL,
    status          TEXT              NOT NULL,
    result_tag      TEXT,
    result_score    DOUBLE PRECISION,
    result_summary  JSONB             NOT NULL DEFAULT '{}'::jsonb,
    run_id          TEXT,
    method          TEXT,
    model_uri       TEXT,
    output_uri      TEXT,
    error_code      TEXT,
    error_message   TEXT,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    tenant_id       TEXT,
    project_id      TEXT,
    updated_at      TIMESTAMPTZ       NOT NULL DEFAULT now(),
    PRIMARY KEY (asset_id, algo_name)
);

-- ============================================================
-- Table 8: asset_events — unified event/outbox (PK: event_id)
-- ============================================================
CREATE TABLE asset_events (
    event_id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_seq               BIGSERIAL    NOT NULL UNIQUE,
    event_type              TEXT         NOT NULL,
    payload_schema_version  TEXT         NOT NULL DEFAULT 'v1',
    asset_id                UUID         REFERENCES assets(asset_id),
    mcap_file_id            UUID         REFERENCES mcap_files(mcap_file_id),
    tenant_id               TEXT,
    project_id              TEXT,
    event_source            TEXT         NOT NULL,
    actor_type              TEXT,
    actor_id                TEXT,
    request_id              TEXT,
    idempotency_key         TEXT,
    run_id                  TEXT,
    occurred_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    publish_state           TEXT         NOT NULL DEFAULT 'pending',
    published_at            TIMESTAMPTZ,
    event_payload           JSONB        NOT NULL DEFAULT '{}'::jsonb,
    retry_count             INT          NOT NULL DEFAULT 0,
    last_error              TEXT
);

-- ============================================================
-- Table 9: outbox_sink_cursors — sink watermark tracking
-- ============================================================
CREATE TABLE outbox_sink_cursors (
    sink_name           TEXT         PRIMARY KEY,
    last_published_seq  BIGINT       NOT NULL DEFAULT 0,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ============================================================
-- Table 10: asset_relations — complex lineage
-- ============================================================
CREATE TABLE asset_relations (
    parent_asset_id         UUID         NOT NULL REFERENCES assets(asset_id),
    child_asset_id          UUID         NOT NULL REFERENCES assets(asset_id),
    relation_type           TEXT         NOT NULL,
    method                  TEXT,
    algo_name               TEXT,
    algo_version            TEXT,
    run_id                  TEXT,
    parent_start_offset_ms  BIGINT,
    parent_end_offset_ms    BIGINT,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (parent_asset_id, child_asset_id, relation_type)
);

-- ============================================================
-- Table 11: asset_algo_events — legacy audit log (kept for now)
-- ============================================================
CREATE TABLE asset_algo_events (
    event_id        UUID            PRIMARY KEY,
    asset_id        UUID            NOT NULL,
    algo_key        VARCHAR(128)    NOT NULL,
    prev_status     VARCHAR(16),
    new_status      VARCHAR(16)     NOT NULL,
    run_id          VARCHAR(64),
    reason          VARCHAR(256),
    created_at      TIMESTAMPTZ     NOT NULL
);

-- ============================================================
-- Indexes — core
-- ============================================================
CREATE INDEX idx_assets_mcap_file_id        ON assets (mcap_file_id);
CREATE INDEX idx_assets_status              ON assets (status);
CREATE INDEX idx_assets_segment_locator     ON assets (segment_locator);
CREATE INDEX idx_assets_created_at          ON assets (created_at DESC);
CREATE INDEX idx_delivery_items_asset_id    ON delivery_items (asset_id);
CREATE INDEX idx_algo_events_asset_created  ON asset_algo_events (asset_id, created_at DESC);
CREATE INDEX idx_algo_events_algo_status    ON asset_algo_events (algo_key, new_status, created_at DESC);
CREATE INDEX idx_algo_events_run_id         ON asset_algo_events (run_id) WHERE run_id IS NOT NULL;

-- ============================================================
-- Indexes — promoted columns on assets
-- ============================================================
CREATE INDEX idx_assets_lifecycle
    ON assets (lifecycle_state) WHERE is_deleted = FALSE;
CREATE INDEX idx_assets_asset_type
    ON assets (asset_type) WHERE is_deleted = FALSE;
CREATE INDEX idx_assets_parent
    ON assets (parent_asset_id) WHERE parent_asset_id IS NOT NULL;
CREATE INDEX idx_assets_root
    ON assets (root_asset_id) WHERE root_asset_id IS NOT NULL;
CREATE INDEX idx_assets_tenant_project
    ON assets (tenant_id, project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_assets_metadata_gin
    ON assets USING GIN (metadata);

-- ============================================================
-- Indexes — promoted columns on mcap_files / deliveries
-- ============================================================
CREATE UNIQUE INDEX uq_mcap_files_hash_md5
    ON mcap_files (raw_hash_md5) WHERE raw_hash_md5 IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX idx_mcap_files_tenant_project
    ON mcap_files (tenant_id, project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_mcap_files_ingest_state
    ON mcap_files (ingest_state) WHERE is_deleted = FALSE;
CREATE INDEX idx_mcap_files_metadata_gin
    ON mcap_files USING GIN (metadata);

CREATE INDEX idx_deliveries_tenant_project
    ON deliveries (tenant_id, project_id) WHERE is_deleted = FALSE;

-- ============================================================
-- Indexes — projection / event tables
-- ============================================================
CREATE INDEX idx_asset_tags_key_value_str
    ON asset_tags (tag_key, tag_value, asset_id);
CREATE INDEX idx_asset_tags_key_value_num
    ON asset_tags (tag_key, tag_value_num, asset_id) WHERE tag_type = 'number';
CREATE INDEX idx_asset_tags_key_value_bool
    ON asset_tags (tag_key, tag_value_bool, asset_id) WHERE tag_type = 'bool';
CREATE INDEX idx_asset_tags_tenant_project
    ON asset_tags (tenant_id, project_id);

CREATE INDEX idx_asset_algo_latest_algo_status
    ON asset_algo_latest (algo_name, status);
CREATE INDEX idx_asset_algo_latest_algo_version
    ON asset_algo_latest (algo_name, algo_version);
CREATE INDEX idx_asset_algo_latest_run
    ON asset_algo_latest (run_id) WHERE run_id IS NOT NULL;
CREATE INDEX idx_asset_algo_latest_updated
    ON asset_algo_latest (updated_at DESC);

CREATE INDEX idx_asset_events_type_time
    ON asset_events (event_type, occurred_at DESC);
CREATE INDEX idx_asset_events_asset
    ON asset_events (asset_id, occurred_at DESC) WHERE asset_id IS NOT NULL;
CREATE INDEX idx_asset_events_run
    ON asset_events (run_id) WHERE run_id IS NOT NULL;
CREATE INDEX idx_asset_events_publish_pending
    ON asset_events (publish_state, event_seq) WHERE publish_state = 'pending';
CREATE INDEX idx_asset_events_tenant_project
    ON asset_events (tenant_id, project_id, occurred_at DESC);

CREATE INDEX idx_asset_relations_child
    ON asset_relations (child_asset_id, relation_type);

-- ============================================================
-- Foreign Keys
-- ============================================================
ALTER TABLE delivery_items
    ADD CONSTRAINT fk_delivery_items_delivery
    FOREIGN KEY (delivery_id) REFERENCES deliveries (delivery_id);

ALTER TABLE delivery_items
    ADD CONSTRAINT fk_delivery_items_asset
    FOREIGN KEY (asset_id) REFERENCES assets (asset_id);

ALTER TABLE asset_algo_events
    ADD CONSTRAINT fk_algo_events_asset
    FOREIGN KEY (asset_id) REFERENCES assets (asset_id);

-- ============================================================
-- Triggers — set_updated_at on projection tables
-- ============================================================
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_asset_tags_updated_at
    BEFORE UPDATE ON asset_tags
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_asset_algo_latest_updated_at
    BEFORE UPDATE ON asset_algo_latest
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

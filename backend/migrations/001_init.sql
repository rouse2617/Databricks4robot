-- 001_init.sql — Full DDL for data4cyber PostgreSQL schema.
-- Executed automatically on first docker-compose up via /docker-entrypoint-initdb.d mount.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================
-- Table 1: assets
-- ============================================================
CREATE TABLE assets (
    asset_id            UUID            PRIMARY KEY,
    mcap_file_id        UUID            NOT NULL,
    start_timestamp_ns  BIGINT          NOT NULL,
    end_timestamp_ns    BIGINT          NOT NULL,
    status              VARCHAR(32)     NOT NULL DEFAULT 'approved',
    is_deleted          BOOLEAN         DEFAULT FALSE,
    segment_locator     CHAR(40),
    cf_meta             JSONB           DEFAULT '{}',
    cf_algo             JSONB           DEFAULT '{}',
    cf_tag              JSONB           DEFAULT '{}',
    cf_files            JSONB           DEFAULT '{}',
    created_at          TIMESTAMPTZ     NOT NULL,
    updated_at          TIMESTAMPTZ     NOT NULL,
    version             BIGINT          DEFAULT 1
);

-- ============================================================
-- Table 2: mcap_files
-- ============================================================
CREATE TABLE mcap_files (
    mcap_file_id    UUID            PRIMARY KEY,
    raw_hash_md5    VARCHAR(32),
    is_deleted      BOOLEAN         DEFAULT FALSE,
    cf_meta         JSONB           DEFAULT '{}',
    cf_process      JSONB           DEFAULT '{}',
    created_at      TIMESTAMPTZ     NOT NULL,
    updated_at      TIMESTAMPTZ     NOT NULL,
    version         BIGINT          DEFAULT 1
);

-- ============================================================
-- Table 3: deliveries
-- ============================================================
CREATE TABLE deliveries (
    delivery_id     UUID            PRIMARY KEY,
    customer_id     VARCHAR(64)     NOT NULL,
    status          VARCHAR(16)     NOT NULL DEFAULT 'pending',
    delivered_at    TIMESTAMPTZ,
    is_deleted      BOOLEAN         DEFAULT FALSE,
    cf_meta         JSONB           DEFAULT '{}',
    created_at      TIMESTAMPTZ     NOT NULL,
    updated_at      TIMESTAMPTZ     NOT NULL,
    version         BIGINT          DEFAULT 1
);

-- ============================================================
-- Table 4: delivery_items (junction table)
-- ============================================================
CREATE TABLE delivery_items (
    delivery_id     UUID            NOT NULL,
    asset_id        UUID            NOT NULL,
    created_at      TIMESTAMPTZ     DEFAULT now(),
    PRIMARY KEY (delivery_id, asset_id)
);

-- ============================================================
-- Table 5: asset_algo_events (audit log)
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
-- Table 6: idempotency_keys
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
-- Indexes
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

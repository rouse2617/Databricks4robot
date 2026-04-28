-- 008_schema_evolution_tables.sql — Create new projection/event/relation tables.
-- Phase A (DDL only): asset_tags, asset_algo_latest, asset_events,
-- outbox_sink_cursors, asset_relations.
-- All statements are idempotent (CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS).

-- ============================================================
-- Table 1: asset_tags — tag projection (PK: asset_id, tag_key)
-- ============================================================
CREATE TABLE IF NOT EXISTS asset_tags (
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

-- asset_tags indexes (4)
CREATE INDEX IF NOT EXISTS idx_asset_tags_key_value_str
    ON asset_tags (tag_key, tag_value, asset_id);
CREATE INDEX IF NOT EXISTS idx_asset_tags_key_value_num
    ON asset_tags (tag_key, tag_value_num, asset_id) WHERE tag_type = 'number';
CREATE INDEX IF NOT EXISTS idx_asset_tags_key_value_bool
    ON asset_tags (tag_key, tag_value_bool, asset_id) WHERE tag_type = 'bool';
CREATE INDEX IF NOT EXISTS idx_asset_tags_tenant_project
    ON asset_tags (tenant_id, project_id);

-- ============================================================
-- Table 2: asset_algo_latest — algo projection (PK: asset_id, algo_name)
-- ============================================================
CREATE TABLE IF NOT EXISTS asset_algo_latest (
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

-- asset_algo_latest indexes (4)
CREATE INDEX IF NOT EXISTS idx_asset_algo_latest_algo_status
    ON asset_algo_latest (algo_name, status);
CREATE INDEX IF NOT EXISTS idx_asset_algo_latest_algo_version
    ON asset_algo_latest (algo_name, algo_version);
CREATE INDEX IF NOT EXISTS idx_asset_algo_latest_run
    ON asset_algo_latest (run_id) WHERE run_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_asset_algo_latest_updated
    ON asset_algo_latest (updated_at DESC);

-- ============================================================
-- Table 3: asset_events — unified event/outbox (PK: event_id)
-- ============================================================
CREATE TABLE IF NOT EXISTS asset_events (
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

-- asset_events indexes (5)
CREATE INDEX IF NOT EXISTS idx_asset_events_type_time
    ON asset_events (event_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_asset_events_asset
    ON asset_events (asset_id, occurred_at DESC) WHERE asset_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_asset_events_run
    ON asset_events (run_id) WHERE run_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_asset_events_publish_pending
    ON asset_events (publish_state, event_seq) WHERE publish_state = 'pending';
CREATE INDEX IF NOT EXISTS idx_asset_events_tenant_project
    ON asset_events (tenant_id, project_id, occurred_at DESC);

-- ============================================================
-- Table 4: outbox_sink_cursors — sink watermark tracking (PK: sink_name)
-- ============================================================
CREATE TABLE IF NOT EXISTS outbox_sink_cursors (
    sink_name           TEXT         PRIMARY KEY,
    last_published_seq  BIGINT       NOT NULL DEFAULT 0,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ============================================================
-- Table 5: asset_relations — complex lineage (PK: parent, child, type)
-- ============================================================
CREATE TABLE IF NOT EXISTS asset_relations (
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

-- asset_relations indexes (1)
CREATE INDEX IF NOT EXISTS idx_asset_relations_child
    ON asset_relations (child_asset_id, relation_type);

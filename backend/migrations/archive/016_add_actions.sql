-- 016_add_actions.sql — seg 内时间分段标注 (mcap → seg → action 第三层)
-- See docs/review/data-platform-design.md §5.2.15 / docs/review/schema-reference.md#actions
--
-- 约定：
--   · action 永远挂在 asset_type='segment' 的 seg 上 (FK ON DELETE RESTRICT)
--   · action 不参与 lifecycle_state，不进 deliveries，不出现在资产列表 API
--   · 写入同事务追加 asset_events (action_upserted / action_deleted)
--   · start_ns / end_ns 与 assets.start_timestamp_ns / end_timestamp_ns 同口径

CREATE TABLE IF NOT EXISTS actions (
    action_id       TEXT                 PRIMARY KEY,
    -- Keep plain text for compatibility with local DB snapshots that use
    -- different asset_id physical types.
    asset_id        TEXT                 NOT NULL,

    start_ns        BIGINT               NOT NULL,
    end_ns          BIGINT               NOT NULL,
    action_index    INT,

    primary_label   TEXT,
    labels          TEXT[]               NOT NULL DEFAULT '{}',
    description     TEXT,
    attrs           JSONB                NOT NULL DEFAULT '{}'::jsonb,

    source_type     TEXT                 NOT NULL DEFAULT 'human',
    source_name     TEXT,
    source_version  TEXT,
    run_id          TEXT,
    confidence      DOUBLE PRECISION,
    external_id     TEXT,

    tenant_id       TEXT,
    project_id      TEXT,

    is_deleted      BOOLEAN              NOT NULL DEFAULT FALSE,
    version         BIGINT               NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ          NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ          NOT NULL DEFAULT now(),

    CONSTRAINT actions_id_chk CHECK (action_id ~ '^[0-9A-Za-z]{8}$'),
    CONSTRAINT actions_range_chk CHECK (end_ns >= start_ns)
);

CREATE INDEX IF NOT EXISTS idx_actions_asset_start
    ON actions (asset_id, start_ns) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_actions_primary_label
    ON actions (primary_label) WHERE is_deleted = FALSE AND primary_label IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_actions_labels_gin
    ON actions USING GIN (labels) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_actions_run_id
    ON actions (source_name, run_id) WHERE is_deleted = FALSE AND run_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_actions_external
    ON actions (asset_id, source_name, external_id) WHERE external_id IS NOT NULL AND is_deleted = FALSE;

COMMENT ON TABLE actions IS
    'seg 内时间分段标注 (mcap → seg → action 第三层)。docs/review/data-platform-design.md §5.2.15。';

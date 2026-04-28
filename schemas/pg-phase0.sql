-- =====================================================================
-- Databricks4robot · PostgreSQL Reference DDL
--
-- 这个文件是 docs/sql.md 描述的目标 schema 的 DDL 镜像，作为：
--   1) 新环境从零拉起时的参考脚本（idempotent，可重复执行）
--   2) docs/sql.md 字段表的"机器可读"版本
--
-- 它 **不是** 生产迁移脚本；生产/CI 环境通过 backend/migrations/00X_*.sql
-- 顺序执行。两者保持字段集合一致，但本文件不保证记录历史变更顺序。
--
-- 命名与 docs/sql.md 第 0/4 节对齐：
--   · Phase 0  (当前已实现)：mcap_files / assets / deliveries / delivery_items
--                            / asset_algo_events / idempotency_keys
--                            （含遗留 cf_meta / cf_algo / cf_tag / cf_files / cf_process）
--   · Phase 1  (近期落地)  ：assets/mcap_files 提升高频列、新增投影/事件表
--                            asset_tags / asset_algo_latest / asset_events / asset_relations
--   · Phase 2+ (长期目标)  ：datasets / dataset_snapshots / training_runs
--                            catalog_objects / catalog_object_versions
--
-- 设计原则（详见 docs/sql.md §3）：
--   · 高频过滤字段 = 普通列；JSONB 仅作低频扩展区
--   · tag / algo 当前态走投影表，历史走 asset_events + Iceberg
--   · 外部数据对象用 Catalog 引用 (catalog_name + namespace + object_name + version_ref)
--   · tenant_id / project_id 在投影表/事件表上冗余，便于将来 RLS / 分区
-- =====================================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- =====================================================================
-- 1. mcap_files —— 原始 MCAP 文件当前态
--    docs/sql.md §4.1
-- =====================================================================
CREATE TABLE IF NOT EXISTS mcap_files (
    mcap_file_id        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 内容标识
    raw_hash_md5        TEXT,
    raw_hash_sha256     TEXT,
    mcap_uri            TEXT         NOT NULL,
    size_bytes          BIGINT,

    -- 时间 / 结构摘要
    file_duration_ms    BIGINT,
    start_timestamp_ns  BIGINT,
    end_timestamp_ns    BIGINT,
    channel_count       INT,
    chunk_count         INT,

    -- 处理与采集 provenance
    ingest_state        TEXT         NOT NULL DEFAULT 'pending',
    vendor_id           TEXT,
    collector_id        TEXT,
    task_id             TEXT,
    device_id           TEXT,
    camera_model        TEXT,
    data_source         TEXT,
    location_id         TEXT,
    scene_id            TEXT,
    environment_id      TEXT,
    collection_method   TEXT,

    -- 多租户 / 生命周期 / 治理
    tenant_id           TEXT,
    project_id          TEXT,
    owner               TEXT,
    retention_tier      TEXT,
    expire_at           TIMESTAMPTZ,

    -- 扩展区
    metadata            JSONB        NOT NULL DEFAULT '{}'::jsonb,
    process_state       JSONB        NOT NULL DEFAULT '{}'::jsonb,

    -- Phase 0 遗留：当前代码仍在读写 cf_meta / cf_process。
    -- Phase 1 backfill 完成后，可在 backend/migrations 里 DROP。
    cf_meta             JSONB        NOT NULL DEFAULT '{}'::jsonb,
    cf_process          JSONB        NOT NULL DEFAULT '{}'::jsonb,

    is_deleted          BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    version             BIGINT       NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_mcap_files_hash_md5
    ON mcap_files (raw_hash_md5) WHERE raw_hash_md5 IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_mcap_files_tenant_project
    ON mcap_files (tenant_id, project_id) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_mcap_files_ingest_state
    ON mcap_files (ingest_state) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_mcap_files_created
    ON mcap_files (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mcap_files_metadata_gin
    ON mcap_files USING GIN (metadata);

COMMENT ON TABLE mcap_files IS
    '原始 MCAP 文件当前态。docs/sql.md §4.1。';

-- =====================================================================
-- 2. assets —— 资产当前态 (segment / clip / frame_set / derived_asset)
--    docs/sql.md §4.2
-- =====================================================================
CREATE TABLE IF NOT EXISTS assets (
    asset_id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    mcap_file_id            UUID         NOT NULL REFERENCES mcap_files(mcap_file_id),

    -- 类型 / 资源 / 缩略
    asset_type              TEXT         NOT NULL DEFAULT 'segment',
    storage_uri             TEXT,
    thumb_uri               TEXT,
    segment_locator         TEXT,

    -- 资产血缘 (普通一父多子)
    parent_asset_id         UUID,
    root_asset_id           UUID,
    asset_level             INT          NOT NULL DEFAULT 0,
    split_method            TEXT,
    split_algo_name         TEXT,
    split_algo_version      TEXT,
    split_run_id            TEXT,
    split_reason            TEXT,
    segment_index           INT,
    parent_start_offset_ms  BIGINT,
    parent_end_offset_ms    BIGINT,

    -- 时间
    start_timestamp_ns      BIGINT       NOT NULL,
    end_timestamp_ns        BIGINT       NOT NULL,
    duration_ms             BIGINT,

    -- 生命周期 / 状态
    -- lifecycle_state 是资产状态唯一权威字段：
    --   created / processing / ready / rejected / delivered / archived / superseded
    -- status 是 Phase 0 遗留列，代码暂时还在读，Phase 1 切到 lifecycle_state 后下线。
    lifecycle_state         TEXT         NOT NULL DEFAULT 'created',
    status                  TEXT         NOT NULL DEFAULT 'approved',

    -- 多租户 / 权限 / 治理
    tenant_id               TEXT,
    project_id              TEXT,
    owner                   TEXT,
    reviewer                TEXT,
    retention_tier          TEXT,
    expire_at               TIMESTAMPTZ,

    -- 交付汇总（冗余，权威在 deliveries / delivery_items）
    last_delivered_at       TIMESTAMPTZ,
    last_delivered_to       TEXT,
    delivery_count          INT          NOT NULL DEFAULT 0,

    -- 扩展区
    metadata                JSONB        NOT NULL DEFAULT '{}'::jsonb,
    files                   JSONB        NOT NULL DEFAULT '{}'::jsonb,

    -- Phase 0 遗留：当前代码仍读写 cf_meta / cf_algo / cf_tag / cf_files。
    -- Phase 1 双写期：cf_tag <-> asset_tags、cf_algo <-> asset_algo_latest。
    -- backfill 完成后，由迁移脚本 DROP 这些列。
    cf_meta                 JSONB        NOT NULL DEFAULT '{}'::jsonb,
    cf_algo                 JSONB        NOT NULL DEFAULT '{}'::jsonb,
    cf_tag                  JSONB        NOT NULL DEFAULT '{}'::jsonb,
    cf_files                JSONB        NOT NULL DEFAULT '{}'::jsonb,

    is_deleted              BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    version                 BIGINT       NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_assets_mcap_file
    ON assets (mcap_file_id, start_timestamp_ns, asset_id) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_assets_lifecycle
    ON assets (lifecycle_state) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_assets_status
    ON assets (status) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_assets_asset_type
    ON assets (asset_type) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_assets_tenant_project
    ON assets (tenant_id, project_id) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_assets_parent
    ON assets (parent_asset_id) WHERE parent_asset_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_assets_root
    ON assets (root_asset_id) WHERE root_asset_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_assets_created
    ON assets (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_assets_segment_locator
    ON assets (segment_locator) WHERE segment_locator IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_assets_metadata_gin
    ON assets USING GIN (metadata);
-- Phase 0 遗留 GIN 索引（与 cf_* 列同生命周期，Phase 1 退役时一起删）：
CREATE INDEX IF NOT EXISTS idx_assets_cf_meta_gin ON assets USING GIN (cf_meta);
CREATE INDEX IF NOT EXISTS idx_assets_cf_algo_gin ON assets USING GIN (cf_algo);
CREATE INDEX IF NOT EXISTS idx_assets_cf_tag_gin  ON assets USING GIN (cf_tag);
CREATE INDEX IF NOT EXISTS idx_assets_cf_files_gin ON assets USING GIN (cf_files);

COMMENT ON TABLE assets IS
    '资产当前态。docs/sql.md §4.2。行业相关 facet (city/weather/...) 入 asset_tags，不进本表。';

-- =====================================================================
-- 3. asset_relations —— 复杂血缘（多父 / 融合 / 拼接 / 派生）
--    docs/sql.md §4.2.1
-- =====================================================================
CREATE TABLE IF NOT EXISTS asset_relations (
    parent_asset_id         UUID         NOT NULL REFERENCES assets(asset_id),
    child_asset_id          UUID         NOT NULL REFERENCES assets(asset_id),
    relation_type           TEXT         NOT NULL,    -- split_from / derived_from / merged_from / sampled_from
    method                  TEXT,
    algo_name               TEXT,
    algo_version            TEXT,
    run_id                  TEXT,
    parent_start_offset_ms  BIGINT,
    parent_end_offset_ms    BIGINT,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (parent_asset_id, child_asset_id, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_asset_relations_child
    ON asset_relations (child_asset_id, relation_type);

COMMENT ON TABLE asset_relations IS
    '资产复杂血缘。一父多子可只用 assets.parent_asset_id；多父/融合/采样在此表表达。';

-- =====================================================================
-- 4. asset_tags —— 资产 tag 当前态投影
--    docs/sql.md §4.3
-- =====================================================================
CREATE TABLE IF NOT EXISTS asset_tags (
    asset_id        UUID                 NOT NULL REFERENCES assets(asset_id),
    tag_key         TEXT                 NOT NULL,
    tag_value       TEXT                 NOT NULL,
    tag_value_num   DOUBLE PRECISION,
    tag_value_bool  BOOLEAN,
    tag_type        TEXT                 NOT NULL DEFAULT 'string',  -- string/number/bool/enum
    source_type     TEXT                 NOT NULL DEFAULT 'human',   -- human/algo/rule/system
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

CREATE INDEX IF NOT EXISTS idx_asset_tags_key_value_str
    ON asset_tags (tag_key, tag_value, asset_id);
CREATE INDEX IF NOT EXISTS idx_asset_tags_key_value_num
    ON asset_tags (tag_key, tag_value_num, asset_id) WHERE tag_type = 'number';
CREATE INDEX IF NOT EXISTS idx_asset_tags_key_value_bool
    ON asset_tags (tag_key, tag_value_bool, asset_id) WHERE tag_type = 'bool';
CREATE INDEX IF NOT EXISTS idx_asset_tags_tenant_project
    ON asset_tags (tenant_id, project_id);

COMMENT ON TABLE asset_tags IS
    '资产 tag 当前态投影；ES asset 文档 tags 字段来源；asset 软删后不级联，靠 JOIN assets 过滤。';

-- =====================================================================
-- 5. asset_algo_latest —— 资产算法最新状态投影
--    docs/sql.md §4.4
-- =====================================================================
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

CREATE INDEX IF NOT EXISTS idx_asset_algo_latest_algo_status
    ON asset_algo_latest (algo_name, status);
CREATE INDEX IF NOT EXISTS idx_asset_algo_latest_algo_version
    ON asset_algo_latest (algo_name, algo_version);
CREATE INDEX IF NOT EXISTS idx_asset_algo_latest_run
    ON asset_algo_latest (run_id) WHERE run_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_asset_algo_latest_updated
    ON asset_algo_latest (updated_at DESC);

COMMENT ON TABLE asset_algo_latest IS
    '每个 (asset, algo) 最新一行；列表/筛选/ES 文档算法字段来源；完整生命周期写 asset_events。';

-- =====================================================================
-- 6. asset_events —— 统一业务事件表（瘦身版 outbox）
--    docs/sql.md §4.5
-- =====================================================================
CREATE TABLE IF NOT EXISTS asset_events (
    event_id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    -- 单调序号：所有 outbox / CDC / replay 消费者必须按 event_seq 严格递增推进
    -- watermark，不能用 occurred_at（并发写入下时间戳会冲突 / 倒序）。
    event_seq               BIGSERIAL    NOT NULL UNIQUE,
    event_type              TEXT         NOT NULL,
    -- payload schema 版本：每种 event_type 对应 schemas/events/<event_type>.v<n>.json
    payload_schema_version  TEXT         NOT NULL DEFAULT 'v1',
    asset_id                UUID         REFERENCES assets(asset_id),
    mcap_file_id            UUID         REFERENCES mcap_files(mcap_file_id),
    tenant_id               TEXT,
    project_id              TEXT,
    event_source            TEXT         NOT NULL,   -- backend / worker / dagster / spark / daft / system
    actor_type              TEXT,                    -- user / service / algo / system
    actor_id                TEXT,
    request_id              TEXT,
    idempotency_key         TEXT,
    run_id                  TEXT,
    occurred_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    publish_state           TEXT         NOT NULL DEFAULT 'pending',  -- pending / published / failed
    published_at            TIMESTAMPTZ,
    event_payload           JSONB        NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_asset_events_type_time
    ON asset_events (event_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_asset_events_asset
    ON asset_events (asset_id, occurred_at DESC) WHERE asset_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_asset_events_run
    ON asset_events (run_id) WHERE run_id IS NOT NULL;
-- outbox worker 走这个索引按 event_seq 推进；published 行不在索引里，避免膨胀。
CREATE INDEX IF NOT EXISTS idx_asset_events_publish_pending
    ON asset_events (publish_state, event_seq) WHERE publish_state = 'pending';
CREATE INDEX IF NOT EXISTS idx_asset_events_tenant_project
    ON asset_events (tenant_id, project_id, occurred_at DESC);

COMMENT ON TABLE asset_events IS
    '统一业务事件 / 审计 / outbox。类型详情统一进 event_payload，不摊为大量 nullable 列；消费按 event_seq 推进。';
COMMENT ON COLUMN asset_events.event_seq IS
    '单调递增序号，UNIQUE。outbox / CDC / replay 消费者必须按此字段推进 watermark，不能用 occurred_at。';
COMMENT ON COLUMN asset_events.payload_schema_version IS
    'event_payload 的 schema 版本，对应 schemas/events/<event_type>.v<n>.json；新增字段 minor、破坏性变更 major。';

-- =====================================================================
-- 7. asset_algo_events —— 算法状态变更审计（Phase 0 遗留）
--    Phase 1 后被 asset_events (event_type IN (algo_started, algo_finished, algo_failed))
--    取代；过渡期保留写入。
-- =====================================================================
CREATE TABLE IF NOT EXISTS asset_algo_events (
    event_id        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id        UUID         NOT NULL REFERENCES assets(asset_id),
    algo_key        TEXT         NOT NULL,
    prev_status     TEXT,
    new_status      TEXT         NOT NULL,
    run_id          TEXT,
    reason          TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_algo_events_asset
    ON asset_algo_events (asset_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_algo_events_algo_status
    ON asset_algo_events (algo_key, new_status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_algo_events_run_id
    ON asset_algo_events (run_id) WHERE run_id IS NOT NULL;

COMMENT ON TABLE asset_algo_events IS
    'Phase 0 遗留：算法状态变更审计；Phase 1 后被 asset_events 取代。';

-- =====================================================================
-- 8. deliveries —— 客户交付批次当前态
--    docs/sql.md §4.6
-- =====================================================================
CREATE TABLE IF NOT EXISTS deliveries (
    delivery_id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             TEXT,
    project_id            TEXT,
    customer_id           TEXT         NOT NULL,
    contract_id           TEXT,
    delivery_type         TEXT         NOT NULL DEFAULT 'asset_set',
    status                TEXT         NOT NULL DEFAULT 'pending',
    requested_by          TEXT,
    approved_by           TEXT,
    delivered_by          TEXT,
    manifest_uri          TEXT,
    replay_manifest_uri   TEXT,
    item_count            BIGINT       NOT NULL DEFAULT 0,
    total_size_bytes      BIGINT,
    delivered_at          TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    metadata              JSONB        NOT NULL DEFAULT '{}'::jsonb,
    -- Phase 0 遗留
    cf_meta               JSONB        NOT NULL DEFAULT '{}'::jsonb,
    is_deleted            BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    version               BIGINT       NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_deliveries_customer
    ON deliveries (customer_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_deliveries_status
    ON deliveries (status) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_deliveries_completed
    ON deliveries (completed_at DESC) WHERE completed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_deliveries_tenant_project
    ON deliveries (tenant_id, project_id) WHERE is_deleted = FALSE;

COMMENT ON TABLE deliveries IS '客户交付批次当前态。docs/sql.md §4.6。';

-- =====================================================================
-- 9. delivery_items —— Delivery ↔ Asset M:N 明细
--    docs/sql.md §4.7
-- =====================================================================
CREATE TABLE IF NOT EXISTS delivery_items (
    delivery_id    UUID         NOT NULL REFERENCES deliveries(delivery_id) ON DELETE CASCADE,
    asset_id       UUID         NOT NULL REFERENCES assets(asset_id),
    asset_version  BIGINT,
    item_state     TEXT         NOT NULL DEFAULT 'pending',
    checksum       TEXT,
    export_uri     TEXT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (delivery_id, asset_id)
);

CREATE INDEX IF NOT EXISTS idx_delivery_items_by_asset
    ON delivery_items (asset_id, delivery_id);

-- =====================================================================
-- 10. datasets / dataset_snapshots —— 数据集与快照
--     docs/sql.md §4.8
-- =====================================================================
CREATE TABLE IF NOT EXISTS datasets (
    dataset_id     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT         NOT NULL,
    description    TEXT,
    owner          TEXT,
    tenant_id      TEXT,
    project_id     TEXT,
    dataset_type   TEXT         NOT NULL,                          -- training/eval/replay/delivery/experiment
    status         TEXT         NOT NULL DEFAULT 'active',         -- active/archived/deleted
    created_by     TEXT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_datasets_name_per_project
    ON datasets (COALESCE(tenant_id, ''), COALESCE(project_id, ''), name);
CREATE INDEX IF NOT EXISTS idx_datasets_owner
    ON datasets (owner) WHERE status <> 'deleted';

COMMENT ON TABLE datasets IS '数据集定义父表。明细快照在 dataset_snapshots。';

CREATE TABLE IF NOT EXISTS dataset_snapshots (
    dataset_id                  UUID         NOT NULL REFERENCES datasets(dataset_id),
    snapshot_id                 UUID         NOT NULL DEFAULT gen_random_uuid(),
    snapshot_version            TEXT,
    created_by                  TEXT,
    query_spec                  JSONB        NOT NULL DEFAULT '{}'::jsonb,
    source_query_hash           TEXT,
    source_catalog_name         TEXT,
    source_namespace            TEXT,
    source_object_name          TEXT,
    source_object_version_ref   TEXT,
    manifest_uri                TEXT         NOT NULL,
    item_count                  BIGINT       NOT NULL DEFAULT 0,
    status                      TEXT         NOT NULL DEFAULT 'building', -- building/ready/failed/archived
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (dataset_id, snapshot_id)
);

CREATE INDEX IF NOT EXISTS idx_dataset_snapshots_status
    ON dataset_snapshots (status);
CREATE INDEX IF NOT EXISTS idx_dataset_snapshots_created
    ON dataset_snapshots (dataset_id, created_at DESC);

COMMENT ON TABLE dataset_snapshots IS
    '数据集快照元信息；明细仅在 Iceberg；引用源用 catalog_name + namespace + object_name + version_ref。';

-- =====================================================================
-- 11. training_runs —— 训练任务记录（自包含数据引用）
--     docs/sql.md §4.9
-- =====================================================================
CREATE TABLE IF NOT EXISTS training_runs (
    training_run_id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                TEXT,
    project_id               TEXT,
    dataset_id               UUID         NOT NULL,
    snapshot_id              UUID         NOT NULL,
    model_name               TEXT         NOT NULL,
    model_version            TEXT,
    algo_name                TEXT,
    code_version             TEXT,
    config_uri               TEXT,
    -- 自包含的数据对象引用，不强依赖可变 FK
    data_manifest_uri        TEXT,
    data_catalog_name        TEXT,
    data_namespace           TEXT,
    data_object_name         TEXT,
    data_object_version_ref  TEXT,
    status                   TEXT         NOT NULL DEFAULT 'pending',
    metrics                  JSONB        NOT NULL DEFAULT '{}'::jsonb,
    artifact_uri             TEXT,
    started_at               TIMESTAMPTZ,
    finished_at              TIMESTAMPTZ,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    -- 软引用（不外键）：dataset_snapshots(dataset_id, snapshot_id)
    -- 训练记录是审计证据，不允许被快照表的物理 FK 级联影响
    CONSTRAINT chk_training_runs_status CHECK (
        status IN ('pending','running','succeeded','failed','cancelled')
    )
);

CREATE INDEX IF NOT EXISTS idx_training_runs_dataset
    ON training_runs (dataset_id, snapshot_id);
CREATE INDEX IF NOT EXISTS idx_training_runs_model
    ON training_runs (model_name, model_version);
CREATE INDEX IF NOT EXISTS idx_training_runs_status_time
    ON training_runs (status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_training_runs_tenant_project
    ON training_runs (tenant_id, project_id);

COMMENT ON TABLE training_runs IS
    '训练任务记录。自包含 catalog 引用，不强依赖 dataset_snapshots 的物理 FK，便于审计追溯。';

-- =====================================================================
-- 12. catalog_objects / catalog_object_versions —— Platform Catalog 抽象
--     docs/sql.md §4.11
-- =====================================================================
CREATE TABLE IF NOT EXISTS catalog_objects (
    object_id      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_name   TEXT         NOT NULL,                  -- online / lakehouse / search / multimodal
    namespace      TEXT         NOT NULL,                  -- public / robot.bronze / robot.gold
    object_name    TEXT         NOT NULL,
    object_type    TEXT         NOT NULL,                  -- postgres_table / iceberg_table / elasticsearch_index / lance_dataset / object_prefix / manifest
    provider       TEXT         NOT NULL,                  -- iceberg_rest / postgres / elasticsearch / lance / polaris / gravitino / glue / dlf
    format         TEXT         NOT NULL,                  -- postgres / iceberg / elasticsearch / lance / parquet / webdataset / manifest
    storage_uri    TEXT,
    external_ref   TEXT,
    owner          TEXT,
    tenant_id      TEXT,
    project_id     TEXT,
    description    TEXT,
    tags           JSONB        NOT NULL DEFAULT '{}'::jsonb,
    properties     JSONB        NOT NULL DEFAULT '{}'::jsonb,
    status         TEXT         NOT NULL DEFAULT 'active', -- active / deprecated / deleted
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_catalog_objects_identity
    ON catalog_objects (catalog_name, namespace, object_name, object_type);
CREATE INDEX IF NOT EXISTS idx_catalog_objects_type
    ON catalog_objects (object_type, status);
CREATE INDEX IF NOT EXISTS idx_catalog_objects_tenant_project
    ON catalog_objects (tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_catalog_objects_tags_gin
    ON catalog_objects USING GIN (tags);

COMMENT ON TABLE catalog_objects IS
    '中立 Catalog 注册：湖表 / PG 表 / ES 索引 / Lance dataset 等。docs/sql.md §1.2 / §4.11。';

CREATE TABLE IF NOT EXISTS catalog_object_versions (
    object_version_id  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    object_id          UUID         NOT NULL REFERENCES catalog_objects(object_id) ON DELETE CASCADE,
    version_ref        TEXT         NOT NULL,
    version_type       TEXT         NOT NULL,    -- iceberg_snapshot / lance_version / manifest_version / index_generation / schema_version
    schema_ref         TEXT,
    manifest_uri       TEXT,
    row_count          BIGINT,
    size_bytes         BIGINT,
    checksum           TEXT,
    created_by         TEXT,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    properties         JSONB        NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_catalog_versions_object_time
    ON catalog_object_versions (object_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_catalog_versions_object_ref
    ON catalog_object_versions (object_id, version_ref);

COMMENT ON TABLE catalog_object_versions IS
    'Iceberg snapshot_id / Lance version / ES index generation 等版本引用统一进 version_ref。';

-- =====================================================================
-- 13. idempotency_keys —— API 幂等保护
--     docs/sql.md §4.12
-- =====================================================================
-- 主键 (scope, idem_key) 与 docs/sql.md §4.12 一致。
-- expires_at 由独立 lifecycle job 定期清理（response_json 可能含 PII，禁止无限保留）。
CREATE TABLE IF NOT EXISTS idempotency_keys (
    scope          TEXT         NOT NULL,
    idem_key       TEXT         NOT NULL,
    resource_type  TEXT,
    resource_id    TEXT,
    request_hash   TEXT,
    response_json  JSONB        NOT NULL DEFAULT '{}'::jsonb,
    status_code    INT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at     TIMESTAMPTZ,
    PRIMARY KEY (scope, idem_key)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires
    ON idempotency_keys (expires_at) WHERE expires_at IS NOT NULL;

-- =====================================================================
-- 14. 触发器：updated_at 自动维护
-- =====================================================================
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOREACH tbl IN ARRAY ARRAY[
        'mcap_files', 'assets', 'asset_tags', 'asset_algo_latest',
        'deliveries', 'datasets', 'dataset_snapshots',
        'training_runs', 'catalog_objects'
    ]
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS trg_%1$s_updated_at ON %1$s', tbl);
        EXECUTE format(
            'CREATE TRIGGER trg_%1$s_updated_at
               BEFORE UPDATE ON %1$s
               FOR EACH ROW EXECUTE FUNCTION set_updated_at()', tbl);
    END LOOP;
END
$$;

-- =====================================================================
-- 15. 查询示例（注释，不执行）
-- =====================================================================
-- 1) 按 tag 过滤资产（投影表，比 cf_tag JSONB 更稳定）：
--    SELECT a.asset_id
--      FROM assets a
--      JOIN asset_tags t1 ON t1.asset_id = a.asset_id AND t1.tag_key='quality' AND t1.tag_value='good'
--      JOIN asset_tags t2 ON t2.asset_id = a.asset_id AND t2.tag_key='weather' AND t2.tag_value='rain'
--     WHERE a.is_deleted = FALSE
--     LIMIT 1000;
--
-- 2) 哪些 asset 跑过 hand_tracking 且失败：
--    SELECT asset_id FROM asset_algo_latest
--     WHERE algo_name = 'hand_tracking' AND status = 'failed'
--     LIMIT 1000;
--
-- 3) 取一段时间内未发布的事件（outbox 消费）：
--    SELECT *
--      FROM asset_events
--     WHERE publish_state = 'pending'
--     ORDER BY occurred_at
--     LIMIT 500;
--
-- 4) 训练任务实际读取的湖表与版本：
--    SELECT training_run_id, data_catalog_name, data_namespace,
--           data_object_name, data_object_version_ref, data_manifest_uri
--      FROM training_runs
--     WHERE training_run_id = $1;

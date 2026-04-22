-- =====================================================================
-- data4cyber - Phase 0 Postgres 宽表 DDL
--
-- 目标:用 Postgres JSONB 模拟 Bigtable 9 列族(8 JSONB + asset_events 独立表代替 cf:event),走通 E2E
-- 规模:< 100 万 asset(Phase 0 足够)
-- 迁移:Phase 1 切 Bigtable 时双写 → backfill → 切读 → 下线
-- =====================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- =====================================================================
-- 1. 资产主表(宽表,asset_id 主键,8 个 JSONB 模拟 Bigtable 的 8 个 CF;cf:event 在下面 asset_events 表)
-- =====================================================================
CREATE TABLE IF NOT EXISTS assets (
    asset_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant        TEXT NOT NULL,
    urn           TEXT GENERATED ALWAYS AS ('urn:grace:asset:' || asset_id::text) STORED,

    -- 8 个"伪列族"
    cf_core       JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_time       JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_tag        JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_file       JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_qa         JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_lineage    JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_algo       JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_emb        JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 元数据
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    version       BIGINT NOT NULL DEFAULT 1,    -- 乐观锁
    deleted_at    TIMESTAMPTZ                    -- soft delete
);

-- GIN 索引加速 JSONB 过滤
CREATE INDEX IF NOT EXISTS idx_assets_cf_tag   ON assets USING GIN (cf_tag);
CREATE INDEX IF NOT EXISTS idx_assets_cf_qa    ON assets USING GIN (cf_qa);
CREATE INDEX IF NOT EXISTS idx_assets_cf_file  ON assets USING GIN (cf_file);
CREATE INDEX IF NOT EXISTS idx_assets_cf_algo  ON assets USING GIN (cf_algo);

-- 常用等值 / 范围
CREATE INDEX IF NOT EXISTS idx_assets_tenant   ON assets (tenant);
CREATE INDEX IF NOT EXISTS idx_assets_created  ON assets (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_assets_not_del  ON assets (asset_id) WHERE deleted_at IS NULL;

COMMENT ON TABLE assets IS
  'Phase 0 宽表,8 个 JSONB 字段模拟 Bigtable 的 8 个 CF(cf:core/time/tag/file/qa/lineage/algo/emb);cf:event 独立为 asset_events 表。Phase 1 迁移至 Cloud Bigtable(9 CF)。';

-- =====================================================================
-- 2. 事件历史表(对应 Bigtable cf:event 的多版本,PG 里独立表)
-- =====================================================================
CREATE TABLE IF NOT EXISTS asset_events (
    id            BIGSERIAL PRIMARY KEY,
    asset_id      UUID NOT NULL REFERENCES assets(asset_id) ON DELETE CASCADE,
    event_type    TEXT NOT NULL,
    payload       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_asset_time
  ON asset_events (asset_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_type
  ON asset_events (event_type, created_at DESC);

COMMENT ON TABLE asset_events IS
  '资产事件流(对应 Bigtable cf:event 多版本)';

-- =====================================================================
-- 3. 外部 URN ↔ asset 映射表(便于反查)
-- =====================================================================
CREATE TABLE IF NOT EXISTS external_refs (
    external_urn  TEXT PRIMARY KEY,                    -- 例:urn:grace:mcap:gs://...
    asset_id      UUID NOT NULL REFERENCES assets(asset_id) ON DELETE CASCADE,
    ref_kind      TEXT NOT NULL,                       -- 'mcap' | 'dataset' | ...
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ext_refs_asset
  ON external_refs (asset_id);

-- =====================================================================
-- 4. Schema Registry(Classification / Tag / FileKind 可加载版)
--    与 schemas/fields.yaml 保持同步,服务启动时灌库
-- =====================================================================
CREATE TABLE IF NOT EXISTS schema_classifications (
    name          TEXT PRIMARY KEY,
    description   TEXT,
    mutually_exclusive BOOLEAN DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS schema_tags (
    classification TEXT NOT NULL REFERENCES schema_classifications(name),
    name           TEXT NOT NULL,
    description    TEXT,
    deprecated     BOOLEAN DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (classification, name)
);

CREATE TABLE IF NOT EXISTS schema_file_kinds (
    name          TEXT PRIMARY KEY,
    description   TEXT,
    mutable       BOOLEAN DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =====================================================================
-- 5. 乐观锁 / 状态机推进样例(service 层调用)
-- =====================================================================
--
-- 示例:把某 asset QA 置为 approved(要求 prev_state=in_review,乐观锁版本匹配)
--
-- UPDATE assets
--   SET cf_qa = cf_qa
--                || jsonb_build_object('state', 'approved',
--                                       'qa_by', $3,
--                                       'qa_at', now(),
--                                       'prev_state', cf_qa->>'state'),
--       version    = version + 1,
--       updated_at = now()
-- WHERE asset_id  = $1
--   AND version   = $2
--   AND cf_qa->>'state' = 'in_review'
--   AND deleted_at IS NULL
-- RETURNING version;
--
-- 若返回空行 → 冲突,客户端重试。

-- =====================================================================
-- 6. 常用查询样例
-- =====================================================================
--
-- 1) 按 tag 过滤:Scene.urban AND Weather.rainy
--    SELECT asset_id FROM assets
--    WHERE cf_tag @> '{"Scene.urban":1, "Weather.rainy":1}'
--      AND deleted_at IS NULL
--    LIMIT 1000;
--
-- 2) 按 QA 状态 + 时长:
--    SELECT asset_id FROM assets
--    WHERE cf_qa->>'state' = 'approved'
--      AND (cf_time->>'duration_ns')::BIGINT > 30000000000
--    ORDER BY created_at DESC LIMIT 100;
--
-- 3) 查询 asset 的事件流:
--    SELECT event_type, payload, created_by, created_at
--    FROM asset_events
--    WHERE asset_id = $1
--    ORDER BY created_at DESC
--    LIMIT 100;
--
-- 4) 按 MCAP URI 反查 asset:
--    SELECT asset_id FROM external_refs
--    WHERE external_urn = 'urn:grace:mcap:gs://...';

-- =====================================================================
-- 7. updated_at 自动维护触发器
-- =====================================================================
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_assets_updated_at ON assets;
CREATE TRIGGER trg_assets_updated_at
  BEFORE UPDATE ON assets
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();

-- =====================================================================
-- 8. 审计视图(便于管理员查看)
-- =====================================================================
CREATE OR REPLACE VIEW v_assets_summary AS
SELECT
    asset_id,
    tenant,
    cf_core->>'type'              AS asset_type,
    cf_core->>'status'            AS status,
    cf_qa->>'state'               AS qa_state,
    (cf_time->>'duration_ns')::BIGINT / 1e9 AS duration_sec,
    cf_time->>'mcap_uri'          AS mcap_uri,
    (SELECT COUNT(*) FROM jsonb_object_keys(cf_file)) AS file_count,
    (SELECT COUNT(*) FROM jsonb_object_keys(cf_tag))  AS tag_count,
    created_at,
    updated_at,
    deleted_at
FROM assets;

COMMENT ON VIEW v_assets_summary IS
  '管理视图:每 asset 一行,摘要展示各列族关键字段';

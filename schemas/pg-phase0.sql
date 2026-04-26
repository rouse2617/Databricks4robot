-- =====================================================================
-- data4cyber - Phase 0 Postgres DDL
--
-- 与 schemas/sql.md 和 column-families.yaml 完全同步:
--   · 4 张表:mcap_files(2 CF) + assets(3 CF) + deliveries(1 CF) + delivery_items(关联表)
--   · 6 个 JSONB 字段 = 6 个 Bigtable CF
--   · FK / 索引热字段从 JSONB 提升为真实列,其他字段留在 JSONB
--
-- 规模:< 100 万 asset(Phase 0 足够)
-- 迁移:Phase 1 切 Bigtable 时双写 → backfill → 切读 → 下线
-- =====================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- =====================================================================
-- 1. mcap_files - 原始 MCAP 文件级元数据(引用表)
--    对应 Bigtable 2 个 CF:cf:meta(20 字段,含 owner 占位)+ cf:process(8 字段)
-- =====================================================================
CREATE TABLE IF NOT EXISTS mcap_files (
    mcap_file_id  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 热字段从 cf:meta 提升为真实列(文件防重 / 软删过滤)
    raw_hash_md5  TEXT         NOT NULL,
    is_deleted    BOOLEAN      NOT NULL DEFAULT FALSE,

    -- 2 个伪列族
    cf_meta       JSONB        NOT NULL DEFAULT '{}'::jsonb,
    cf_process    JSONB        NOT NULL DEFAULT '{}'::jsonb,

    -- 运行期元字段
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    version       BIGINT       NOT NULL DEFAULT 1                 -- 乐观锁
);

CREATE INDEX IF NOT EXISTS idx_mcap_files_cf_meta_gin
  ON mcap_files USING GIN (cf_meta);
CREATE INDEX IF NOT EXISTS idx_mcap_files_cf_process_gin
  ON mcap_files USING GIN (cf_process);
CREATE INDEX IF NOT EXISTS idx_mcap_files_not_del
  ON mcap_files (mcap_file_id) WHERE is_deleted = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_mcap_files_hash_uniq
  ON mcap_files (raw_hash_md5) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_mcap_files_created
  ON mcap_files (created_at DESC);

COMMENT ON TABLE mcap_files IS
  'Phase 0 原始 MCAP 文件级元数据;2 个 JSONB(cf_meta/cf_process)对应 Bigtable 2 CF。';

-- =====================================================================
-- 2. assets - 资产主表(Segment 级,一行 = 一个有效视频片段)
--    对应 Bigtable 3 个 CF:cf:meta(22 字段,含 owner 占位 + 3 个交付汇总)+ cf:algo(扁平)+ cf:tag(扁平)
--
-- 交付相关字段说明(在 cf_meta 内,不单独提升):
--   · last_delivered_at   - 最近一次交付时间(NULL=从未交付)
--   · delivery_count      - 累计交付次数
--   · last_delivered_to   - 最近一次交付客户 URN
-- 这 3 个字段是**汇总快照**,权威数据在 deliveries + delivery_items。
-- 写 delivery_items 时,由 trg_delivery_items_sync_asset_summary 触发器自动刷新。
-- =====================================================================
CREATE TABLE IF NOT EXISTS assets (
    asset_id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 热字段从 cf:meta 提升为真实列:
    --   · mcap_file_id       - FK 引用完整性
    --   · start_timestamp_ns - idx_segments_by_file 复合索引要用
    --   · status             - 常用过滤
    --   · is_deleted         - 软删过滤
    -- 这些字段在 Phase 1 Bigtable 里仍在 cf:meta 中,语义不变
    mcap_file_id        UUID         NOT NULL REFERENCES mcap_files(mcap_file_id),
    start_timestamp_ns  BIGINT       NOT NULL,
    status              TEXT         NOT NULL DEFAULT 'approved',
    is_deleted          BOOLEAN      NOT NULL DEFAULT FALSE,

    -- 3 个伪列族
    cf_meta             JSONB        NOT NULL DEFAULT '{}'::jsonb,
    cf_algo             JSONB        NOT NULL DEFAULT '{}'::jsonb,
    cf_tag              JSONB        NOT NULL DEFAULT '{}'::jsonb,

    -- 运行期元字段
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    version             BIGINT       NOT NULL DEFAULT 1            -- 乐观锁
);

COMMENT ON TABLE assets IS
  'Phase 0 资产主表(Segment 级);3 个 JSONB(cf_meta/cf_algo/cf_tag)对应 Bigtable 3 CF;cf_algo 扁平键名 <algo>@<ver>:<field>。';

COMMENT ON COLUMN assets.mcap_file_id IS
  '逻辑外键(Phase 0 真实 FK,Phase 1 Bigtable 无 FK,由 service 层保证)';
COMMENT ON COLUMN assets.start_timestamp_ns IS
  '从 cf:meta 提升为真实列,用于 idx_segments_by_file 复合索引';

-- =====================================================================
-- 3. deliveries - 交付批次表(1 行 = 1 次交付事件)
--    对应 Bigtable 1 个 CF:cf:meta(16 字段)
--
-- 为什么要单独建表而不在 assets 加字段:
--   · 一个 asset 可以交付给多个客户(M:N)—— 一个字段存不下
--   · 同一个客户可能收同一 asset 多次(补发 / 重交付)—— 需要历史
--   · 要支持 "召回某批次" / "客户 A 在 Q1 收了哪些" 这类分析查询
-- 详见 docs/adr/ADR-011-delivery-tracking.md
-- =====================================================================
CREATE TABLE IF NOT EXISTS deliveries (
    delivery_id    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 热字段提升为真实列(列表页 / 索引用)
    customer_id    TEXT         NOT NULL,        -- 客户 URN, e.g. urn:grace:customer:A
    status         TEXT         NOT NULL DEFAULT 'pending',
                                -- pending / delivered / accepted / rejected / recalled
    delivered_at   TIMESTAMPTZ,                  -- NULL 表示还在 pending
    is_deleted     BOOLEAN      NOT NULL DEFAULT FALSE,

    -- 1 个伪列族:其余字段(contract_id / manifest_uri / owner / asset_count /
    -- total_size_bytes / accepted_at / rejected_at / recalled_at / rejection_reason)
    -- 全部进 cf_meta JSONB,Phase 1 直接映射到 Bigtable cf:meta
    cf_meta        JSONB        NOT NULL DEFAULT '{}'::jsonb,

    -- 运行期元字段
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    version        BIGINT       NOT NULL DEFAULT 1
);

COMMENT ON TABLE deliveries IS
  'Phase 0 交付批次表;1 行=1 次交付事件;asset 明细在 delivery_items;对应 Bigtable 1 CF。';

-- =====================================================================
-- 4. delivery_items - asset ↔ delivery 关联表(M:N)
--    Phase 0:普通关联表;Phase 1 Bigtable:拆成 idx_asset_deliveries + idx_customer_deliveries 两张索引
-- =====================================================================
CREATE TABLE IF NOT EXISTS delivery_items (
    delivery_id  UUID         NOT NULL REFERENCES deliveries(delivery_id) ON DELETE CASCADE,
    asset_id     UUID         NOT NULL REFERENCES assets(asset_id),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    PRIMARY KEY (delivery_id, asset_id)
);

COMMENT ON TABLE delivery_items IS
  'Phase 0 asset↔delivery M:N 关联表;Phase 1 Bigtable 拆成两张索引表承载双向查询。';

-- =====================================================================
-- 5. 索引
-- =====================================================================

-- 5.1 idx_segments_by_file:按 MCAP 拿所有 segment(Phase 1 = 独立 Bigtable 索引表)
CREATE INDEX IF NOT EXISTS idx_segments_by_file
  ON assets (mcap_file_id, start_timestamp_ns, asset_id)
  WHERE is_deleted = FALSE;

-- 5.2 JSONB 过滤(Phase 1 走 OpenSearch CDC;Phase 0 本地 GIN 够用)
CREATE INDEX IF NOT EXISTS idx_assets_cf_meta_gin ON assets USING GIN (cf_meta);
CREATE INDEX IF NOT EXISTS idx_assets_cf_algo_gin ON assets USING GIN (cf_algo);
CREATE INDEX IF NOT EXISTS idx_assets_cf_tag_gin  ON assets USING GIN (cf_tag);

-- 5.3 常用等值 / 范围
CREATE INDEX IF NOT EXISTS idx_assets_status
  ON assets (status) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_assets_created
  ON assets (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_assets_not_del
  ON assets (asset_id) WHERE is_deleted = FALSE;

-- 5.4 deliveries 索引
CREATE INDEX IF NOT EXISTS idx_deliveries_customer
  ON deliveries (customer_id, delivered_at DESC)
  WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_deliveries_status
  ON deliveries (status) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_deliveries_cf_meta_gin
  ON deliveries USING GIN (cf_meta);

-- 5.5 delivery_items 反向索引:asset → deliveries
-- 对应 Phase 1 Bigtable idx_asset_deliveries
CREATE INDEX IF NOT EXISTS idx_delivery_items_by_asset
  ON delivery_items (asset_id, delivery_id);

-- =====================================================================
-- 6. 乐观锁 / 状态机样例
-- =====================================================================
-- Asset 由 CommitQA 一次性落 approved / rejected,不经 pending / in_review。
-- 状态流转只发生在 rework(approved→superseded)/ archive / dispute 这几条路径。
--
-- 示例:rework 把老 asset 置为 superseded(要求 prev_status=approved + 版本匹配)
--
-- UPDATE assets
--   SET status     = 'superseded',
--       cf_meta    = cf_meta || jsonb_build_object('prev_state', status),
--       version    = version + 1,
--       updated_at = now()
-- WHERE asset_id   = $1
--   AND version    = $2           -- 客户端上一次读到的乐观锁 version
--   AND status     = 'approved'   -- 状态机前件
--   AND is_deleted = FALSE
-- RETURNING version;
--
-- 若返回空行 → 并发冲突,客户端重试。

-- =====================================================================
-- 7. 常用查询样例
-- =====================================================================
--
-- 1) 按 MCAP 文件拿所有 segment(走 idx_segments_by_file):
--    SELECT asset_id, start_timestamp_ns,
--           (cf_meta->>'end_timestamp_ns')::BIGINT AS end_ns,
--           cf_meta->>'type'  AS seg_type
--    FROM assets
--    WHERE mcap_file_id = $1 AND is_deleted = FALSE
--    ORDER BY start_timestamp_ns;
--
-- 2) 按算法状态过滤(例:sam2@1.2.0 跑失败的):
--    SELECT asset_id FROM assets
--    WHERE cf_algo->>'sam2@1.2.0:status' = 'failed'
--      AND is_deleted = FALSE
--    LIMIT 1000;
--
-- 3) 按 tag 过滤(priority=A AND quality=good):
--    SELECT asset_id FROM assets
--    WHERE cf_tag @> '{"priority":"A","quality":"good"}'
--      AND is_deleted = FALSE
--    LIMIT 1000;
--
-- 4) 按 QA 状态 + 时长筛选:
--    SELECT asset_id FROM assets
--    WHERE status = 'approved'
--      AND (cf_meta->>'duration_sec')::NUMERIC > 30
--      AND is_deleted = FALSE
--    ORDER BY created_at DESC LIMIT 100;
--
-- 5) 看某 MCAP 文件的处理状态:
--    SELECT mcap_file_id,
--           cf_process->>'hand_tracking' AS hand,
--           cf_process->>'deface'        AS deface,
--           cf_process->>'deblur'        AS deblur,
--           cf_process->>'failed_reason' AS err
--    FROM mcap_files WHERE mcap_file_id = $1;
--
-- 6) 【交付】某资产交付给过哪些客户 / 哪几次(asset → deliveries):
--    SELECT d.delivery_id, d.customer_id, d.delivered_at, d.status
--    FROM delivery_items di
--    JOIN deliveries d USING (delivery_id)
--    WHERE di.asset_id = $1 AND d.is_deleted = FALSE
--    ORDER BY d.delivered_at DESC;
--
-- 7) 【交付】某客户某时段收到的所有 asset(customer → assets):
--    SELECT di.asset_id, d.delivery_id, d.delivered_at
--    FROM deliveries d
--    JOIN delivery_items di USING (delivery_id)
--    WHERE d.customer_id = $1
--      AND d.delivered_at BETWEEN $2 AND $3
--      AND d.is_deleted = FALSE;
--
-- 8) 【交付】召回某批次(合规 / GDPR):
--    UPDATE deliveries
--      SET status = 'recalled',
--          cf_meta = cf_meta || jsonb_build_object('recalled_at', now(),
--                                                  'rejection_reason', 'gdpr_request'),
--          version = version + 1
--    WHERE delivery_id = $1 AND status != 'recalled';
--    -- 然后用 SELECT ... FROM delivery_items WHERE delivery_id=$1 拉到所有 asset 做后续下游通知
--
-- 9) 【交付】快判资产是否交付过(走 assets.cf_meta 汇总快照,免 JOIN):
--    SELECT asset_id,
--           cf_meta->>'last_delivered_at' AS last_at,
--           (cf_meta->>'delivery_count')::INT AS cnt
--    FROM assets WHERE asset_id = $1;

-- =====================================================================
-- 8. updated_at 自动维护
-- =====================================================================
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_mcap_files_updated_at ON mcap_files;
CREATE TRIGGER trg_mcap_files_updated_at
  BEFORE UPDATE ON mcap_files
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_assets_updated_at ON assets;
CREATE TRIGGER trg_assets_updated_at
  BEFORE UPDATE ON assets
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_deliveries_updated_at ON deliveries;
CREATE TRIGGER trg_deliveries_updated_at
  BEFORE UPDATE ON deliveries
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- =====================================================================
-- 9. 交付汇总回填触发器:delivery_items 写入 → assets.cf_meta 刷新
--
-- 写 delivery_items 时同事务更新 assets.cf_meta 的 3 个汇总字段,
-- 保证 Phase 0 的强一致。Phase 1 Bigtable 改为 service 层 + Outbox 最终一致。
-- =====================================================================
CREATE OR REPLACE FUNCTION sync_asset_delivery_summary() RETURNS TRIGGER AS $$
DECLARE
    v_delivered_at TIMESTAMPTZ;
    v_customer_id  TEXT;
BEGIN
    SELECT delivered_at, customer_id
      INTO v_delivered_at, v_customer_id
      FROM deliveries WHERE delivery_id = NEW.delivery_id;

    -- 只有 deliveries.status 已经到 delivered 才视为"真交付"
    IF v_delivered_at IS NULL THEN
        RETURN NEW;
    END IF;

    UPDATE assets
      SET cf_meta = cf_meta
                  || jsonb_build_object(
                        'last_delivered_at', v_delivered_at,
                        'last_delivered_to', v_customer_id,
                        'delivery_count',
                            COALESCE((cf_meta->>'delivery_count')::INT, 0) + 1),
          updated_at = now(),
          version    = version + 1
    WHERE asset_id = NEW.asset_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_delivery_items_sync_asset_summary ON delivery_items;
CREATE TRIGGER trg_delivery_items_sync_asset_summary
  AFTER INSERT ON delivery_items
  FOR EACH ROW EXECUTE FUNCTION sync_asset_delivery_summary();

-- =====================================================================
-- 10. 审计视图
-- =====================================================================
CREATE OR REPLACE VIEW v_assets_summary AS
SELECT
    a.asset_id,
    a.mcap_file_id,
    a.status,
    a.start_timestamp_ns,
    (a.cf_meta->>'end_timestamp_ns')::BIGINT           AS end_timestamp_ns,
    (a.cf_meta->>'duration_sec')::NUMERIC              AS duration_sec,
    a.cf_meta->>'type'                                 AS seg_type,
    a.cf_meta->>'env'                                  AS env,
    a.cf_meta->>'task'                                 AS task,
    (SELECT COUNT(*) FROM jsonb_object_keys(a.cf_algo)) AS algo_field_count,
    (SELECT COUNT(*) FROM jsonb_object_keys(a.cf_tag))  AS tag_count,
    m.cf_meta->>'mcap_uri'                             AS mcap_uri,
    m.cf_meta->>'vendor_id'                            AS vendor_id,
    m.cf_meta->>'device_id'                            AS device_id,
    a.created_at,
    a.updated_at,
    a.is_deleted
FROM assets a
LEFT JOIN mcap_files m ON a.mcap_file_id = m.mcap_file_id;

COMMENT ON VIEW v_assets_summary IS
  '管理视图:assets + mcap_files JOIN 摘要,便于 QA / 算法用户浏览';

-- 交付历史视图:asset ↔ delivery ↔ customer 的完整历史(替代 assets 上的单值快照)
CREATE OR REPLACE VIEW v_asset_delivery_history AS
SELECT
    di.asset_id,
    d.delivery_id,
    d.customer_id,
    d.status             AS delivery_status,
    d.delivered_at,
    d.cf_meta->>'contract_id'      AS contract_id,
    d.cf_meta->>'manifest_uri'     AS manifest_uri,
    (d.cf_meta->>'accepted_at')::TIMESTAMPTZ  AS accepted_at,
    (d.cf_meta->>'rejected_at')::TIMESTAMPTZ  AS rejected_at,
    (d.cf_meta->>'recalled_at')::TIMESTAMPTZ  AS recalled_at,
    d.cf_meta->>'rejection_reason' AS rejection_reason
FROM delivery_items di
JOIN deliveries d USING (delivery_id)
WHERE d.is_deleted = FALSE;

COMMENT ON VIEW v_asset_delivery_history IS
  '交付历史视图:asset 的完整交付/客户/批次轨迹,回答 "哪些 seg 交付给过哪些客户"';

-- =====================================================================
-- 11. Idempotency keys (for API-level idempotent POSTs)
-- =====================================================================
CREATE TABLE IF NOT EXISTS idempotency_keys (
    scope         TEXT         NOT NULL,
    idem_key      TEXT         NOT NULL,
    request_hash  TEXT         NOT NULL,
    status_code   INT          NOT NULL,
    response_json JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (scope, idem_key)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_created_at
  ON idempotency_keys (created_at DESC);

-- =====================================================================
-- 12. asset_algo_events - 算法状态变更审计表
--
-- 记录每次算法状态转换事件,用于审计和调试。
-- 每行 = 一次状态变更(start/finish/reset/依赖解锁)。
-- =====================================================================
CREATE TABLE IF NOT EXISTS asset_algo_events (
    event_id       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id       UUID         NOT NULL REFERENCES assets(asset_id),
    algo_key       TEXT         NOT NULL,
    prev_status    TEXT,
    new_status     TEXT         NOT NULL,
    run_id         TEXT,
    reason         TEXT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE asset_algo_events IS
  '算法状态变更审计表;每行=一次状态转换(start/finish/reset/依赖解锁);用于审计和调试。';

-- 12.1 按资产查事件(走 idx_algo_events_asset)
CREATE INDEX IF NOT EXISTS idx_algo_events_asset
  ON asset_algo_events (asset_id, created_at DESC);

-- 12.2 按算法+状态查事件
CREATE INDEX IF NOT EXISTS idx_algo_events_algo_status
  ON asset_algo_events (algo_key, new_status, created_at DESC);

-- 12.3 按 run_id 反查(部分索引,仅非空)
CREATE INDEX IF NOT EXISTS idx_algo_events_run_id
  ON asset_algo_events (run_id) WHERE run_id IS NOT NULL;

-- =====================================================================
-- 13. assets 表新增 cf_files JSONB 列 — 文件引用收敛
--
-- 所有关联文件的注册表。key=逻辑名(如 raw_mcap, hand_tracking@1.2.0),
-- value=GCS URI。finish_algo 时同步写入,一次查询拿到所有文件。
-- =====================================================================
ALTER TABLE assets ADD COLUMN IF NOT EXISTS
    cf_files JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_assets_cf_files_gin
    ON assets USING GIN (cf_files);

COMMENT ON COLUMN assets.cf_files IS
    '所有关联文件的注册表。key=逻辑名, value=GCS URI。finish_algo 时同步写入。';

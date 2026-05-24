# DataBrew Schema (rev.12)

| 字段 | 值 |
|------|----|
| 状态 | Authoritative schema SoT |
| 日期 | 2026-05-21 (rev.12 — 业务建模重构后) |
| 关联 | `./README.md`（业务模型 SoT）、`./archive/prd-rev11-historic.md` |
| 取代 | `schema-entity-aspect.md` (rev.2-rev.11) — Entity-Aspect 4 张通用 Aspect 表方案 |

---

## 0. 设计原则

> **业务实体优先，schema 实现服从业务建模。**

- 业务模型见 `./README.md`（4 类一等实体 + 7 个 asset_type + algo_runs / customers 新增）
- 本文档定义具体 PG 物理 schema（CREATE TABLE / 索引 / 约束 / trigger）
- **不再走 Entity-Aspect 4 张通用 Aspect 拆分**（rev.11 决策：现网 8 张 satellite 已经是 80% Aspect 模式，再拆收益接近零）
- **没上线红利**：schema 一次建对，无 migration 7 步骤 / dual-write / DROP COLUMN 延后

---

## 1. 完整表清单

| # | 表 | 类型 | 备注 |
|---|----|------|------|
| 1 | `assets` | Entity 主表 | 7 个 asset_type，含 raw_mcap |
| 2 | `logical_assets` | 协调表 | B 路由版本聚合（rev.11 已砍 current_asset_id）|
| 3 | `mcap_files` | raw_mcap 专用扩展表（1:1） | 1:1 PK = assets.asset_id |
| 4 | `algo_runs` | 执行事件 | **rev.12 新增**，run-level 元数据 |
| 5 | `customers` | 业务参考 | **rev.12 新增**，B2B 主数据 |
| 6 | `asset_tags` | 多源标签 | 含 source_type / source_name / source_version（`source` 为语义名） |
| 7 | `asset_algo_latest` | 算法当前态 | rev.12 加 run_id 外键 / is_pinned / run_inputs |
| 8 | `asset_metrics` | 标量指标 | 含 rating.* |
| 9 | `asset_eval_results` | 评估原档 | rev.12 加 run_id 外键 |
| 10 | `asset_events` | 审计流 | 三段式（actor + system_metadata + payload）|
| 11 | `asset_relations` | 关系边 | 含新 `derived_from` 边 |
| 12 | `actions` | action 专用扩展表（1:1） | rev.12 加 run_id 外键 + task_id 占位 |
| 13 | `deliveries / delivery_items` | 交付 | customer_id 加 外键 |
| 14 | `delivery_rules` | 资格规则 | |

> 计数口径（物理表）：前 12 行各 1 张 + `deliveries/delivery_items` 2 张 + `delivery_rules` 1 张 = **15 张 P1 物理表**。
> 若按逻辑模块口径，则是 13 组（把交付三件套视作 1 组）。

VIEW + trigger:
- `asset_relations_readable`（自然语言风查询）
- `trg_logical_assets_type_immutable`（LA1）

---

## 2. `assets` 主表（38 列宽表）

> **rev.12 决策**：撤回 Entity-Aspect 4 表拆分后，`assets` **保留 38 列宽表**（沿用现网 schemas/pg-phase0.sql:106-163 现有列）。
> rev.12 在 assets 表上**只新增 3 列**：`logical_asset_id` / `revision` / `is_current`（多版本身份）。其余 35 列业务字段位置**不动**。

### 2.1 rev.12 新增 3 列（多版本身份）

PG 不允许 `DEFAULT` 引用其他列（`DEFAULT asset_id` 是非法 DDL），必须**两段式**：

```sql
-- ① 先加 nullable 列
ALTER TABLE assets ADD COLUMN logical_asset_id TEXT;
ALTER TABLE assets ADD COLUMN revision         BIGINT NOT NULL DEFAULT 1;
ALTER TABLE assets ADD COLUMN is_current       BOOL NOT NULL DEFAULT true;

-- ② 回填 logical_asset_id = asset_id（自引用，首版规则）
UPDATE assets SET logical_asset_id = asset_id WHERE logical_asset_id IS NULL;

-- ③ 加 NOT NULL 约束（回填完成后才能加）
ALTER TABLE assets ALTER COLUMN logical_asset_id SET NOT NULL;

-- ④ 关键 partial unique：同一逻辑资产只能一个当前版
CREATE UNIQUE INDEX uq_assets_current_per_logical
  ON assets (logical_asset_id) WHERE is_current = TRUE AND is_deleted = FALSE;

CREATE INDEX idx_assets_logical ON assets (logical_asset_id);

-- ⑤ 反向外键到 logical_assets（DEFERRABLE，让首版 INSERT 顺序自由）
ALTER TABLE assets ADD CONSTRAINT fk_assets_logical
  FOREIGN KEY (logical_asset_id) REFERENCES logical_assets(logical_asset_id)
  DEFERRABLE INITIALLY DEFERRED;
```

**INSERT 时如何让 `logical_asset_id` 自动等于 `asset_id`**：由 AssetWriter 应用层填（首版 = asset_id 自引用；B 路由 = 继承 parent 的 logical_asset_id）。**不**走 DB trigger（trigger 增加 DB 隐式行为，调试难）。

### 2.2 mcap_file_id 约束修正（rev.12 L8）

现网 `mcap_files.mcap_file_id` 是独立 PK 且 `assets.mcap_file_id NOT NULL`，但 rev.12 `derived_asset` 跨多 segment 聚合时 mcap_file_id 应为 NULL（L8 不变式）：

```sql
-- 撤销 NOT NULL
ALTER TABLE assets ALTER COLUMN mcap_file_id DROP NOT NULL;

-- 加条件 CHECK：仅 derived_asset 允许 NULL，其他 asset_type 必须有 mcap_file_id
ALTER TABLE assets ADD CONSTRAINT chk_mcap_file_required
  CHECK (asset_type = 'derived_asset' OR mcap_file_id IS NOT NULL);
```

### 2.2 assets 完整 38 列概览（sql.md §4.2 / pg-phase0.sql 权威）

| 字段组 | 列 |
|--------|----|
| **身份**（4）| asset_id, asset_type, **logical_asset_id（rev.12 新）**, **revision（rev.12 新）** |
| **当前态**（1）| **is_current（rev.12 新）** |
| **血缘**（13）| mcap_file_id, parent_asset_id, root_asset_id, asset_level, split_method, split_algo_name, split_algo_version, split_run_id, split_reason, segment_index, parent_start_offset_ms, parent_end_offset_ms, segment_locator |
| **时间**（3）| start_timestamp_ns, end_timestamp_ns, duration_ms |
| **产物**（4）| materialization, storage_uri, thumb_uri, files JSONB |
| **治理**（10）| lifecycle_state, owner, reviewer, retention_tier, expire_at, delivery_count, last_delivered_at, last_delivered_to, tenant_id, project_id |
| **扩展 JSONB**（3）| metadata, algo_inputs_uris, annot_inputs_uris |
| **系统**（4）| is_deleted, created_at, updated_at, row_version |

→ **总计 38 + 3 (rev.12) = 41 列**。但 schemas/pg-phase0.sql 现网已含 38 列（含 lifecycle_state 等），rev.12 加 3 列后 = **41 列宽表**。

### 2.3 索引（沿用现网 + 新增 partial unique）

```sql
-- 沿用现网（pg-phase0.sql:165-188）
CREATE INDEX idx_assets_mcap_file ON assets(mcap_file_id, start_timestamp_ns, asset_id) WHERE NOT is_deleted;
CREATE INDEX idx_assets_lifecycle ON assets(lifecycle_state) WHERE NOT is_deleted;
CREATE INDEX idx_assets_type      ON assets(asset_type) WHERE NOT is_deleted;
CREATE INDEX idx_assets_parent    ON assets(parent_asset_id);
CREATE INDEX idx_assets_root      ON assets(root_asset_id);
CREATE INDEX idx_assets_created   ON assets(created_at DESC);
CREATE INDEX idx_assets_segment_locator ON assets(segment_locator);
-- GIN: assets(metadata), assets(algo_inputs_uris), assets(annot_inputs_uris)

-- rev.12 新增
CREATE UNIQUE INDEX uq_assets_current_per_logical
  ON assets(logical_asset_id) WHERE is_current = TRUE AND is_deleted = FALSE;
CREATE INDEX idx_assets_logical ON assets(logical_asset_id);
```

### 2.4 重要的非拆分声明（rev.12 撤回）

- ✗ **不**拆 asset_lineage / asset_content / asset_governance / asset_usage_stats 4 张通用 Aspect 表
- ✗ **不**建 asset_full VIEW
- ✗ **不**做 AssetWriter 同事务写 5 张表的拆分器
- ✓ **保留** assets 41 列宽表（含 rev.12 新加 3 列）
- ✓ **保留** 现网 8 张 satellite 表（asset_tags / asset_algo_latest / asset_metrics / asset_eval_results / actions / asset_events / mcap_files / asset_relations）
- 理由：现网已是 80% 多表分解（按写关注点拆 satellite，algo / tags / events 路径已不锁 assets 行）（algo / tags / metrics / events 写路径不锁 assets 行）；再拆 4 表收益≈0

> 未来真出性能问题（assets 表 > 500 列 / 单行 > 8KB / 行锁监控持续告警）才考虑 hot/cold 拆分。

---

## 3. `logical_assets`（B 路由协调）

先说人话：`logical_assets` 不是放资产内容的表，它是「版本总控卡」。

- 具体每个版本在 `assets`（`clipA_v1` / `clipA_v2` / ...）
- `logical_assets` 只记录这条版本链的汇总状态（`current_revision` / `total_revisions`）
- 作用是让「查当前版」和「版本列表统计」不必每次 `GROUP BY assets`

最小例子：

1. 首版：`assets` 插 `clipA_v1`，`logical_assets` 写 `current_revision=1,total_revisions=1`
2. 升版：`assets` 插 `clipA_v2`，同事务把 `clipA_v1.is_current=false`
3. 同事务更新 `logical_assets` 为 `current_revision=2,total_revisions=2`

→ 这样版本内容和版本协调分工清晰：`assets` 存内容，`logical_assets` 存总控状态。

```sql
CREATE TABLE logical_assets (
  logical_asset_id    TEXT PRIMARY KEY CHECK (logical_asset_id ~ '^[0-9A-Za-z]{8}$'),
  asset_type          TEXT NOT NULL,                   -- LA1: 不可改，trigger 阻拦
  display_name        TEXT,
  description         TEXT,
  owner               TEXT,
  status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),

  -- rev.11 砍掉 current_asset_id 冗余指针
  -- 查当前版本：SELECT asset_id FROM assets WHERE logical_asset_id=? AND is_current=true
  -- partial unique uq_assets_current_per_logical 保证唯一性

  -- ⚠ 以下两列是 precomputed cache（可由 assets 表 GROUP BY 重算），用作 B 路由协调 + 列表性能。
  --   事实源是 assets 表；此处冗余存储仅为查询便利（避免列表页 N+1 COUNT）。
  --   定期巡检脚本对账：SELECT logical_asset_id, count(*), max(revision)
  --                    FROM assets WHERE NOT is_deleted GROUP BY logical_asset_id
  current_revision    BIGINT NOT NULL DEFAULT 1 CHECK (current_revision >= 1),    -- cache: max(revision) of current is_current
  total_revisions     BIGINT NOT NULL DEFAULT 1 CHECK (total_revisions >= current_revision),  -- cache: count(*) for this logical

  metadata            JSONB NOT NULL DEFAULT '{}',
  extra               JSONB NOT NULL DEFAULT '{}',     -- §16.7 schema agility
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  row_version         BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX idx_lassets_owner   ON logical_assets(owner) WHERE owner IS NOT NULL;
CREATE INDEX idx_lassets_type    ON logical_assets(asset_type, status);
CREATE INDEX idx_lassets_status  ON logical_assets(status) WHERE status <> 'active';
CREATE INDEX idx_lassets_updated ON logical_assets(updated_at DESC);

-- LA1: asset_type 不可改
CREATE OR REPLACE FUNCTION trg_logical_assets_type_immutable() RETURNS TRIGGER AS $$
BEGIN
  IF NEW.asset_type IS DISTINCT FROM OLD.asset_type THEN
    RAISE EXCEPTION 'logical_assets.asset_type is immutable (LA1)';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER logical_assets_type_immutable
  BEFORE UPDATE ON logical_assets
  FOR EACH ROW EXECUTE FUNCTION trg_logical_assets_type_immutable();
```

### 3.1 同步逻辑（B 路由 3 字段更新）

```sql
-- 首版创建
BEGIN;
  INSERT INTO assets (asset_id='clipA_v1', logical_asset_id='clipA_v1', revision=1, is_current=true, ...);
  INSERT INTO logical_assets (logical_asset_id='clipA_v1', current_revision=1, total_revisions=1, ...);
COMMIT;

-- B 路由（生新版本，rev.11 决策：只更 3 字段）
BEGIN;
  INSERT INTO assets (asset_id='clipA_v2', logical_asset_id='clipA_v1', revision=2, is_current=true, ...);

  UPDATE assets SET is_current=false, row_version=row_version+1, updated_at=now()
   WHERE asset_id='clipA_v1' AND row_version=$parent_row_version;
  -- 若 0 行 → 409 (乐观锁冲突)

  UPDATE logical_assets SET
    current_revision = 2,
    total_revisions  = total_revisions + 1,
    updated_at       = now()
  WHERE logical_asset_id = 'clipA_v1';

  -- 版本边带原因
  INSERT INTO asset_relations (parent_asset_id='clipA_v2', child_asset_id='clipA_v1',
                                relation_type='revision_of',
                                metadata='{"run_id":"R001","algo":"hand_track@2.0"}');

  INSERT INTO asset_events (type='asset_revised', ...);
COMMIT;
```

---

## 4. `mcap_files`（raw_mcap 专用扩展表（1:1））

```sql
CREATE TABLE mcap_files (
  mcap_file_id     TEXT PRIMARY KEY REFERENCES assets(asset_id)
                   DEFERRABLE INITIALLY DEFERRED,
                   -- 1:1 with assets where asset_type='raw_mcap'

  -- 物理文件
  mcap_uri         TEXT NOT NULL,
  raw_hash_md5     TEXT,
  raw_hash_sha256  TEXT,
  size_bytes       BIGINT,
  codec            TEXT,

  -- 时间范围（与 assets.start/end_timestamp_ns 同步；冗余存取查询便利）
  start_timestamp_ns BIGINT,
  end_timestamp_ns   BIGINT,

  -- MCAP 元数据
  channel_count    INT,
  chunk_count      INT,
  topic_count      INT,
  topic_meta       JSONB DEFAULT '{}',

  -- 摄入状态
  ingest_state     TEXT NOT NULL DEFAULT 'pending',
                   -- pending | indexing | ready | failed
  ingest_error     TEXT,
  ingested_at      TIMESTAMPTZ,

  -- summary 索引状态（参考 mcap-index-metadata-v1.md）
  summary_index_state TEXT,
  summary_index_uri   TEXT,

  -- 采集 provenance
  vehicle_id       TEXT,
  vendor_id        TEXT,
  device_id        TEXT,
  recorded_at      TIMESTAMPTZ,

  -- 扩展
  metadata         JSONB NOT NULL DEFAULT '{}',
  process_state    JSONB NOT NULL DEFAULT '{}',

  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_mcap_hash_md5     ON mcap_files (raw_hash_md5) WHERE raw_hash_md5 IS NOT NULL;
CREATE INDEX idx_mcap_ingest_state       ON mcap_files (ingest_state);
CREATE INDEX idx_mcap_summary_pending    ON mcap_files (summary_index_state) WHERE summary_index_state='pending';
CREATE INDEX idx_mcap_recorded           ON mcap_files (recorded_at DESC);
CREATE INDEX idx_mcap_vendor_vehicle     ON mcap_files (vendor_id, vehicle_id);
```

**rev.12 关键变化：**
- `mcap_file_id` 改 外键 指 assets（不再是独立 PK）
- 上传 MCAP 时同事务建：assets(asset_type='raw_mcap') + mcap_files
- raw_mcap 享 asset 全套机制（logical_id / revision / lifecycle / tag / delivery）

---

## 5. `algo_runs`（执行事件，rev.12 新增）

```sql
CREATE TABLE algo_runs (
  run_id           TEXT PRIMARY KEY CHECK (run_id ~ '^[0-9A-Za-z]{16}$'),

  -- 算法身份
  algo_name        TEXT NOT NULL,
  algo_version    TEXT NOT NULL,
  algo_kind        TEXT NOT NULL CHECK (algo_kind IN ('processing','split','qa','enrichment')),

  -- 触发与状态
  triggered_by     TEXT NOT NULL,
                   -- scheduled_cron | manual:<user_id> | retry_of:<run_id>
  status           TEXT NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending','running','ok','failed','cancelled')),

  -- 时间
  started_at       TIMESTAMPTZ,
  finished_at      TIMESTAMPTZ,
  duration_ns      BIGINT GENERATED ALWAYS AS
                   (EXTRACT(EPOCH FROM (finished_at - started_at))::BIGINT * 1000000000) STORED,

  -- 输入快照（reproducibility 核心）
  input_filter     JSONB NOT NULL DEFAULT '{}',
                   -- {"asset_type":"segment", "tag.scenario":"kitchen"}
  input_asset_ids  TEXT[],
                   -- 小批量（<1000）物化；大批量靠 input_filter 复现
  params           JSONB NOT NULL DEFAULT '{}',

  -- 代码与镜像
  code_commit      TEXT,
  image_digest     TEXT,
  pipeline_name    TEXT,
  pipeline_version TEXT,

  -- 输出统计
  assets_processed INT,
  assets_succeeded INT,
  assets_failed    INT,
  actions_created  INT,
  metrics_written  INT,

  -- 非 asset 产物（10% 中间副产物 / per-frame data 等，C1 决策）
  outputs          JSONB NOT NULL DEFAULT '{}',
                   -- {
                   --   "assets_created": ["clipA_v2", "actionB_1"],
                   --   "intermediate_uri": "gs://.../runs/{run_id}/intermediate/",
                   --   "per_frame_data_uri": "gs://.../{run_id}/frames.parquet"
                   -- }

  -- 资源消耗
  cpu_seconds      BIGINT,
  gpu_seconds      BIGINT,
  cost_usd_micros  BIGINT,

  -- 失败信息
  error_class      TEXT,
                   -- timeout | oom | algo_internal | input_invalid
  error_message    TEXT,

  -- 系统字段
  tenant_id        TEXT,
  project_id       TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  row_version      BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX idx_algo_runs_algo      ON algo_runs(algo_name, algo_version, started_at DESC);
CREATE INDEX idx_algo_runs_status    ON algo_runs(status) WHERE status IN ('pending','running','failed');
CREATE INDEX idx_algo_runs_started   ON algo_runs(started_at DESC);
CREATE INDEX idx_algo_runs_triggered ON algo_runs(triggered_by) WHERE triggered_by LIKE 'manual:%';
```

### 5.1 子表 run_id 关联（rev.12：4 强 FK + asset_events JSONB 索引）

```sql
ALTER TABLE asset_algo_latest
  ADD CONSTRAINT fk_aal_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

ALTER TABLE actions
  ADD CONSTRAINT fk_act_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

ALTER TABLE asset_eval_results
  ADD CONSTRAINT fk_eval_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

-- asset_events.system_metadata->>'run_id' 保留 JSONB（不强 外键），保 GIN 索引
```

---

## 6. `customers`（业务参考，rev.12 新增）

```sql
CREATE TABLE customers (
  customer_id      TEXT PRIMARY KEY
                   CHECK (customer_id ~ '^[a-z][a-z0-9_-]{2,31}$'),
                   -- slug 风格，例 'cust_alpha_robotics'

  display_name     TEXT NOT NULL,
  legal_name       TEXT,

  status           TEXT NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active','trial','suspended','offboarded')),
  region           TEXT,  -- us | eu | cn | other

  sla_tier         TEXT NOT NULL DEFAULT 'standard'
                   CHECK (sla_tier IN ('standard','premium','enterprise')),
  account_owner    TEXT,

  compliance_tags  JSONB NOT NULL DEFAULT '[]',
                   -- ["gdpr_strict","no_pii","us_only"]
  exclude_tags     JSONB NOT NULL DEFAULT '[]',

  metadata         JSONB NOT NULL DEFAULT '{}',
  extra            JSONB NOT NULL DEFAULT '{}',

  onboarded_at     TIMESTAMPTZ,
  offboarded_at    TIMESTAMPTZ,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  row_version      BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX idx_customers_status      ON customers(status);
CREATE INDEX idx_customers_sla         ON customers(sla_tier);
CREATE INDEX idx_customers_account_own ON customers(account_owner) WHERE account_owner IS NOT NULL;
CREATE INDEX idx_customers_region      ON customers(region) WHERE region IS NOT NULL;

-- deliveries.customer_id 加 外键
ALTER TABLE deliveries
  ADD CONSTRAINT fk_delivery_customer
  FOREIGN KEY (customer_id) REFERENCES customers(customer_id);
```

---

## 7. `asset_tags`（多源标签，保留+加列）

沿用现网 `schemas/pg-phase0.sql:276-293` DDL，保留现有兼容列（`tag_type` / `tag_value_num` / `tag_value_bool` / `source_type`），在其上补 rev.12 多源唯一性。

```sql
-- 已有，仅列关键
asset_tags (
  asset_id        TEXT REFERENCES assets(asset_id),
  tag_key         TEXT,
  tag_value       TEXT,

  -- 现网兼容列（保留，不 drop）
  tag_value_num   DOUBLE PRECISION,
  tag_value_bool  BOOLEAN,
  tag_type        TEXT NOT NULL DEFAULT 'string',  -- string / number / bool / enum

  source_type     TEXT NOT NULL,     -- algo_sdk | rule_engine | human | system | compliance | llm | ...
  source_name     TEXT,              -- algo 名 / user_id / rule 名
  source_version  TEXT,              -- algo 版本 / rule 版本
  run_id          TEXT REFERENCES algo_runs(run_id) ON DELETE SET NULL,  -- rev.12 加 外键
  tenant_id       TEXT,
  project_id      TEXT,
  applied_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- 多源共存：用 generated column 替代非法 PK 表达式
  source_version_norm TEXT GENERATED ALWAYS AS (COALESCE(source_version, '')) STORED,
  CONSTRAINT uq_asset_tags_identity
    UNIQUE (asset_id, tag_key, tag_value, source_type, source_version_norm)
);

CREATE INDEX idx_atags_lookup       ON asset_tags(tag_key, tag_value, asset_id);
CREATE INDEX idx_atags_source       ON asset_tags(source_type, source_name, source_version);
CREATE INDEX idx_atags_propagation  ON asset_tags(tag_key) WHERE tag_key LIKE 'compliance.%';
```

> 说明：design 文档中常用 `source` 作为语义名称，落库列名统一用现网兼容的 `source_type`。

**rev.12 修代码 bug**：现网 `AssetTagRepo.Upsert` 只写 5 列（缺 source_name/source_version/run_id），是代码 bug。Upsert 路径要补全这 3 列写入。

---

## 8. `asset_algo_latest`（rev.12 加 pin + run_inputs + run_id 外键）

```sql
asset_algo_latest (
  asset_id         TEXT REFERENCES assets(asset_id),
  algo_name        TEXT NOT NULL,
  algo_version     TEXT NOT NULL,
  algo_kind        TEXT NOT NULL DEFAULT 'processing',
                   -- 仅 processing 一种（rev.11 决策：qa/split/enrichment 真需要时各起表，不共表）
  status           TEXT NOT NULL,   -- pending | running | ok | failed
  run_id           TEXT REFERENCES algo_runs(run_id) ON DELETE SET NULL,  -- rev.12 外键
  started_at       TIMESTAMPTZ,
  finished_at      TIMESTAMPTZ,
  method           TEXT,
  output_uri       TEXT,
  result_summary   JSONB DEFAULT '{}',

  -- rev.12 新增
  is_pinned        BOOL NOT NULL DEFAULT false,
  pinned_at        TIMESTAMPTZ,
  pinned_by        TEXT,
  run_inputs       JSONB NOT NULL DEFAULT '{}',
                   -- reproducibility: {"params":{...}, "input_revisions":{...}, "code_commit":"..."}

  metadata         JSONB NOT NULL DEFAULT '{}',
  extra            JSONB NOT NULL DEFAULT '{}',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (asset_id, algo_name)
);

CREATE INDEX idx_aal_algo       ON asset_algo_latest(algo_name, algo_version);
CREATE INDEX idx_aal_status     ON asset_algo_latest(algo_name, status);
CREATE INDEX idx_aal_pinned     ON asset_algo_latest(asset_id) WHERE is_pinned = true;
CREATE INDEX idx_aal_run        ON asset_algo_latest(run_id) WHERE run_id IS NOT NULL;
```

**rev.12 重要修订：** 之前 rev.10 提议 `algo_kind` 多 kind 共表（processing/qa/split/enrichment），rev.12 撤回 —— 保持单 kind = processing。qa / split / enrichment 真需要专表时各起表（如 `asset_qa_latest` 等），不共表。**减一个 PK 维度，简化查询**。

---

## 9. `asset_metrics`（保留）

沿用现网 `schemas/pg-phase0.sql` DDL，包括 rating.* 命名空间（人工评分 §7.4.1）。

```sql
-- 已有
asset_metrics (
  asset_id        TEXT REFERENCES assets(asset_id),
  metric_key      TEXT,
  metric_value    DOUBLE PRECISION,
  source          TEXT NOT NULL,
  source_name     TEXT,
  source_version  TEXT,
  target_type     TEXT,
  target_id       TEXT,
  eval_name       TEXT,
  eval_version    TEXT,
  scored_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  metadata        JSONB DEFAULT '{}',
  ...
  PRIMARY KEY (asset_id, metric_key, source, source_name, source_version, scored_at)
);

CREATE INDEX idx_ametrics_lookup    ON asset_metrics(metric_key, metric_value);
CREATE INDEX idx_ametrics_asset     ON asset_metrics(asset_id, target_type, target_id);
CREATE INDEX idx_ametrics_key_time  ON asset_metrics(metric_key, scored_at DESC);  -- rev.10 J1 评分时间线
```

---

## 10. `asset_eval_results`（rev.12 加 run_id 外键）

```sql
-- 已有
asset_eval_results (
  eval_result_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  asset_id         TEXT REFERENCES assets(asset_id),
  eval_name        TEXT NOT NULL,
  eval_version     TEXT NOT NULL,
  target_type      TEXT,
  target_id        TEXT,
  status           TEXT NOT NULL,
  result_payload   JSONB NOT NULL DEFAULT '{}',
  source           TEXT NOT NULL,
  source_name      TEXT,
  source_version   TEXT,
  run_id           TEXT REFERENCES algo_runs(run_id) ON DELETE SET NULL,  -- rev.12 外键
  evaluated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_aeval_asset       ON asset_eval_results(asset_id, target_type, target_id);
CREATE INDEX idx_aeval_name_ver    ON asset_eval_results(eval_name, eval_version);
CREATE INDEX idx_aeval_run         ON asset_eval_results(run_id) WHERE run_id IS NOT NULL;
```

---

## 11. `asset_events`（保留 + rev.10 三段式）

```sql
-- 沿用现网 schemas/pg-phase0.sql:349-376（已分区按月）
asset_events (
  event_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_seq        BIGINT NOT NULL,           -- 全局单调，订阅水位
  asset_id         TEXT,                       -- 可空（global event）
  event_type       TEXT NOT NULL,
  event_time       TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- 顶层 4 字段（高频审计查询，rev.10 三段式）
  actor            TEXT NOT NULL,             -- AV1: 严格命名（user: / algo_sdk:name@v 等）
  request_id       TEXT NOT NULL,
  idempotency_key  TEXT NOT NULL,
  caller_ip        TEXT,

  -- system_metadata JSONB（对齐 DataHub SystemMetadata）
  system_metadata  JSONB NOT NULL DEFAULT '{}',
                   -- run_id, pipeline_name, pipeline_version, algo_name, algo_version, ...

  -- payload JSONB（业务 diff，只装变更内容）
  event_payload    JSONB NOT NULL DEFAULT '{}',
                   -- changed_fields[], before{}, after{}

  publish_state    TEXT NOT NULL DEFAULT 'pending',
  tenant_id        TEXT,
  project_id       TEXT,
  retention_until  TIMESTAMPTZ,
  CONSTRAINT uq_event_seq UNIQUE (event_seq)
) PARTITION BY RANGE (event_time);

CREATE INDEX idx_aevents_asset_time ON asset_events(asset_id, event_time DESC);
CREATE INDEX idx_aevents_type_time  ON asset_events(event_type, event_time DESC);
CREATE INDEX idx_aevents_actor      ON asset_events(actor, event_time DESC);
CREATE INDEX idx_aevents_run        ON asset_events((system_metadata->>'run_id'))
                                    WHERE system_metadata ? 'run_id';
CREATE INDEX idx_aevents_changed    ON asset_events USING GIN ((event_payload->'changed_fields'));
CREATE INDEX idx_aevents_pending    ON asset_events(publish_state) WHERE publish_state = 'pending';
```

---

## 12. `asset_relations`（rev.12 加 `derived_from` 边）

> ⚠️ **边方向约定（ER1 不变式）**：
> `parent_asset_id` = **主语 / 产物 / 新物**（关系的发起方）
> `child_asset_id`  = **宾语 / 源 / 旧物**（关系的指向）
> **永远朝 parent 方向读：「parent {relation_type} child」是合法句子。**
>
> 反直觉提醒：split_from 的 parent 是「切出来的新物」，child 是「源 segment」。详见 §12.1 读法对照表。

```sql
asset_relations (
  parent_asset_id  TEXT NOT NULL REFERENCES assets(asset_id),    -- ⚠ 主语 / 产物 / 新物
  child_asset_id   TEXT NOT NULL REFERENCES assets(asset_id),    -- ⚠ 宾语 / 源 / 旧物
  relation_type    TEXT NOT NULL CHECK (relation_type IN
                   ('split_from','derived_from','contains','sampled_from','merged_from','revision_of')),
                   -- rev.12 新增 derived_from（算法产物专用，区别于人工 split_from）

  metadata         JSONB NOT NULL DEFAULT '{}',
                   -- revision_of 边: {"type":"algo_rerun","run_id":"R789","algo":"hand_track@2.0"}
                   -- derived_from 边: {"run_id":"R001","algo_name":"hand_track","algo_version":"2.0"}

  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

  PRIMARY KEY (parent_asset_id, child_asset_id, relation_type)
);

CREATE INDEX idx_arel_child         ON asset_relations(child_asset_id, relation_type);
CREATE INDEX idx_arel_derived_run   ON asset_relations((metadata->>'run_id'))
                                    WHERE relation_type = 'derived_from';
```

**写入示例（验证方向不写反）：**

```sql
-- ✅ 正确：clip_X 是从 seg_Y 切出来的
INSERT INTO asset_relations (parent_asset_id, child_asset_id, relation_type)
VALUES ('clip_X', 'seg_Y', 'split_from');
-- 读法：clip_X IS split_from seg_Y

-- ✅ 正确：clipA_v2 是 clipA_v1 的新版本
INSERT INTO asset_relations (parent_asset_id, child_asset_id, relation_type)
VALUES ('clipA_v2', 'clipA_v1', 'revision_of');
-- 读法：clipA_v2 IS revision_of clipA_v1

-- ❌ 错误（方向反了）：以为 parent 是源
INSERT INTO asset_relations (parent_asset_id, child_asset_id, relation_type)
VALUES ('seg_Y', 'clip_X', 'split_from');
-- 这会读成：seg_Y IS split_from clip_X，语义错误
```

### 12.1 边方向（ER1 不变式）

> **`parent_asset_id` = 主语，`child_asset_id` = 宾语。「parent {relation_type} child」永远是合法句子。**

| relation_type | parent | child | 读法 |
|---------------|--------|-------|------|
| `split_from` | 切出来的 child | 源 parent | "clip_X is **split_from** seg_Y"（**人工/规则**切）|
| `derived_from` | 派生产物（算法产物）| 源 asset | "clip_X is **derived_from** seg_Y"（**算法**跑出，rev.12 新增）|
| `contains` | 父容器 task | 内含 action | "task_T **contains** action_A" |
| `sampled_from` | 采样产物 frame | 源 segment | "frame_F is **sampled_from** seg_Y" |
| `merged_from` | 融合产物 derived | 源 asset | "merged_X is **merged_from** source_Y" |
| `revision_of` | 新版 asset | 老版 asset | "clipA_v2 is **revision_of** clipA_v1" |

### 12.2 helper VIEW

```sql
CREATE VIEW asset_relations_readable AS
SELECT parent_asset_id AS subject, relation_type AS predicate,
       child_asset_id AS object, metadata, created_at
FROM asset_relations;

-- 用法
SELECT subject, predicate, object FROM asset_relations_readable
WHERE predicate = 'derived_from' AND object = 'seg_Y001';
-- → 哪些资产派生自 seg_Y001？
```

---

## 13. `actions`（保留 + rev.12 加 task_id 占位）

```sql
-- 沿用现网 schemas/pg-phase0.sql:226-270
actions (
  action_id        TEXT PRIMARY KEY REFERENCES assets(asset_id),  -- 1:1 with assets where asset_type='action'
  asset_id         TEXT NOT NULL REFERENCES assets(asset_id),     -- parent segment 或 task
  start_ns         BIGINT NOT NULL,
  end_ns           BIGINT NOT NULL,
  primary_label    TEXT,
  labels           TEXT[] DEFAULT '{}',
  confidence       DOUBLE PRECISION,
  source_type      TEXT NOT NULL,              -- algo | human | rule
  source_name      TEXT,
  source_version   TEXT,
  run_id           TEXT REFERENCES algo_runs(run_id) ON DELETE SET NULL,  -- rev.12 外键
  task_id          TEXT,                       -- rev.12 占位，P1.5 加 外键 -> annotation_tasks
  attrs            JSONB DEFAULT '{}',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  version          BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX idx_actions_asset_time ON actions(asset_id, start_ns);
CREATE INDEX idx_actions_label      ON actions(primary_label);
CREATE INDEX idx_actions_labels     ON actions USING GIN (labels);
CREATE INDEX idx_actions_run        ON actions(run_id) WHERE run_id IS NOT NULL;
CREATE INDEX idx_actions_task       ON actions(task_id) WHERE task_id IS NOT NULL;
```

---

## 14. `deliveries` + `delivery_items`（rev.12 customer_id 加 外键）

```sql
-- 沿用现网 schemas/pg-phase0.sql:464-515
deliveries (
  delivery_id      TEXT PRIMARY KEY,
  customer_id      TEXT NOT NULL REFERENCES customers(customer_id),  -- rev.12 外键
  contract_id      TEXT,                       -- P1 保持 TEXT，P1.5 视情况加 contracts 表
  delivery_type    TEXT NOT NULL DEFAULT 'asset_set',
  status           TEXT NOT NULL DEFAULT 'draft',
                   -- draft | resolving | ready_to_commit | committed
  manifest_uri     TEXT,
  replay_manifest_uri TEXT,
  delivered_at     TIMESTAMPTZ,
  metadata         JSONB DEFAULT '{}',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deliveries_customer ON deliveries(customer_id, created_at DESC);
CREATE INDEX idx_deliveries_status   ON deliveries(status);

delivery_items (
  delivery_id      TEXT NOT NULL REFERENCES deliveries(delivery_id),
  asset_id         TEXT NOT NULL REFERENCES assets(asset_id),
  expected_revision BIGINT,                    -- rev.10 C2-D：commit 时校验 == assets.revision
  payload_mode     TEXT NOT NULL,              -- materialized | virtual
  forced           BOOL NOT NULL DEFAULT false, -- rev.10 C2 commit force_overrides
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (delivery_id, asset_id)
);

CREATE INDEX idx_di_asset ON delivery_items(asset_id);
```

---

## 15. `delivery_rules`（rev.10 dsl_version + enforce_mode）

```sql
delivery_rules (
  rule_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,
  owner         TEXT NOT NULL,
  customer_id   TEXT REFERENCES customers(customer_id),  -- rev.12 外键，NULL=通用
  query_dsl     JSONB NOT NULL,
  dsl_version   TEXT NOT NULL DEFAULT 'v1',
  enforce_mode  TEXT NOT NULL DEFAULT 'block'
                CHECK (enforce_mode IN ('block','warn','tag_only')),
  rating_scope  TEXT NOT NULL DEFAULT 'current'
                CHECK (rating_scope IN ('current','logical')),
  is_active     BOOL NOT NULL DEFAULT true,
  version       BIGINT NOT NULL DEFAULT 1,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_drules_customer ON delivery_rules(customer_id, is_active);
CREATE INDEX idx_drules_active   ON delivery_rules(is_active) WHERE is_active = true;
```

---

## 16. 不变式索引（精简版，10 条核心）

| # | 不变式 | enforce 位置 |
|---|--------|-------------|
| **L1-L7** | 层级树（segment 父 = raw_mcap；clip/action 父 = segment；task 父子约束；mcap_file_id 继承根；等）| AssetWriteValidator |
| **N1** | 同 logical_asset_id 仅一条 is_current=true | DB partial unique index |
| **N2** | revision 严格线性递增（v1→v2→v3，禁止分叉）| AssetWriteValidator |
| **M1-M3** | materialization：virtual ⇒ storage_uri NULL；materialized ⇒ storage_uri 符合 r<n>-<uuid> 格式；materialized → virtual 禁止 | AssetWriteValidator |
| **AE1** | 任何业务表写入必同事务写 asset_events | AssetWriter 强制 |
| **AE2** | asset_events append-only（PG GRANT 限制 UPDATE/DELETE）| DB 权限 |
| **AV1** | asset_events.actor 必须命中命名约定正则（机器类必带 @version）| AssetWriter Validator |
| **AR1**（rev.11）| PATCH 命中 revision_triggers → 422 + 指引 POST /revisions（不自动升 B）| API gateway + Validator |
| **LA1**（rev.11）| logical_assets.asset_type 不可改 | PG trigger |
| **LA2**（rev.11）| current_revision <= total_revisions | CHECK 约束 |
| **ER1**（rev.10）| asset_relations 边方向统一（parent=主语，child=宾语）| 文档约定 + helper VIEW |
| **I4** | 写入幂等三层（Idempotency-Key 必填 + parent_asset_version 乐观锁 + DB partial unique 兜底）| API gateway + Validator + DB |
| **IR1** | 点查走 PG（asset_full VIEW 等价，但 rev.12 砍 VIEW 概念 → 直接 SELECT FROM assets + hydrate）；列表走 ES | code review |

> **rev.12 砍掉的不变式：** AS1/AS2/OPT1/OPT3/RL1/RL2/RL3/LA3/LA4/AK1/RL4 等共 ~25 条 —— 这些是 Aspect 拆分场景下才需要的（VIEW 同步 / 跨表 Validator / Orphan 巡检 / algo_kind 共表约束等）。砍 Aspect 拆分 = 自动砍这些不变式。

---

## 17. P1 上线 sequence（按「新建 / 改造 / 破坏性 / dev 数据」四类分清）

> **rev.12 红利的前提**：当前**未上线 → 无 prod 数据 → 无双写期 / 无 DROP COLUMN 延后**。
>
> ⚠️ **若未来已上 prod**（有真实客户数据），本章节所有 ALTER / DROP / 破坏性 migration 都必须走标准 7 步骤（pre-deploy / dual-write / shadow read / cutover / cleanup / rollback plan / monitoring）—— 本文档**不**覆盖那个场景。
>
> dev 环境可能已经跑过 `schemas/pg-phase0.sql` 部分表（assets / asset_tags / asset_algo_latest / asset_events / actions / mcap_files / asset_relations 等已存在）→ 必须区分「新建表」vs「改造已有表」vs「**破坏性重建**」vs「dev 数据处理」。

### 17.1 类别 A — 全新表（直接 `CREATE TABLE`）

```sql
-- A1. logical_assets（rev.12 新增协调表）
CREATE TABLE logical_assets (...);   -- §3
CREATE TRIGGER logical_assets_type_immutable ...;

-- A2. algo_runs（rev.12 新增执行事件实体）
CREATE TABLE algo_runs (...);        -- §5

-- A3. customers（rev.12 新增业务参考实体）
CREATE TABLE customers (...);        -- §6

-- A4. 清理幽灵索引（pg-phase0.sql:169 引用已 drop 的 assets.status 列）
--     migration 025_drop_assets_status.sql 已 drop 列，但 pg-phase0.sql 未同步
DROP INDEX IF EXISTS idx_assets_status;
```

→ **3 张全新表 + 1 个清理，无历史数据，CREATE 直接成功**。

### 17.2 类别 B — 已有表改造（`ALTER TABLE`）

```sql
-- B1. assets 加 rev.12 多版本身份 3 列 + partial unique（两段式，详见 §2.1）
ALTER TABLE assets ADD COLUMN logical_asset_id TEXT;
ALTER TABLE assets ADD COLUMN revision         BIGINT NOT NULL DEFAULT 1;
ALTER TABLE assets ADD COLUMN is_current       BOOL NOT NULL DEFAULT true;
UPDATE assets SET logical_asset_id = asset_id WHERE logical_asset_id IS NULL;
ALTER TABLE assets ALTER COLUMN logical_asset_id SET NOT NULL;

CREATE UNIQUE INDEX uq_assets_current_per_logical
  ON assets(logical_asset_id) WHERE is_current = TRUE AND is_deleted = FALSE;

CREATE INDEX idx_assets_logical ON assets(logical_asset_id);

ALTER TABLE assets ADD CONSTRAINT fk_assets_logical
  FOREIGN KEY (logical_asset_id) REFERENCES logical_assets(logical_asset_id)
  DEFERRABLE INITIALLY DEFERRED;

-- B1.b mcap_file_id 约束修正（L8）
ALTER TABLE assets ALTER COLUMN mcap_file_id DROP NOT NULL;
ALTER TABLE assets ADD CONSTRAINT chk_mcap_file_required
  CHECK (asset_type = 'derived_asset' OR mcap_file_id IS NOT NULL);

-- B2. mcap_files 改为 raw_mcap 资产的专用扩展表（PK 改为 FK 到 assets）
-- 已有 mcap_files 表，mcap_file_id 是独立 PK；rev.12 让它 FK 到 assets.asset_id
ALTER TABLE mcap_files ADD CONSTRAINT fk_mcap_asset
  FOREIGN KEY (mcap_file_id) REFERENCES assets(asset_id)
  DEFERRABLE INITIALLY DEFERRED;

-- B3. asset_algo_latest 加 pin + run_inputs + run_id FK
ALTER TABLE asset_algo_latest
  ADD COLUMN is_pinned   BOOL NOT NULL DEFAULT false,
  ADD COLUMN pinned_at   TIMESTAMPTZ,
  ADD COLUMN pinned_by   TEXT,
  ADD COLUMN run_inputs  JSONB NOT NULL DEFAULT '{}',
  ADD CONSTRAINT fk_aal_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

-- B4. actions 加 run_id FK + task_id 占位
ALTER TABLE actions
  ADD CONSTRAINT fk_act_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;
ALTER TABLE actions ADD COLUMN task_id TEXT;  -- P1.5 加 FK 到 annotation_tasks

-- B5. asset_eval_results 加 run_id FK
ALTER TABLE asset_eval_results
  ADD CONSTRAINT fk_eval_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

-- B6. asset_tags 补 run_id FK（schema 早有列，只缺 FK）
ALTER TABLE asset_tags
  ADD CONSTRAINT fk_atags_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

-- B7. deliveries 加 customer_id FK
ALTER TABLE deliveries
  ADD CONSTRAINT fk_delivery_customer FOREIGN KEY (customer_id) REFERENCES customers(customer_id);

-- B8. asset_relations CHECK 加 derived_from 边
ALTER TABLE asset_relations
  DROP CONSTRAINT chk_relation_type,
  ADD CONSTRAINT chk_relation_type CHECK (relation_type IN
    ('split_from','derived_from','contains','sampled_from','merged_from','revision_of'));
```

→ **8 处 ALTER，沿用现有数据 + 加列/加 FK，无数据迁移**（B1 的 `UPDATE ... SET logical_asset_id = asset_id` 自动回填）。

### 17.2.D 类别 D — 破坏性重建（仅 `asset_tags` 一张表）

> ⚠️ **真严重 GAP**：现网 `asset_tags` PK = `(asset_id, tag_key)` 2 列，rev.12 设计要求多源共存（同 asset 同 key 可有多个 source）必须改 PK 为 5 列。这不是 ALTER 能搞定，需要 DROP / 重建 / 回填依赖索引。

```sql
-- D1. asset_tags PK 改造（2 列 → 5 列多源共存）
--   现网：PRIMARY KEY (asset_id, tag_key)
--   rev.12：UNIQUE (asset_id, tag_key, tag_value, source_type, source_version_norm)
ALTER TABLE asset_tags DROP CONSTRAINT asset_tags_pkey;

-- 加 surrogate PK（让 PK 不参与业务唯一性）
ALTER TABLE asset_tags ADD COLUMN tag_id BIGSERIAL PRIMARY KEY;

-- 保留现网兼容列（不 drop，避免破坏现有数值/布尔筛选能力）
-- tag_value_num / tag_value_bool / tag_type 沿用现网语义

-- 用 generated column + UNIQUE CONSTRAINT（可被 ON CONFLICT 稳定引用）
ALTER TABLE asset_tags
  ADD COLUMN source_version_norm TEXT GENERATED ALWAYS AS (COALESCE(source_version, '')) STORED;
ALTER TABLE asset_tags
  ADD CONSTRAINT uq_asset_tags_multi_source
  UNIQUE (asset_id, tag_key, tag_value, source_type, source_version_norm);

-- 现网 source_type ∈ {human/algo/rule/system}（4 种），rev.12 source ∈ {algo_sdk/rule_engine/human/system/compliance/llm/vendor}（7+ 种）
-- 这里沿用 source_type 列名（不再造 source 新列），新 enum 值由 application 层 lint
ALTER TABLE asset_tags DROP CONSTRAINT IF EXISTS chk_source_type;
ALTER TABLE asset_tags ADD CONSTRAINT chk_source_type
  CHECK (source_type IN ('human','algo_sdk','rule_engine','system','compliance','llm','vendor','algo','rule'));
  -- 保留旧 'algo'/'rule' 值兼容旧数据；新写入走 'algo_sdk' / 'rule_engine'

-- 同时修代码 bug：repos.go:1314 AssetTagRepo.Upsert 必须开始填 source_name / source_version / run_id 3 列
-- 这是 application 层改动，不在 schema migration 范围
```

> **dev 走捷径**：若 dev 数据无价值，可直接 `TRUNCATE asset_tags` 后跑新 DDL（见 17.3 C1），跳过 PK 改造。**prod 必须走完整 7 步骤**。

### 17.3 类别 C — dev 环境数据处理

```sql
-- C1. 若 dev 环境有 dummy 数据导致 ALTER 失败（如 DEFAULT 与现有数据冲突）
TRUNCATE TABLE assets CASCADE;          -- dev 库可清；prod 永不可这么干（但未上线无 prod）
TRUNCATE TABLE mcap_files CASCADE;
-- ... 其他相关表

-- C2. 若 dev seed 数据要保留，按需在 B1 ALTER 后回填
UPDATE assets SET logical_asset_id = asset_id WHERE logical_asset_id IS NULL;
-- B1 两段式 UPDATE 已经搞定，这步通常不需要
```

→ **dev 环境随便 truncate；prod 不存在，无风险**。

### 17.4 上线 VIEW + cron

```sql
-- VIEW
CREATE VIEW asset_relations_readable AS
SELECT parent_asset_id AS subject, relation_type AS predicate,
       child_asset_id AS object, metadata, created_at
FROM asset_relations;

-- logical_assets 同步 cron（B 路由事务内由 AssetWriter 写；同时定期巡检对账）
-- 见 §3.1 logical_assets 表注释
```

### 17.5 业务代码改造（**真实工程量，rev.12 修订**）

> ⚠️ **rev.12 第二轮 review 修订**：原估算「3-5 天 / ~1 周」**严重低估**。下表是按完整 GAP 矩阵（README §3.5）覆盖的诚实估算。

| # | 改造 | 复杂度 | 工时估 |
|---|------|------|--------|
| 1 | AssetWriter B 路由：INSERT new asset + UPDATE old is_current + UPDATE logical_assets 计数（含原子切 `is_current` 事务）| 中 | 1 天 |
| 2 | **AssetWriteValidator 实现 L1-L7 不变式**（每次 INSERT/UPDATE 多 1-3 SELECT 查 parent 元数据；含批量 INSERT 时 share parent 查询的优化）| 中 | **3 天** |
| 3 | **algo API per-asset → per-run 翻转**：新建 `POST /algo-runs` / `POST /algo-runs/{id}/finish`；废弃或并存 `POST /assets/:id/algo/:algo/start`（见 [algo-runs.md §11](./design/algo-runs.md)）；重写 `algo_handler.go` / `algo_usecase.go` | 高 | **5 天** |
| 4 | AlgoRunWriter：run_id 写入 4 张子表（asset_algo_latest / actions / asset_eval_results / asset_tags）+ asset_events.system_metadata JSONB | 中 | 1 天 |
| 5 | **ES builder.go 改造**：新增 `logical_asset_id` / `revision` / `is_current` 投影；rebuild ES 索引；应用 IR1（点查走 PG，列表走 ES 接受 1-3s 滞后）| 中 | **3 天** |
| 6 | **`mcap_files` 同事务化**：重写 `McapFileRepo.Set` + `AssetRepo.InsertNew`，让 `POST /mcap-files/upload/finalize` 在一个 `withMutationTx` 里同时写 `assets` + `mcap_files` + `asset_events`（FK 约束才生效）| 中 | 2 天 |
| 7 | **asset_tags 多源破坏性 migration**：DROP PK + surrogate id + 5 列 UNIQUE；修 `AssetTagRepo.Upsert`（补 source_name / source_version / run_id 字段写入）| 中 | 2 天 |
| 8 | CustomerRepo + delivery customer_id FK 校验 | 低 | 0.5 天 |
| 9 | **delivery 状态机 + C2 commit 协议**：`draft → resolving → ready_to_commit → committed`；含 expected_revision 校验、delivery_rules 校验、409 冲突解决（见 [customers-and-deliveries §4.2 + §5](./design/customers-and-deliveries.md)）；同时保留 `POST /deliveries` 旧路径作降级 | 高 | **5 天** |
| 10 | 集成测试 + 回归 + 性能验证（含 AssetWriteValidator 在批量场景下的 N+1 评估）| 中 | **3-5 天** |
| **小计** | | | **25-30 天 ≈ 3-4 周** |

### 17.6 总工程量

| 阶段 | 工作 | 时间 |
|------|------|------|
| Schema 改造（17.1-17.4） | 全部 DDL 跑一遍（含类别 D 破坏性重建）| **2 小时** |
| 业务代码改造（17.5） | 10 项改造 | **3-4 周** |
| **总** | | **3-4 周（按 1 人全职估算）** |

> **降级策略**：若业务急需先上 customers / deliveries（无版本化），可只做 #8 + #9 简化版（C2 commit 跳过 expected_revision 校验），约 1 周。其余 8 项按 README §4.1.1 实施依赖图分阶段落地。

---

## 18. P1.5 启用（新增 6 张物理表；DDL 已就绪，业务对接时跑 migration）

```sql
-- annotation_tasks（B 规模标注队，独立设计 doc 落地）
CREATE TABLE annotation_tasks (...);   -- 见 annotation-tasks-p1.5.md

-- ML 平台对接
CREATE TABLE datasets (...);            -- 沿用现 schemas/pg-phase0.sql:521-538
CREATE TABLE dataset_snapshots (...);   -- 沿用现 schemas/pg-phase0.sql:540-567
CREATE TABLE training_runs (...);       -- 沿用现 schemas/pg-phase0.sql:573-614

-- Amundsen 发现层
CREATE TABLE asset_usage_stats (...);   -- view/download/favorite/trending_score
CREATE TABLE asset_favorites (...);

-- actions.task_id 外键
ALTER TABLE actions ADD CONSTRAINT fk_actions_task FOREIGN KEY (task_id) REFERENCES annotation_tasks(task_id);
```

---

## 19. 不做的事（rev.12 永不做）

```
✗ asset_lineage / asset_content / asset_governance 4 张通用 Aspect 拆分
✗ asset_full VIEW（不拆就不需要兼容层）
✗ HydrateScope 按需 fetch 框架
✗ AssetWriter 同事务写 5 表的"拆分器"概念
✗ asset_algo_latest 的 algo_kind 多 kind 共表（rev.10 撤回，保单 kind）
✗ databrew:// URI scheme
✗ G3 GCS 双 commit + orphan GC（改 pending → materialized 状态机）
✗ A2 PATCH 自动升 B（rev.11 改 422）
✗ MetadataChangeProposal (MCP) wire format
✗ workflows / pipelines 编排表
✗ catalog_objects（移 P2 不做的事）
✗ subscriptions / billing
✗ 38 条编号不变式（精简到 ~13 条）
✗ 24 命名子系统（精简到 6 个）
✗ Schema Agility Playbook 五段式（保留单条实践：热字段升列，冷字段进 metadata JSONB）
```

---

## 20. 一句话总结

> **rev.12 schema = P1 15 张物理表（13 组逻辑模块）+ P1.5 新增 6 张（累计 21 张）+ asset_relations 加 derived_from 边。砍通用 Aspect 拆分整套（4 表 + VIEW + 25 不变式 + 5 段式 playbook），保留现网 8 张 satellite 模式。重点是补 algo_runs / customers / mcap_files 专用扩展表（1:1）三个真业务缺口。**

**与 rev.11 的核心差异：**
- ✗ 不再有 asset_lineage / asset_content / asset_governance / asset_usage_stats 四张通用 Aspect 表
- ✗ 不再有 asset_full VIEW
- ✗ 不再有 9 列 assets 壳（保留 38 列宽表，业务字段不动）
- ✓ 新增 algo_runs（执行事件类一等实体）
- ✓ 新增 customers（业务参考类一等实体）
- ✓ mcap_files 改为 raw_mcap 的专用扩展表（1:1）（与 actions 对称）
- ✓ asset_relations 加 derived_from 边（算法产出专用）
- ✓ asset_algo_latest 加 is_pinned + run_inputs + run_id 外键
- ✓ 不变式 38 → 13（精简）
- ✓ 命名子系统 24 → 6（精简）
- ✓ Migration 7 步骤 → 1 步骤（没上线红利）

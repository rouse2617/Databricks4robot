# DataBrew Schema: Entity-Aspect 落地设计

| 字段 | 值 |
|------|-----|
| 状态 | Draft（配套 PRD rev.9） |
| 日期 | 2026-05-21（rev.2 — 合并 temporal / 拆 usage_stats / 移除 revision_reason） |
| 关联 | `prd-unified-asset-catalog-provenance.md`、`data-platform-design.md` |
| 参考 | DataHub `entity_aspect` 模式；OpenMetadata Entity-Extension |
| SoT 声明 | **本文档是 schema 的权威定义**；PRD §3.8 仅给概览。两处不一致时以本文档为准 |

---

## 1. 设计哲学

> **「Entity 是身份证，Aspect 是档案袋。」**

类比视频平台：
- `raw_mcap` = 2 小时未剪辑原始母带（物理事实）
- `segment` = up 主精剪出的「第 1 集：厨房扫地演示（5 分钟）」（业务实体）
- `asset_tags` = 给这集贴的标签档案袋（Aspect）
- `actions` = 这集里的动作标注档案袋（Aspect）
- `asset_algo_latest` = 这集被哪些算法处理过的档案袋（Aspect）

**Entity 壳只做一件事：给业务资产一个全局唯一、稳定的身份证（asset_id）。**
所有内容、状态、标签、算法、血缘，都挂在这个身份证上，独立演进、互不阻塞。

---

## 2. 并发优势（为什么一定要 Entity-Aspect）

```text
算法 A 写 actions（动作标注结果）
算法 B 写 asset_metrics（质量评分）
运营 C 写 asset_tags（贴业务标签）
前端 D view_count += 1（asset_usage_stats 高频累加）

四个操作同时进行 → 写不同表的不同行 → 零行锁竞争
assets 壳表完全不参与！
```

如果所有字段都在 `assets` 一张表：
- 算法 A 改字段 → 锁住 `seg_A001` 那一行
- 算法 B / 运营 C / view 累加都要 → 等待 → 高并发下行锁风暴

---

## 3. Entity 壳：`assets` 表（9 列，永远不增加列）

```sql
CREATE TABLE assets (
  -- 身份
  asset_id         TEXT PRIMARY KEY CHECK (asset_id ~ '^[0-9A-Za-z]{8}$'),
  asset_type       TEXT NOT NULL,           -- raw_mcap|segment|clip|action|frame|task|derived_asset

  -- 版本血缘（Entity 级核心身份，必须留在壳里以支持 partial unique index）
  logical_asset_id TEXT NOT NULL,           -- 首版 = asset_id；走 B 路由时继承父
  revision         BIGINT NOT NULL DEFAULT 1,   -- rev.10 E2：与 row_version / event_seq 类型统一（避免「INT 表示 revision 很小」隐含假设）
  is_current       BOOL NOT NULL DEFAULT true,

  -- 系统字段
  is_deleted       BOOL NOT NULL DEFAULT false,  -- DB 级软删（与业务 lifecycle 正交，见 §3.1）
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  row_version      BIGINT NOT NULL DEFAULT 1     -- PostgreSQL 乐观锁（≠ 业务 revision）
);

-- 当前版本唯一性约束：同一逻辑资产只能有一个 current 版本
CREATE UNIQUE INDEX uq_assets_current_per_logical
  ON assets (logical_asset_id) WHERE is_current = TRUE AND is_deleted = FALSE;

CREATE INDEX idx_assets_type    ON assets (asset_type) WHERE is_deleted = FALSE;
CREATE INDEX idx_assets_logical ON assets (logical_asset_id);
```

### 3.1 字段不变性说明

| 字段 | 为什么必须留在 Entity 壳 |
|------|--------------------|
| `asset_id` / `asset_type` | 身份与类型 discriminator |
| `logical_asset_id` / `revision` / `is_current` | partial unique index `uq_assets_current_per_logical` 强约束依赖 |
| `is_deleted` | 同一 partial unique index 依赖（必须在本表）|
| `created_at` / `updated_at` | 通用审计 |
| `row_version` | 乐观锁（PG 级，≠ 业务 `revision`）；写入幂等三层（§9.3）依赖 |

### 3.2 与业务 lifecycle 的关系

- **`is_deleted`**：DB 级软删标记。`is_deleted=true` 的行从所有「常态查询」自动隔离（partial unique index + `asset_full` VIEW 都会过滤）。
- **`lifecycle_state`**（在 `asset_governance`）：业务级生命周期 `ready / archived / superseded`，用户能看到。
- **两者正交**：`is_deleted` 不是 lifecycle 的一个值；可以 `lifecycle_state='archived' AND is_deleted=false`（归档但未删除）。

### 3.3 revision_reason 不在壳里（rev.2 决策）

历史上 `revision_reason` JSONB 列曾在 `assets` 壳里。**rev.2 移除**，原因：

- Entity 壳追求极薄；条件性 JSONB 与原则冲突
- 信息已经在两处冗余：`asset_events.system_metadata`（审计权威）+ `asset_relations(revision_of).metadata`（血缘边带）

**真源**：`asset_relations` 中 `revision_of` 边的 `metadata` JSONB 字段（如 `{"type":"algo_rerun","run_id":"R789","algo":"hand_track@2.0"}`）。
查询「某资产这版怎么来的」= `SELECT metadata FROM asset_relations WHERE child_asset_id = X AND relation_type = 'revision_of'`。
**首版**（无 `revision_of` 边）= 无 revision_reason，自然合理。

---

## 3.5 逻辑资产聚合：`logical_assets`（rev.11 DDL）

> 同一逻辑资产的多个 revision 通过 `assets.logical_asset_id` 聚合到此表。**PRD §8 给设计要点，本节是 schema SoT**。

```sql
CREATE TABLE logical_assets (
  logical_asset_id    TEXT PRIMARY KEY CHECK (logical_asset_id ~ '^[0-9A-Za-z]{8}$'),
  asset_type          TEXT NOT NULL,                  -- 整组类型固定（LA1：首版决定不可改）
  display_name        TEXT,
  description         TEXT,
  owner               TEXT,
  status              TEXT NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active','archived')),
  -- rev.11：砍掉 current_asset_id 冗余指针。
  -- "当前版本"通过查询获得：SELECT asset_id FROM assets WHERE logical_asset_id=? AND is_current=true
  -- partial unique index uq_assets_current_per_logical 保证唯一性，无需应用层 Validator。
  current_revision    BIGINT NOT NULL DEFAULT 1
                      CHECK (current_revision >= 1),
  total_revisions     BIGINT NOT NULL DEFAULT 1
                      CHECK (total_revisions >= current_revision),    -- LA2
  metadata            JSONB NOT NULL DEFAULT '{}',
  extra               JSONB NOT NULL DEFAULT '{}',    -- §16.7 schema agility 实验区
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  row_version         BIGINT NOT NULL DEFAULT 1       -- 乐观锁
);

CREATE INDEX idx_lassets_owner   ON logical_assets(owner)        WHERE owner IS NOT NULL;
CREATE INDEX idx_lassets_type    ON logical_assets(asset_type, status);
CREATE INDEX idx_lassets_status  ON logical_assets(status) WHERE status <> 'active';
CREATE INDEX idx_lassets_updated ON logical_assets(updated_at DESC);

-- assets 表反向 FK
ALTER TABLE assets ADD CONSTRAINT fk_assets_logical
  FOREIGN KEY (logical_asset_id) REFERENCES logical_assets(logical_asset_id)
  DEFERRABLE INITIALLY DEFERRED;

-- LA1: asset_type 首版决定不可改（PG trigger）
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

### 3.5.1 同步逻辑（AssetWriter）

**首版创建：**

```sql
BEGIN;
  -- 1. 先 INSERT assets（logical_asset_id = asset_id 自引用）
  INSERT INTO assets (asset_id, asset_type, logical_asset_id, revision, is_current, ...)
  VALUES ('clipA_v1', 'clip', 'clipA_v1', 1, true, ...);

  -- 2. 再 INSERT logical_assets（无 current_asset_id，靠查询）
  INSERT INTO logical_assets (logical_asset_id, asset_type, current_revision, total_revisions, ...)
  VALUES ('clipA_v1', 'clip', 1, 1, ...);

  -- FK 在 COMMIT 时统一校验
COMMIT;
```

**B 路由（生新版本，只需更新 2 个计数器）：**

```sql
BEGIN;
  -- 1. INSERT 新 assets 行
  INSERT INTO assets (asset_id, asset_type, logical_asset_id, revision, is_current, ...)
  VALUES ('clipA_v2', 'clip', 'clipA_v1', 2, true, ...);

  -- 2. 老版下线（乐观锁）
  UPDATE assets SET is_current = false, row_version = row_version + 1, updated_at = now()
  WHERE asset_id = 'clipA_v1' AND row_version = $parent_row_version;
  -- 若 0 行 → 409

  -- 3. logical_assets 只更新计数器（current_asset_id 已砍，无需同步）
  UPDATE logical_assets SET
    current_revision = 2,
    total_revisions  = total_revisions + 1,
    updated_at       = now()
  WHERE logical_asset_id = 'clipA_v1';
COMMIT;
```

### 3.5.2 不变式速查

| # | 规则 | enforce 位置 |
|---|------|-------------|
| **LA1** | `asset_type` 不可 UPDATE | PG trigger（`trg_logical_assets_type_immutable`）|
| **LA2** | `current_revision <= total_revisions` | CHECK 约束 |

> ~~LA3~~ / ~~LA4~~ 已消除（rev.11 砍掉 `current_asset_id` 冗余指针后自然消失）。

### 3.5.3 查询当前版本

```sql
-- 查某逻辑资产的当前版本 asset_id
SELECT asset_id FROM assets
WHERE logical_asset_id = $1 AND is_current = TRUE AND is_deleted = FALSE;
-- 走 partial unique index uq_assets_current_per_logical，O(1)

-- 批量查所有逻辑资产的当前版本
SELECT DISTINCT ON (a.logical_asset_id) a.logical_asset_id, a.asset_id
FROM assets a
WHERE a.is_current = TRUE AND a.is_deleted = FALSE
ORDER BY a.logical_asset_id, a.created_at DESC;
```

---

## 4. 四张通用 Aspect 表

### 4.1 结构血缘 + 时间维度：`asset_lineage`（合并版）

> rev.2 决策：原 `asset_lineage` + `asset_temporal` 合并为一张表。理由：时间窗在创建时确定后**几乎不改**（B 路由生新 asset_id 而非改时间），不需要独立 Aspect 的并发隔离收益。
>
> rev.10 K1 决策：`split_*` 5 个字段保持 typed 列（不压缩为 JSONB）。理由：填充率 ~60%（不算稀疏）+ 全是热查询字段（filter / facet 频繁）+ NULL 不消耗存储（PG NULL bitmap）+ Aspect 表风格统一靠 `extra JSONB` 兜底冷字段，typed 列负责热字段。

```sql
-- 表达：「我来自哪段物理 MCAP / 我是谁的孩子 / 我是哪段时间」
CREATE TABLE asset_lineage (
  asset_id               TEXT PRIMARY KEY REFERENCES assets(asset_id),

  -- 结构血缘
  mcap_file_id           TEXT,    -- rev.10：derived_asset 跨多 mcap 聚合时允许 NULL；其余类型由 L8 不变式 (AssetWriter Validator) 强制 NOT NULL
  parent_asset_id        TEXT REFERENCES assets(asset_id),
  root_asset_id          TEXT REFERENCES assets(asset_id),
  asset_level            INT  NOT NULL DEFAULT 0,
  split_method           TEXT,                    -- manual|rule|algo|pipeline
  split_algo_name        TEXT,
  split_algo_version     TEXT,
  split_run_id           TEXT,
  split_reason           TEXT,
  segment_index          INT,
  parent_start_offset_ms BIGINT,
  parent_end_offset_ms   BIGINT,
  segment_locator        TEXT,

  -- 时间维度（合并自原 asset_temporal）
  start_timestamp_ns     BIGINT NOT NULL,
  end_timestamp_ns       BIGINT NOT NULL,
  duration_ns            BIGINT GENERATED ALWAYS AS
                         (end_timestamp_ns - start_timestamp_ns) STORED,    -- rev.10 G1：满精度保 ns，避免整数除法截断；API 层 / 1_000_000 转 ms 给前端

  updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),    -- rev.10 H1：所有 Aspect 表统一含 updated_at
  extra                  JSONB NOT NULL DEFAULT '{}',           -- §16.7 schema agility 实验区

  CONSTRAINT chk_lineage_temporal_range CHECK (end_timestamp_ns >= start_timestamp_ns)
);

CREATE INDEX idx_alineage_parent  ON asset_lineage (parent_asset_id) WHERE parent_asset_id IS NOT NULL;
CREATE INDEX idx_alineage_root    ON asset_lineage (root_asset_id)   WHERE root_asset_id IS NOT NULL;
CREATE INDEX idx_alineage_mcap    ON asset_lineage (mcap_file_id, asset_level);
CREATE INDEX idx_alineage_range   ON asset_lineage (start_timestamp_ns, end_timestamp_ns);
CREATE INDEX idx_alineage_updated ON asset_lineage (updated_at DESC);
```

**B 路由行为：** v1 和 v2 通常共享同一 `parent_asset_id`（都来自同一段 segment）且时间窗相同，lineage aspect 各自独立插入相同的值。极少数情况（算法把时间窗调整了）lineage 才真正不同。

### 4.2 产物内容：`asset_content`

```sql
-- 表达：「我的 GCS 产物是什么，我是 virtual 还是 materialized」
CREATE TABLE asset_content (
  asset_id        TEXT PRIMARY KEY REFERENCES assets(asset_id),
  materialization TEXT NOT NULL DEFAULT 'materialized', -- virtual|materialized
  storage_uri     TEXT,     -- 格式：assets/<logical_asset_id>/r<n>-<uuid4>/<filename>
  thumb_uri       TEXT,
  files           JSONB NOT NULL DEFAULT '{}',
  frame_kind      TEXT,     -- 仅 asset_type=frame：single|set
  frame_count     INT,
  summary_text    TEXT,     -- ES 全文索引
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),    -- rev.10 H1
  extra           JSONB NOT NULL DEFAULT '{}',           -- §16.7
  CONSTRAINT chk_virtual_uri CHECK (
    (materialization = 'virtual'  AND storage_uri IS NULL) OR
    (materialization = 'materialized')
  )
);

CREATE INDEX idx_acontent_updated     ON asset_content (updated_at DESC);
CREATE UNIQUE INDEX idx_acontent_storage_uri
                                      ON asset_content (storage_uri) WHERE storage_uri IS NOT NULL;   -- rev.10 OPT3：同 GCS 路径不可属于两个 asset；G3 双 commit 协议兜底
```

**B 路由行为：** 这是最常被 B 路由更新的 Aspect（storage_uri 变了就触发 B 路由，生新 asset_id 的同时插新 asset_content 行）。

### 4.3 治理状态：`asset_governance`

```sql
-- 表达：「我处于什么生命周期状态，归属、交付、留存信息」
CREATE TABLE asset_governance (
  asset_id          TEXT PRIMARY KEY REFERENCES assets(asset_id),
  lifecycle_state   TEXT NOT NULL DEFAULT 'ready',   -- ready|archived|superseded
  owner             TEXT,
  reviewer          TEXT,
  retention_tier    TEXT DEFAULT 'hot',              -- hot|warm|cold|archive
  expire_at         TIMESTAMPTZ,
  delivery_count    INT NOT NULL DEFAULT 0,
  last_delivered_at TIMESTAMPTZ,
  last_delivered_to TEXT,
  tenant_id         TEXT,
  project_id        TEXT,
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),    -- rev.10 H1：低频治理写入更需要 updated_at 审计
  extra             JSONB NOT NULL DEFAULT '{}'            -- §16.7
);

CREATE INDEX idx_agov_lifecycle ON asset_governance (lifecycle_state);
CREATE INDEX idx_agov_owner     ON asset_governance (owner) WHERE owner IS NOT NULL;
CREATE INDEX idx_agov_updated   ON asset_governance (updated_at DESC);
CREATE INDEX idx_agov_tenant    ON asset_governance (tenant_id, project_id) WHERE tenant_id IS NOT NULL;  -- rev.10 J1：多租户列表（P1.5+）
```

→ **低频写入**。治理变更、归属变更、交付汇总（projector 维护）。

### 4.4 使用信号：`asset_usage_stats`（P1.5 新增）

> rev.2 决策：从 `asset_governance` 拆出。理由：使用信号是**高频累加**（每次 view / download），与 governance 低频治理写入混在一张表会引发行锁竞争 —— 这正是 Entity-Aspect 模式要根除的问题。

```sql
-- 表达：「这个资产被浏览/下载/收藏多少次，最近什么时候被访问」
CREATE TABLE asset_usage_stats (
  asset_id           TEXT PRIMARY KEY REFERENCES assets(asset_id),
  view_count         INT NOT NULL DEFAULT 0,
  download_count     INT NOT NULL DEFAULT 0,
  favorited_count    INT NOT NULL DEFAULT 0,
  trending_score     REAL,                   -- projector 计算（时间衰减）
  last_viewed_at     TIMESTAMPTZ,
  last_downloaded_at TIMESTAMPTZ,
  last_favorited_at  TIMESTAMPTZ,
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ausage_trending  ON asset_usage_stats (trending_score DESC) WHERE trending_score IS NOT NULL;
CREATE INDEX idx_ausage_views     ON asset_usage_stats (view_count DESC);                                              -- rev.10 J1：发现层「最热」排行榜
CREATE INDEX idx_ausage_downloads ON asset_usage_stats (download_count DESC);
CREATE INDEX idx_ausage_favorited ON asset_usage_stats (favorited_count DESC) WHERE favorited_count > 0;
```

→ **高频写入**。`UsageStatsAccumulator` ActionHandler（§4.11 PRD）订阅 `asset_viewed / asset_downloaded / asset_favorited` events 异步累加；不阻塞主线写入。

---

## 5. 保留的现有 Aspect 表（不动）

| Aspect 表 | 语义 | 与 assets 关系 |
|-----------|------|--------------|
| `mcap_files` | raw_mcap 的物理文件事实 | `mcap_files.mcap_file_id = assets.asset_id`（asset_type=raw_mcap 时 1:1） |
| `actions` | action 的标注时间窗 + label | `actions.action_id = assets.asset_id`（asset_type=action 时 1:1） |
| `asset_tags` | 开放多源 tag（5 类源 a~e） | N:1 via asset_id |
| `asset_algo_latest` | 算法当前态（支持 pin + 多 kind 共表 + run_inputs reproducibility，详见 PRD §4.4） | N:1 via asset_id；PK `(asset_id, algo_kind, algo_name)`；含 `algo_kind ∈ {processing, qa, split, enrichment}` + `run_inputs JSONB` + `is_pinned` |
| `asset_metrics` | 标量指标投影（含 `rating.*`） | N:1 via asset_id |
| `asset_eval_results` | 评估原档 JSONB | N:1 via asset_id |
| `asset_relations` | 结构 + 版本血缘边（`split_from / contains / merged_from / derived_from / sampled_from / revision_of`） | N:N；**`revision_of` 边的 `metadata` JSONB 装版本原因**（§3.3）|
| `asset_events` | append-only 审计日志（三段式：顶层 4 + system_metadata + payload） | N:1 via asset_id |
| `logical_assets` | 逻辑资产聚合（display_name / owner / current_*） | 1:1 via logical_asset_id |

---

## 6. 兼容读取层：`asset_full` VIEW

```sql
-- 透明兼容：repos.go 的所有 SELECT 只改 FROM 目标，字段名不变
-- 注意：必须用显式列名，不能 SELECT a.*, l.*, ...（asset_id 会列名冲突）
CREATE VIEW asset_full AS
SELECT
  -- Entity 核心（assets）
  a.asset_id,
  a.asset_type,
  a.logical_asset_id,
  a.revision,
  a.is_current,
  a.is_deleted,
  a.created_at,
  a.updated_at,
  a.row_version,

  -- 结构血缘 + 时间维度（asset_lineage）
  l.mcap_file_id,
  l.parent_asset_id,
  l.root_asset_id,
  l.asset_level,
  l.split_method,
  l.split_algo_name,
  l.split_algo_version,
  l.split_run_id,
  l.split_reason,
  l.segment_index,
  l.parent_start_offset_ms,
  l.parent_end_offset_ms,
  l.segment_locator,
  l.start_timestamp_ns,
  l.end_timestamp_ns,
  l.duration_ns,

  -- 产物内容（asset_content）
  c.materialization,
  c.storage_uri,
  c.thumb_uri,
  c.files,
  c.frame_kind,
  c.frame_count,
  c.summary_text,

  -- 治理状态（asset_governance）
  g.lifecycle_state,
  g.owner,
  g.reviewer,
  g.retention_tier,
  g.expire_at,
  g.delivery_count,
  g.last_delivered_at,
  g.last_delivered_to,
  g.tenant_id,
  g.project_id,

  -- 使用信号（asset_usage_stats，P1.5 后存在）
  u.view_count,
  u.download_count,
  u.favorited_count,
  u.trending_score,
  u.last_viewed_at,
  u.last_downloaded_at,
  u.last_favorited_at

FROM assets a
LEFT JOIN asset_lineage      l ON l.asset_id = a.asset_id
LEFT JOIN asset_content      c ON c.asset_id = a.asset_id
LEFT JOIN asset_governance   g ON g.asset_id = a.asset_id
LEFT JOIN asset_usage_stats  u ON u.asset_id = a.asset_id
WHERE a.is_deleted = FALSE;
```

### 6.1 读路径示例

```go
// 改前：FROM assets
const q = `SELECT asset_id, mcap_file_id, ... FROM assets WHERE asset_id = $1`

// 改后：FROM asset_full（字段列表完全不变）
const q = `SELECT asset_id, mcap_file_id, ... FROM asset_full WHERE asset_id = $1`
```

### 6.2 写路径（VIEW 不可写）

VIEW 是只读视图。**所有写入必须由 `AssetWriter` 拆分到 Entity 壳 + 各 Aspect 同事务**，详见 §7。

---

## 7. 写入流程（AssetWriter 拆分）

### 7.1 首版创建（Create）

```sql
BEGIN;
  -- 1. Entity 壳
  INSERT INTO assets (asset_id, asset_type, logical_asset_id, revision, is_current, ...)
  VALUES ($1, $2, $1, 1, true, ...);  -- logical_asset_id = asset_id（首版）

  -- 2. 通用 Aspect（同事务）
  INSERT INTO asset_lineage    (asset_id, mcap_file_id, parent_asset_id, start_timestamp_ns, end_timestamp_ns, ...) VALUES (...);
  INSERT INTO asset_content    (asset_id, materialization, storage_uri, ...)    VALUES (...);
  INSERT INTO asset_governance (asset_id, lifecycle_state, owner, ...)          VALUES (...);
  -- asset_usage_stats 由 UsageStatsAccumulator 首次访问时创建（lazy init）

  -- 3. 专用 Aspect（按 asset_type）
  -- raw_mcap 时：INSERT INTO mcap_files ...
  -- action 时：  INSERT INTO actions ...

  -- 4. 审计 Event
  INSERT INTO asset_events (asset_id, event_type, payload, ...) VALUES (..., 'asset_created', ...);

  -- 5. logical_assets 同步（rev.11：无 current_asset_id 字段）
  INSERT INTO logical_assets (logical_asset_id, asset_type, current_revision, total_revisions, ...)
  VALUES (?, ?, 1, 1, ...)
  ON CONFLICT (logical_asset_id) DO UPDATE SET ...;
COMMIT;
```

### 7.2 A 路由（原地更新）

```sql
-- 例：lifecycle_state 变更（只动 governance Aspect）
BEGIN;
  UPDATE asset_governance SET lifecycle_state = $2, updated_at = now()
  WHERE asset_id = $1;

  -- assets.row_version 递增（乐观锁）
  UPDATE assets SET row_version = row_version + 1, updated_at = now()
  WHERE asset_id = $1 AND row_version = $parent_row_version;
  -- 若 0 行受影响 → 409

  INSERT INTO asset_events (..., 'asset_lifecycle_changed',
    '{"before":{"lifecycle_state":"ready"},"after":{"lifecycle_state":"archived"},"changed_fields":["lifecycle_state"]}');
COMMIT;
```

```sql
-- 例：tag 新增（只动 asset_tags Aspect，assets 不参与）
BEGIN;
  INSERT INTO asset_tags (asset_id, tag_key, tag_value, source, ...) VALUES (...);
  INSERT INTO asset_events (..., 'tag_upserted', ...);
COMMIT;
-- assets 行完全不 UPDATE → 零行锁
```

```sql
-- 例：view_count 累加（UsageStatsAccumulator 异步批量；高频写专表）
BEGIN;
  INSERT INTO asset_usage_stats (asset_id, view_count, last_viewed_at, updated_at)
  VALUES ($1, 1, now(), now())
  ON CONFLICT (asset_id) DO UPDATE
    SET view_count = asset_usage_stats.view_count + EXCLUDED.view_count,
        last_viewed_at = EXCLUDED.last_viewed_at,
        updated_at = EXCLUDED.updated_at;
COMMIT;
-- assets / governance 都不动 → 零行锁
```

### 7.3 B 路由（生新版本）

```sql
BEGIN;
  -- 1. 新 Entity 壳
  INSERT INTO assets (asset_id, asset_type, logical_asset_id, revision, is_current, ...)
  VALUES ('clipA_v2', 'clip', 'L_clipA', 2, true, ...);

  -- 2. 老版 Entity 壳：下线
  UPDATE assets SET is_current = false, row_version = row_version + 1, updated_at = now()
  WHERE asset_id = 'clipA_v1' AND row_version = $parent_row_version;
  -- 若 0 行 → 409（乐观锁）

  -- 3. 新 Aspect 行
  INSERT INTO asset_lineage    ('clipA_v2', ...);  -- 继承 v1 的 parent / 时间窗
  INSERT INTO asset_content    ('clipA_v2', storage_uri='r2-f47ac10b/video.mp4', ...);  -- 变了
  INSERT INTO asset_governance ('clipA_v2', lifecycle_state='ready', ...);

  -- 4. 版本边（带 revision_reason 在 metadata 里）
  INSERT INTO asset_relations (
    parent_asset_id, child_asset_id, relation_type, metadata
  ) VALUES (
    'clipA_v2', 'clipA_v1', 'revision_of',
    '{"type":"algo_rerun","algo_name":"hand_track","algo_version":"2.0","run_id":"R789"}'::jsonb
  );

  -- 5. 审计 Event
  INSERT INTO asset_events (..., 'asset_revised', '{
    "changed_fields": ["storage_uri", "revision"],
    "before": {"asset_id": "clipA_v1", "revision": 1, ...},
    "after":  {"asset_id": "clipA_v2", "revision": 2, ...}
  }');
  -- system_metadata 也带 run_id / algo_*（详见 PRD §4.9.4）

  -- 6. logical_assets 只更新计数器（rev.11：current_asset_id 已砍）
  UPDATE logical_assets SET
    current_revision = 2,
    total_revisions  = total_revisions + 1,
    updated_at       = now()
  WHERE logical_asset_id = 'clipA_v1';

  -- 7. 老版 governance 更新
  UPDATE asset_governance SET lifecycle_state='superseded' WHERE asset_id='clipA_v1';
COMMIT;
```

---

## 8. 完整 Aspect 关系图

```text
assets（Entity 壳，9 列）
   │
   ├──► asset_lineage         ← 我从哪来 + 我是哪段时间（合并 temporal）
   ├──► asset_content         ← 我的 GCS 产物 + frame 形态
   ├──► asset_governance      ← 我的治理状态、归属、交付汇总
   ├──► asset_usage_stats     ← 高频使用信号（独立避免行锁，P1.5）
   │
   ├──► mcap_files            ← raw_mcap 物理文件（专用类型 Aspect，已有）
   ├──► actions               ← action 标注（专用类型 Aspect，已有）
   │
   ├──► asset_tags            ← 开放多源标签（5 类 source，多行）
   ├──► asset_algo_latest     ← 算法当前态（多行，per (algo_kind, algo_name) + pin + run_inputs）
   ├──► asset_metrics         ← 标量指标（含 rating.*）
   ├──► asset_eval_results    ← 评估原档（不进 ES）
   │
   ├──► asset_relations       ← 结构边 + 版本边（revision_of.metadata 装版本原因）
   ├──► asset_events          ← append-only 审计日志（三段式）
   │
   ├──► delivery_items        ← 交付明细
   └──► logical_assets        ← 逻辑资产聚合（display_name / current_*）
```

---

## 9. 迁移策略（dev 阶段，一次性，rev.10 完整版）

> **关键约定**：以下 INSERT 语句**不**列 `duration_ns` / `updated_at` / `extra` —— 它们是 GENERATED 列或 DEFAULT 列，PG 自动填，**显式 INSERT 反而会报错**。

### 9.1 步骤 1：建 logical_assets（B 路由协调表）

```sql
-- 必须先于 assets 的反向 FK 创建（DEFERRABLE 让事务内顺序自由）
CREATE TABLE logical_assets (...);  -- 完整 DDL 见 §3.5
```

### 9.2 步骤 2：assets 主表瘦身改造

```sql
-- 改类型 + 改名 + partial unique（rev.10 E2 + F1 + §3.1）
ALTER TABLE assets RENAME COLUMN version  TO row_version;      -- 与业务 revision 区分
ALTER TABLE assets ALTER  COLUMN revision TYPE BIGINT;          -- E2：与 row_version / event_seq 类型统一
-- assets.revision_reason 不创建（信息走 asset_relations.metadata + asset_events.system_metadata）

CREATE UNIQUE INDEX uq_assets_current_per_logical
  ON assets (logical_asset_id) WHERE is_current = TRUE AND is_deleted = FALSE;
```

### 9.3 步骤 3：建新 Aspect 表

```sql
-- 完整 DDL 见 §4
CREATE TABLE asset_lineage      (...);   -- 含 temporal 字段（start/end_ns + duration_ns GENERATED）+ updated_at + extra
CREATE TABLE asset_content      (...);   -- + updated_at + extra
CREATE TABLE asset_governance   (...);   -- + updated_at + extra
CREATE TABLE asset_usage_stats  (...);   -- P1.5 新增（P1 可先建表）
```

### 9.4 步骤 4：从现有 assets 回填 Aspect 数据

**注意：以下 INSERT 不列以下列**（PG 自动处理）：
- `duration_ns`（GENERATED ALWAYS AS）
- `updated_at`（DEFAULT now()）
- `extra`（DEFAULT '{}'）

```sql
-- asset_lineage（注：duration_ns 是 GENERATED 列，不在 INSERT 列表）
INSERT INTO asset_lineage (
  asset_id, mcap_file_id, parent_asset_id, root_asset_id, asset_level,
  split_method, split_algo_name, split_algo_version, split_run_id, split_reason,
  segment_index, parent_start_offset_ms, parent_end_offset_ms, segment_locator,
  start_timestamp_ns, end_timestamp_ns
)
SELECT asset_id, mcap_file_id, parent_asset_id, root_asset_id, asset_level,
       split_method, split_algo_name, split_algo_version, split_run_id, split_reason,
       segment_index, parent_start_offset_ms, parent_end_offset_ms, segment_locator,
       start_timestamp_ns, end_timestamp_ns
FROM assets;
-- duration_ns 由 PG 自动计算；updated_at / extra 走 DEFAULT

-- asset_content
INSERT INTO asset_content (
  asset_id, materialization, storage_uri, thumb_uri, files, frame_kind, frame_count, summary_text
)
SELECT asset_id,
       COALESCE(materialization, 'materialized'),
       storage_uri, thumb_uri, files, frame_kind, frame_count, summary_text
FROM assets;

-- asset_governance
INSERT INTO asset_governance (
  asset_id, lifecycle_state, owner, reviewer, retention_tier, expire_at,
  delivery_count, last_delivered_at, last_delivered_to, tenant_id, project_id
)
SELECT asset_id, COALESCE(lifecycle_state,'ready'),
       owner, reviewer, retention_tier, expire_at,
       delivery_count, last_delivered_at, last_delivered_to, tenant_id, project_id
FROM assets;

-- asset_usage_stats 不需要回填（计数从 0 起）

-- logical_assets 回填（每个 asset 一行；asset_id == logical_asset_id 自引用，首版）
-- rev.11：无 current_asset_id 列
INSERT INTO logical_assets (
  logical_asset_id, asset_type, current_revision, total_revisions, owner, status
)
SELECT asset_id, asset_type, 1, 1, owner,
       CASE WHEN lifecycle_state IN ('archived','superseded') THEN 'archived' ELSE 'active' END
FROM assets;
```

### 9.5 步骤 5：建 asset_full VIEW

```sql
CREATE VIEW asset_full AS (...);  -- 完整定义见 §6（显式列名，含全 Aspect 字段）
```

### 9.6 步骤 6：revision_reason 迁移（从老 assets 到 asset_relations）

```sql
-- 如果旧 schema 有 assets.revision_reason 列，迁到 revision_of 边 metadata
INSERT INTO asset_relations (parent_asset_id, child_asset_id, relation_type, metadata)
SELECT a2.asset_id AS parent_asset_id,           -- 新版
       a1.asset_id AS child_asset_id,            -- 老版
       'revision_of',
       jsonb_build_object('type', 'migrated_from_assets_column', 'reason', a2.revision_reason)
FROM assets a1
JOIN assets a2 ON a2.logical_asset_id = a1.logical_asset_id
              AND a2.revision = a1.revision + 1
WHERE a2.revision_reason IS NOT NULL;
```

### 9.7 步骤 7：assets 表删冗余列（迁完代码后）

```sql
-- 等 repos.go 全部 FROM assets → FROM asset_full 迁完，handler 验证通过后执行
ALTER TABLE assets DROP COLUMN mcap_file_id;
ALTER TABLE assets DROP COLUMN parent_asset_id;
ALTER TABLE assets DROP COLUMN root_asset_id;
ALTER TABLE assets DROP COLUMN asset_level;
ALTER TABLE assets DROP COLUMN split_method;
ALTER TABLE assets DROP COLUMN split_algo_name;
ALTER TABLE assets DROP COLUMN split_algo_version;
ALTER TABLE assets DROP COLUMN split_run_id;
ALTER TABLE assets DROP COLUMN split_reason;
ALTER TABLE assets DROP COLUMN segment_index;
ALTER TABLE assets DROP COLUMN parent_start_offset_ms;
ALTER TABLE assets DROP COLUMN parent_end_offset_ms;
ALTER TABLE assets DROP COLUMN segment_locator;
ALTER TABLE assets DROP COLUMN start_timestamp_ns;
ALTER TABLE assets DROP COLUMN end_timestamp_ns;
ALTER TABLE assets DROP COLUMN duration_ms;       -- 老 ms 列彻底删，仅保 asset_lineage.duration_ns
ALTER TABLE assets DROP COLUMN materialization;
ALTER TABLE assets DROP COLUMN storage_uri;
ALTER TABLE assets DROP COLUMN thumb_uri;
ALTER TABLE assets DROP COLUMN files;
ALTER TABLE assets DROP COLUMN frame_kind;
ALTER TABLE assets DROP COLUMN frame_count;
ALTER TABLE assets DROP COLUMN summary_text;
ALTER TABLE assets DROP COLUMN lifecycle_state;
ALTER TABLE assets DROP COLUMN owner;
ALTER TABLE assets DROP COLUMN reviewer;
ALTER TABLE assets DROP COLUMN retention_tier;
ALTER TABLE assets DROP COLUMN expire_at;
ALTER TABLE assets DROP COLUMN delivery_count;
ALTER TABLE assets DROP COLUMN last_delivered_at;
ALTER TABLE assets DROP COLUMN last_delivered_to;
ALTER TABLE assets DROP COLUMN tenant_id;
ALTER TABLE assets DROP COLUMN project_id;
ALTER TABLE assets DROP COLUMN metadata;
ALTER TABLE assets DROP COLUMN algo_inputs_uris;
ALTER TABLE assets DROP COLUMN annot_inputs_uris;
ALTER TABLE assets DROP COLUMN revision_reason;   -- 信息已迁移到 asset_relations.metadata + asset_events
```

### 9.8 迁移节奏

| 步骤 | 何时执行 | 影响 |
|------|---------|------|
| 9.1–9.5 | 一次 PR（dev 当晚）| 建表 + 回填 + VIEW；老 assets 表所有列保留，**老代码继续工作** |
| 9.6 | 同上事务 | revision_reason 一次性迁完 |
| 代码迁移 | 分批 PR | `repos.go` / `usecase` `FROM assets` → `FROM asset_full`；写路径走 AssetWriter 拆分 |
| **9.7** | **代码全部迁完，监控 1 周无回滚后** | DROP COLUMN 不可逆；建议先 `RENAME COLUMN xxx TO xxx_deprecated` 保留 1 周再删 |
| 索引 / Lint | 持续 | RL1 VIEW lint + RL2 OrphanAssertion + AS1 updated_at AssetWriter 强制 |

**迁移期间安全网：**
- 步骤 9.7 前**所有列双写**：`AssetWriter` 同时写老 `assets` 列和新 `asset_*` Aspect 表（双写期 ≤ 1 周）
- 步骤 9.7 前**读路径双校验**：`asset_full` VIEW 字段 vs `assets` 老字段，任一差异 → metric 告警 + ops 人工对账
- 步骤 9.7 后**unreachable code 清理**：删 `AssetWriter` 老列写入分支

---

## 10. `queries/run` 影响分析

| 路径 | 现状 | 迁移后 | 影响 |
|------|------|--------|------|
| ES 主路径（99% 请求） | 走 ES，完全不碰 PG assets | 不变（ES doc 从 asset_full VIEW 构建） | **零影响** |
| PG fallback 过滤 | `WHERE assets.start_timestamp_ns > X` | `WHERE asset_full.start_timestamp_ns > X` | 改 FROM 目标，字段名不变 |
| 点查（GET /assets/{id}） | `FROM assets` | `FROM asset_full` | 同上 |

---

## 11. 性能评估

| 操作 | 现状 | Entity-Aspect 后 |
|------|------|----------------|
| 点查 1 个资产 | 1 个 30 列 SELECT | 1 个 VIEW（4 LEFT JOIN），PG 通常优化为 nested loop = 快 |
| 写 tag | UPDATE assets + INSERT tag | INSERT tag 只（assets 行不 UPDATE）→ **更快** |
| 写 algo 结果 | UPDATE assets + upsert algo_latest | upsert algo_latest 只 → **更快** |
| view_count 累加 | UPDATE assets（与 lifecycle 竞争锁）| UPSERT usage_stats（独立表）→ **更快 + 无竞争** |
| B 路由 | INSERT assets(30列) + UPDATE old | INSERT assets(9列) + 3 Aspect INSERT + UPDATE old → 相近 |
| 多算法 + 高频 view 并发 | 行锁雪崩 | **零竞争**（写不同 Aspect 表）|

---

## 附录：字段归属速查（迁移参考）

| 原 `assets` 列 | 迁移去 |
|----------------|--------|
| asset_id, asset_type | **assets（留）** |
| logical_asset_id, revision, is_current | **assets（留）** |
| is_deleted, created_at, updated_at | **assets（留）** |
| version → **row_version** | **assets（留，改名）** |
| ~~revision_reason~~ | **删除**（信息走 `asset_relations(revision_of).metadata` + `asset_events.system_metadata`） |
| mcap_file_id, parent_asset_id, root_asset_id, asset_level | → **asset_lineage** |
| split_method, split_algo_*, split_run_id, split_reason | → **asset_lineage** |
| segment_index, parent_*_offset_ms, segment_locator | → **asset_lineage** |
| start_timestamp_ns, end_timestamp_ns, ~~duration_ms~~ → **duration_ns** | → **asset_lineage**（rev.2 合并自 asset_temporal；rev.10 G1 改 ns 满精度）|
| materialization, storage_uri, thumb_uri, files, frame_kind, frame_count, summary_text | → **asset_content** |
| lifecycle_state, owner, reviewer, retention_tier, expire_at | → **asset_governance** |
| delivery_count, last_delivered_at, last_delivered_to | → **asset_governance** |
| tenant_id, project_id | → **asset_governance** |
| metadata, algo_inputs_uris, annot_inputs_uris | → 视字段决定：稳定的进相应 Aspect；通用扩展可留 `assets.metadata` JSONB（但**不推荐**：尽量提升为列）|

---

## 12. 设计权衡（弊端 + 缓解）

> 任何架构都有 trade-off。Entity-Aspect 模式带来的 5 个真实代价及对应缓解：

| 弊端 | 严重度 | 缓解 | 缓解后影响 |
|------|------|------|----------|
| **读放大**（点查从 1 张 → JOIN 5 张表） | 低 | ES 主路径（90% 列表）+ 物化视图（热点详情） | 几乎无感（点查 ~0.5-1ms，仍比业务可接受快 100x） |
| **写放大**（创建 1 个资产从 1 INSERT → 5+ INSERT） | 中 | PG WAL 顺序写性能近线性叠加；批量场景用 COPY | 单写慢 2-3x（绝对值仍 < 5ms），可接受 |
| **跨 Aspect 复杂查询**（owner + scenario + quality 要 JOIN 3 表） | 中 | 走 ES（D1 denormalize 一次拿齐），不走 PG | PG 端影响小 |
| **孤儿 Aspect 风险**（4 张表事务不齐导致主表有行、aspect 缺） | 高 ⇒ 低 | 同事务原子写（PRD AE1）+ DB FK CASCADE + 定期巡检 | 接近零 |
| **演进复杂度**（改一字段要确定在哪张 Aspect） | 中 | 字段速查表（附录）+ `asset_full` VIEW 对上层透明 + AssetWriter 封装拆分逻辑 | 一次性投入 < 2 天 |

**vs 不做 Entity-Aspect 的代价**：

| 不做的弊端 | 不解决的代价 |
|---|---|
| `assets` 字段持续累加，半年内 50+ 列 | schema 越来越难维护 |
| 多算法并发写同 asset → 行锁竞争 | 高并发下吞吐瓶颈 |
| 新字段每次都改 schema | 每次迁移有风险 |
| B 路由要复制 30+ 列 | 存储与写入冗余 |
| **后期再拆 = 推倒重构，影响所有 repo / handler / migration** | dev 阶段不做，prod 之后做 = **重大事故** |

→ **dev 阶段做 = 最经济；prod 后做 = 灾难。**

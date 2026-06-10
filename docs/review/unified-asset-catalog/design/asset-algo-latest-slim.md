# `asset_algo_latest` 精简方案

| 字段 | 值 |
|------|----|
| 状态 | Proposal（rev.13 候选） |
| 关联 | `../schema.md` §8 / `algo-runs.md` / `asset-versioning.md` |
| 决策来源 | 待评审（rev.13） |

---

## 0. TL;DR

**问题**：rev.12 引入 `algo_runs` 一等实体后，`asset_algo_latest` 有 6 个字段和 `algo_runs` 重复（`algo_version` / `started_at` / `finished_at` / `run_inputs` / `output_uri` / `method`）。多版本场景下还有「latest」语义歧义。

**方案**：保留表，**砍掉 6 个冗余字段**，只留 4 个**真核心**职责（per-asset 处理状态指针 + pin 标记）。

**效果**：表从 12+ 列降到 7 列，去重 60%，去除多版本歧义，工程量 2-3 天。

---

## 1. 现状

### 1.1 表结构（rev.12）

```sql
asset_algo_latest (
  asset_id         TEXT REFERENCES assets(asset_id),
  algo_name        TEXT NOT NULL,
  algo_version     TEXT NOT NULL,                    -- 🔴 与 algo_runs 重复
  algo_kind        TEXT NOT NULL DEFAULT 'processing', -- 🔴 rev.12 已决定单值
  status           TEXT NOT NULL,                    -- ✅ 核心：per-asset 状态
  run_id           TEXT REFERENCES algo_runs(run_id),-- ✅ 核心：指向最近 run
  started_at       TIMESTAMPTZ,                       -- 🔴 algo_runs 已有
  finished_at      TIMESTAMPTZ,                       -- 🔴 algo_runs 已有
  method           TEXT,                              -- 🔴 algo_runs 已有
  output_uri       TEXT,                              -- 🔴 assets.storage_uri 是事实源
  result_summary   JSONB DEFAULT '{}',                -- 🔴 asset_metrics / asset_eval_results 各管各的
  is_pinned        BOOL NOT NULL DEFAULT false,       -- ✅ 核心：pin 标记
  pinned_at        TIMESTAMPTZ,                       -- ✅ pin 元信息
  pinned_by        TEXT,                              -- ✅ pin 元信息
  run_inputs       JSONB NOT NULL DEFAULT '{}',       -- 🔴 algo_runs.run_inputs 已有
  metadata         JSONB DEFAULT '{}',
  extra            JSONB DEFAULT '{}',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (asset_id, algo_name)
);
```

### 1.2 4 个真核心职责

`asset_algo_latest` **不能砍**的理由：

| 职责 | 哪个字段提供 | 替代方案吗？ |
|---|---|---|
| **per-(asset, algo) 最新跑次状态指针** | `run_id` + `status` | 只能 JOIN 重建，慢 5-10x |
| **pin 标记** | `is_pinned` / `pinned_at` / `pinned_by` | 没有同等替代（业务标记） |
| **运行中状态占位**（pending/running） | `status` | events 流可推但散，不适合热查询 |
| **per-asset 处理覆盖率快查** | PK lookup | 视图查询多表 JOIN，热路径瓶颈 |

### 1.3 6 个真冗余字段

| 字段 | 哪里有同等信息 | 砍掉成本 |
|---|---|---|
| `algo_version` | `algo_runs.algo_version`（JOIN 即得） | 改查询 SQL |
| `algo_kind` | rev.12 决定单值 `processing`，无需存 | DROP COLUMN 即可 |
| `started_at` | `algo_runs.started_at` | JOIN |
| `finished_at` | `algo_runs.completed_at` | JOIN |
| `method` | `algo_runs.method` | JOIN |
| `output_uri` | 事实源是 `assets.storage_uri`（产物 asset 的 URI）；非产物类（eval/qa）本来也填 NULL | 移除 |
| `run_inputs` | `algo_runs.run_inputs` 一份；per-asset 输入快照可挪 `asset_events.event_payload` | 移除 |
| `result_summary` | `asset_metrics`（数值）+ `asset_eval_results`（评估）已分流 | 移除 |

---

## 2. 改动方案

### 2.1 精简后的表结构

```sql
asset_algo_latest (
  asset_id      TEXT REFERENCES assets(asset_id),
  algo_name     TEXT NOT NULL,

  -- 最新跑次指针
  run_id        TEXT REFERENCES algo_runs(run_id) ON DELETE SET NULL,
  status        TEXT NOT NULL CHECK (status IN ('pending','running','ok','failed')),

  -- pin 标记（业务标记，非 run 产生）
  is_pinned     BOOL NOT NULL DEFAULT false,
  pinned_at     TIMESTAMPTZ,
  pinned_by     TEXT,
  pinned_reason TEXT,

  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

  PRIMARY KEY (asset_id, algo_name)
);

CREATE INDEX idx_aal_run    ON asset_algo_latest(run_id) WHERE run_id IS NOT NULL;
CREATE INDEX idx_aal_status ON asset_algo_latest(algo_name, status);
CREATE INDEX idx_aal_pinned ON asset_algo_latest(asset_id) WHERE is_pinned = true;
```

字段从 17 列降到 8 列。

### 2.2 兼容 VIEW（保留旧字段名给老查询）

老 SQL 大量用 `algo_version` / `started_at` / `finished_at` 等列。提供兼容 view 让代码不用一次全改：

```sql
CREATE VIEW asset_algo_latest_compat AS
SELECT
  aal.asset_id,
  aal.algo_name,
  r.algo_version,           -- 从 algo_runs JOIN
  'processing' AS algo_kind,-- 单值常量
  aal.status,
  aal.run_id,
  r.started_at,
  r.completed_at AS finished_at,
  r.method,
  -- output_uri 改为查实际产物 asset
  (SELECT a.storage_uri FROM assets a
    JOIN asset_relations ar ON ar.parent_asset_id = a.asset_id
    WHERE ar.child_asset_id = aal.asset_id
      AND ar.metadata->>'run_id' = aal.run_id
    LIMIT 1) AS output_uri,
  r.run_inputs,
  aal.is_pinned,
  aal.pinned_at,
  aal.pinned_by,
  aal.updated_at
FROM asset_algo_latest aal
LEFT JOIN algo_runs r ON r.run_id = aal.run_id;
```

### 2.3 ALTER 脚本

```sql
BEGIN;

-- 1. 删冗余列
ALTER TABLE asset_algo_latest
  DROP COLUMN algo_version,
  DROP COLUMN algo_kind,
  DROP COLUMN started_at,
  DROP COLUMN finished_at,
  DROP COLUMN method,
  DROP COLUMN output_uri,
  DROP COLUMN run_inputs,
  DROP COLUMN result_summary,
  DROP COLUMN metadata,
  DROP COLUMN extra,
  DROP COLUMN created_at;

-- 2. 加 pinned_reason（pin 时记录原因，便于审计）
ALTER TABLE asset_algo_latest
  ADD COLUMN pinned_reason TEXT;

-- 3. status CHECK 收紧
ALTER TABLE asset_algo_latest
  ADD CONSTRAINT chk_aal_status CHECK (status IN ('pending','running','ok','failed'));

-- 4. 兼容 view
CREATE OR REPLACE VIEW asset_algo_latest_compat AS ...;

COMMIT;
```

---

## 3. 工程量

| 任务 | 估时 |
|---|---|
| ALTER schema + 兼容 VIEW | 0.5 天 |
| `AlgoRepo.UpsertLatest` 改造（写入字段精简） | 0.5 天 |
| 调用方查询逐步迁移到新列名（或继续用 compat view 跨 sprint） | 1 天 |
| ES builder 投影字段调整（如有） | 0.5 天 |
| 文档更新 | 0.5 天 |
| 总计 | **2-3 天** |

---

## 4. 多版本下的查询规范（同期落地）

**规范**：查询「这个 logical 物当前版的算法状态」**永远** JOIN `assets WHERE is_current=true`：

```sql
-- ❌ 错误：可能拿到老版本的状态
SELECT * FROM asset_algo_latest WHERE asset_id = 'bbb22222' AND algo_name = 'hand_track';

-- ✅ 正确
SELECT aal.*
FROM assets a
JOIN asset_algo_latest aal ON aal.asset_id = a.asset_id
WHERE a.logical_asset_id = 'L_clipX'
  AND a.is_current = true
  AND aal.algo_name = 'hand_track';
```

**B 路由升版后的语义**：新 asset_id 没有 `asset_algo_latest` 行，**查不到 = 未处理**。不做自动迁移。

---

## 5. 替代方案（评估过被否）

### 方案 B：拆分 `asset_pins` + DROP `asset_algo_latest`

把 pin 抽成独立表，状态查询走 view。

**否决理由**：
- view 查询比单表 PK 慢 5-10x，热路径成瓶颈
- pending/running 状态无处安放（view 只反映已完成 run）
- 现网代码改造量是方案 A 的 2-3 倍

### 方案 C：完全砍掉

把 pin 塞 `assets.metadata` JSONB，状态推 events 流。

**否决理由**：
- pin 失去索引，「找所有 pinned asset」要扫全表
- events 流查最新状态贵且散
- 现网代码全部改写，工程量 5+ 天

---

## 6. 决策点 / 待评审

1. **是否保留兼容 view `asset_algo_latest_compat`？** 保留可分阶段迁移调用方，不保留则要一次性改完所有 SQL
2. **`pinned_reason` 是否必填？** 建议必填（审计 + 后续可解 pin 时知道原因）
3. **`status` 状态机是否扩展 cancelled？** 建议加 `cancelled`（运营手动取消跑次）
4. **ALTER 是否需要灰度？** dev/staging 直接 ALTER；prod 若有真数据走标准 7 步 migration

---

## 7. 一句话总结

> rev.12 加了 `algo_runs` 后 `asset_algo_latest` **6 个字段就是冗余**，砍掉只留**最新跑次指针 + pin 标记**两个核心职责。表从 17 列降到 8 列，多版本查询规范配套落地，工程量 2-3 天。**不砍表**（pin 和热路径快查没替代），但**精简到只剩骨头**。

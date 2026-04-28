# 算法处理生命周期与数据模型设计

> **架构基线说明**
> 本文档当前对齐主线：**`asset_algo_latest`（投影表）+ `asset_events`（统一事件 / outbox）**。
>
> Phase 0 历史形态（在 `assets.cf_algo` JSONB 内嵌算法状态、独立 `asset_algo_events` 表）目前仍可读、仍可写，但**仅作为兼容路径保留**；新代码应直接读写投影表 + 事件表。详见附录 A。
>
> 这份文档与 `data-platform-design.md §5.2.6 / §5.2.7 / §5.6.2`、`schema-reference.md` 同口径。

---

## 一、核心设计决策

### 1.1 算法当前态投影到 `asset_algo_latest`

每个 `(asset_id, algo_name, algo_version)` 三元组在 `asset_algo_latest` 占一行，存放该算法在该资产上的**最新状态**（运行中 / 成功 / 失败）。

```sql
-- 字段简表（完整定义见 schema-reference.md）
asset_algo_latest (
    asset_id        UUID    NOT NULL,
    algo_name       TEXT    NOT NULL,
    algo_version    TEXT    NOT NULL,
    status          TEXT    NOT NULL,    -- blocked / pending / running / ok / failed
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    method          TEXT,                -- ray_batch / local / ...
    dagster_run_id  TEXT,
    result_uri      TEXT,                -- 结果文件路径（GCS / S3 / 本地）
    error_message   TEXT,
    result_meta     JSONB,
    updated_at      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (asset_id, algo_name, algo_version)
);
```

**状态机**：

```
blocked → pending → running → ok
                  → failed → pending  (重试)
```

| 字段 | 必填条件 | 说明 |
|------|----------|------|
| `status` | 全程 | `blocked` / `pending` / `running` / `ok` / `failed` |
| `started_at` | running 时写入 | 开始处理时间 |
| `finished_at` | ok / failed 时写入 | 完成时间 |
| `method` | start 时写入 | 处理方法标识（`ray_batch` / `local` 等） |
| `dagster_run_id` | 可选 | Dagster run id，用于 lineage 追溯 |
| `result_uri` | ok 时必填 | 结果文件路径 |
| `error_message` | failed 时必填 | 错误信息 |
| `result_meta` | 可选 | 算法特定元数据（帧数、置信度等） |

> 在 1.0 / Phase 0 兼容期内，相同信息会**同事务双写**到 `assets.cf_algo` JSONB（key 形如 `hand_tracking@1.2.0:status`）；新代码不应直接读 `cf_algo`。

### 1.2 状态变更同事务追加 `asset_events`

任意一次算法状态变更必须在**同一个事务**里：
1. upsert `asset_algo_latest` 当前态行；
2. append 一条 `asset_events` 记录（语义事件 / outbox 起点）。

```sql
asset_events (
    event_seq               BIGSERIAL UNIQUE,        -- outbox 顺序号
    event_id                UUID PRIMARY KEY,
    asset_id                UUID NOT NULL,
    event_type              TEXT NOT NULL,           -- algo.started / algo.finished / algo.failed / algo.reset / ...
    payload                 JSONB NOT NULL,
    payload_schema_version  INT NOT NULL,
    actor                   TEXT,
    request_id              TEXT,
    created_at              TIMESTAMPTZ NOT NULL,
    published_at            TIMESTAMPTZ              -- outbox worker 标记发布完成
);
```

`event_type` 与算法相关的命名空间使用 `algo.*` 前缀：

| event_type | 触发条件 | payload 关键字段 |
|------------|----------|------------------|
| `algo.started` | `start_algo` 调用，`pending → running` | `algo_name`, `algo_version`, `method`, `dagster_run_id` |
| `algo.finished` | `finish_algo(status=ok)`，`running → ok` | `result_uri`, `result_meta` |
| `algo.failed` | `finish_algo(status=failed)`，`running → failed` | `error_message` |
| `algo.reset` | `reset_algo`，`failed/ok → pending` | `reason` |

**为什么不再单独建 `asset_algo_events` 表**：算法事件、tag 事件、QA 事件、生命周期事件全部走统一 `asset_events`，下游 outbox / ES 同步 / Dagster 入湖 / 审计 / 回放只对接一个表，不需要按 event 类型扇出。完整设计见 `data-platform-design.md §5.2.7 / §5.6.2`。

### 1.3 为什么用"投影表 + 事件表"而不是步骤表

cyber-grace 用独立的 `video_steps` 表管理处理步骤，适合视频级（万级）场景。我们的场景是 Segment 级（百万到千万级），每个 Segment 跑 5–10 个算法：

- **独立步骤表**：1000 万 Segment × 10 算法 = 1 亿行，每跑一次状态变更都要新建一行，写放大严重。
- **投影表 + 事件表**：当前态在 `asset_algo_latest`（每个 (asset, algo, version) 只一行），历史轨迹在 `asset_events`（追加写）。当前态查询走 PK / 二级索引，不需要 GROUP BY 取最新；历史 / 审计 / 回放走事件表，可分区、可冷转。

这是经典的 "current state table + event log" 拆分，避免 cf_algo JSONB 的两个老问题：
1. **JSONB key 形如 `hand_tracking@1.2.0:status`，无法做有效组合索引**——投影表用 `(algo_name, status)` 普通 B-tree 即可；
2. **历史轨迹和当前态混在一行里互相影响 vacuum / TOAST**——拆开后写当前态轻、写事件表也是顺序 append。

---

## 二、算法注册表

用 YAML 配置文件定义合法的算法和版本，平台启动时加载到内存，业务层在 `start_algo / finish_algo` 调用时校验。

```yaml
# 算法注册表（YAML，由平台维护）
algorithms:
  hand_tracking:
    description: "手部追踪"
    versions: ["1.0.0", "1.2.0"]
    output:
      required_fields: ["result_uri", "type"]
      uri_required: true

  head_tracking:
    description: "头部追踪"
    versions: ["1.0.0"]
    output:
      required_fields: ["result_uri", "type"]
      uri_required: true

  deface:
    description: "去人脸"
    versions: ["2.0.0"]
    output:
      required_fields: ["result_uri", "width", "height", "fps"]
      uri_required: true

  env_analysis:
    description: "AI 场景分析"
    versions: ["1.0.0"]
    output:
      required_fields: []
      uri_required: false
```

**加新算法**：编辑 YAML → 重启服务（或热加载），不需要改代码、重新编译。

**验证逻辑**：
- `start_algo`：检查 `algo_name@algo_version` 是否在注册表中。
- `finish_algo(ok)`：检查 `result_uri` / `required_fields` 是否齐全。
- `finish_algo(failed)`：检查 `error_message` 非空。

---

## 三、API 设计

> API endpoint 不变；变化的是后端实现：以前写 `cf_algo` JSONB key，现在写 `asset_algo_latest` 一行 + 追加一条 `asset_events`。SDK 调用方无感知。

### 3.1 开始处理

```
POST /api/v1/assets/:id/algo/:algo_key/start
```

`:algo_key = <name>@<version>`，例如 `hand_tracking@1.2.0`。

```json
{
  "method": "ray_batch",
  "dagster_run_id": "abc-123-def"
}
```

行为（同一事务内）：
1. 校验 `algo_key` 在注册表中、版本合法；
2. upsert `asset_algo_latest`：`status = running`，写 `started_at / method / dagster_run_id`；
3. append `asset_events(event_type='algo.started', payload={...})`；
4. 兼容期同步写 `assets.cf_algo` 对应键；
5. 返回 200。

### 3.2 完成处理（成功）

```
POST /api/v1/assets/:id/algo/:algo_key/finish
```

```json
{
  "status": "ok",
  "result_uri": "gs://bucket/results/xxx.npz",
  "dagster_run_id": "abc-123-def",
  "result_meta": { "frame_count": 3600, "confidence": 0.95 }
}
```

行为：
1. 校验 `required_fields`；
2. update `asset_algo_latest`：`status = ok`，写 `finished_at / result_uri / result_meta`；
3. append `asset_events(event_type='algo.finished', ...)`；
4. 返回 200。

### 3.3 完成处理（失败）

```
POST /api/v1/assets/:id/algo/:algo_key/finish
```

```json
{
  "status": "failed",
  "dagster_run_id": "abc-123-def",
  "error_message": "GPU OOM at frame 1234"
}
```

行为：update 投影表 `status = failed` + append `asset_events(event_type='algo.failed', ...)`。

### 3.4 重置为 pending

```
POST /api/v1/assets/:id/algo/:algo_key/reset
```

清空 `error_message / result_uri / result_meta`，`status → pending`，append `asset_events(event_type='algo.reset', ...)`。

### 3.5 查询算法事件历史

```
GET /api/v1/assets/:id/events?event_type=algo.*&algo_key=hand_tracking@1.2.0
```

直接查 `asset_events`，按 `created_at` 倒序。

### 3.6 批量查询待处理资产（Dagster sensor 用）

```
GET /api/v1/assets?algo=hand_tracking@1.2.0&algo_status=pending&page=1&page_size=100
```

后端用 `asset_algo_latest` 的二级索引 `(algo_name, algo_version, status)` 直接过滤；不再依赖 `cf_algo` JSONB GIN。

---

## 四、Dagster 集成

Dagster 的 sensor 和 asset 通过 SDK 调用上述 API：

```python
# sensor: 发现待处理的 Segment
pending_assets = client.assets.list(
    algo="hand_tracking@1.2.0",
    algo_status="pending",
    page_size=100,
)

# asset: 开始处理
client.assets.start_algo(
    asset_id=asset_id,
    algo_key="hand_tracking@1.2.0",
    method="ray_batch",
    dagster_run_id=context.run_id,
)

# asset: 处理完成
client.assets.finish_algo(
    asset_id=asset_id,
    algo_key="hand_tracking@1.2.0",
    status="ok",
    result_uri="gs://bucket/results/xxx.npz",
    dagster_run_id=context.run_id,
    result_meta={"frame_count": 3600},
)
```

**Lineage 追溯**：
- 从 asset 反查 Dagster：`asset_algo_latest.dagster_run_id` 或 `asset_events.payload->>'dagster_run_id'`；
- 从 Dagster 正查 asset：通过 Dagster asset materialization metadata。

---

## 五、规模与存储路径

| 阶段 | Segment 规模 | 主库形态 | 检索 / 同步 |
|------|--------------|----------|--------------|
| **1.0（当前）** | < 100 万 | PostgreSQL 单库 + JSONB 过渡列 | 无 ES / 无湖仓，列表筛选直接走 PG |
| 2.0 | 100 万 – 1000 万 | 同 PG，热字段提升为真实列 + `asset_tags / asset_algo_latest / asset_events` 投影 | ES 经 outbox + Go worker 同步；湖仓由 Dagster 按事件序号增量入 Bronze |
| 3.0 | 1000 万 – 1 亿 | PG 主库 + Iceberg 历史 + 多模态 lake（Lance） | ES + 列存湖仓 + Trino 联邦 |
| 3.1+ | > 1 亿 | 平台 Catalog（catalog_objects + catalog_object_versions）抽象多源 | 联邦检索、血缘、生命周期托管统一对外 |

> 1.0 现阶段 **不引入** Bigtable / HBase / Lindorm / Spanner，主库统一为 PostgreSQL（详见 `data-platform-design.md §5.4` 主库选型）。

---

## 六、资产管理的完整模型（1.0 → 2.0 目标）

```
┌─────────────────────────────────────────────────────────────────────┐
│                    资产管理模型（2.0 目标态）                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  mcap_files (原始文件当前态)                                         │
│   ├─ 标量列: gcs_path / size_bytes / raw_hash_md5 / device_id ...   │
│   └─ ingest_state, cf_meta(JSONB 兼容)                              │
│         │                                                           │
│         │ 1:N                                                       │
│         ▼                                                           │
│  assets (Segment 级资产当前态)                                       │
│   ├─ 标量列: lifecycle_state / asset_type / duration_ms / qa_state  │
│   ├─ cf_meta / cf_algo / cf_tag (JSONB 兼容，过渡期保留)             │
│   └─ version (OCC)                                                  │
│         │                                                           │
│         ├──► asset_tags (tag 当前态投影；M:N，PK = asset+key)         │
│         ├──► asset_algo_latest (算法当前态投影；PK = asset+algo+ver) │
│         └──► asset_events (统一事件 / outbox; append-only)           │
│                                                                     │
│         │ M:N (delivery_items)                                       │
│         ▼                                                           │
│  deliveries (交付批次当前态)                                         │
│   └─ 标量列: contract_id / customer / status / sealed_at            │
│                                                                     │
│  datasets / dataset_snapshots / training_runs (Phase 2)              │
│  catalog_objects / catalog_object_versions (Phase 3)                 │
│                                                                     │
│  idempotency_keys ((scope, idem_key))                                │
└─────────────────────────────────────────────────────────────────────┘
```

### 待建设（按优先级）

1. **`asset_algo_latest` 投影表**：与 `cf_algo` 双写，作为下一步切换主路径的前置。
2. **`asset_events` outbox + Go worker**：替换 Dagster 60s 轮询。
3. **Dataset / DatasetSnapshot 实体**：筛选 → review → seal → 交付。
4. **MCAP 解析**：用 foxglove/mcap Go 库提取文件摘要写入 `mcap_files` 标量列。

### 显式不做（1.0 / 2.0 阶段）

- 元数据湖（HBase / Lindorm）；
- 统一 Catalog（Gravitino）—— 1.0 只有一个数据源；
- 多模态向量库 / Lance —— 不做训练闭环；
- 联邦检索（Trino over PG + ES + 向量库）—— 2.0 内 PG + ES 即可。

---

## 附录 A：Phase 0 兼容路径（`cf_algo` JSONB）

> 以下内容仅用于解释**历史代码 / 老数据**形态。新代码请勿基于此编写。

Phase 0 早期把算法状态用 key 形如 `<algo>@<version>:<field>` 直接放到 `assets.cf_algo` JSONB 内：

```json
{
  "hand_tracking@1.2.0:status": "ok",
  "hand_tracking@1.2.0:gcs_uri": "gs://bucket/results/xxx.npz",
  "hand_tracking@1.2.0:started_at": "2026-04-24T10:00:00Z",
  "hand_tracking@1.2.0:finished_at": "2026-04-24T10:05:00Z"
}
```

事件单独走一张 `asset_algo_events`：

```sql
asset_algo_events (
    event_id, asset_id, algo_key, prev_status, new_status,
    dagster_run_id, error_message, created_at
);
```

**为什么退场**：
- JSONB key 无法做组合索引，Phase 0.5 后 `(algo_name, status)` 过滤性能下降；
- 算法事件、tag 事件、生命周期事件分散在不同表，下游 outbox / ES 同步要扇出多张表；
- 当前态（cf_algo JSONB）和历史轨迹（asset_algo_events）行宽不对称，事务一致性靠应用层保证。

**迁移窗口**：
1. 双写期：`asset_algo_latest` 与 `cf_algo` 同事务写；读写仍可走 cf_algo。
2. 切读期：所有读路径切到 `asset_algo_latest`；保留 cf_algo 作回退。
3. 停写期：删除 cf_algo 写入，仅留 `asset_algo_latest + asset_events`。

# 算法处理生命周期与数据模型设计

> **架构基线说明**
> 本文档当前对齐主线：**`asset_algo_latest`（投影表）+ `asset_events`（统一事件 / outbox）**。这是 1.0 算法状态的**唯一事实源**——后端 `AlgoUsecase` 全程不再读写 `assets.cf_algo` JSONB，也不再写独立 `asset_algo_events` 表（Issue 2 修正后已彻底切走）。
>
> Phase 0 历史形态（`assets.cf_algo`、`asset_algo_events`）的表结构暂时保留以便回滚，但**只读不写**。详见附录 A。
>
> 这份文档与 `data-platform-design.md §5.2.6 / §5.2.7 / §5.3.3 / §5.6.2`、`schema-reference.md` 同口径。

---

## 一、核心设计决策

### 1.1 算法当前态投影到 `asset_algo_latest`

每个 `(asset_id, algo_name)` 二元组在 `asset_algo_latest` 占**一行**——同一个算法在一个资产上**同时只有一个版本是当前态**。`algo_version` 是普通列，参与并发安全的"单调守卫"：旧版本不会覆盖新版本（详见 §1.4）。

```sql
-- 字段简表（完整定义见 schema-reference.md / migrations/008_schema_evolution_tables.sql）
asset_algo_latest (
    asset_id        UUID    NOT NULL REFERENCES assets(asset_id),
    algo_name       TEXT    NOT NULL,
    algo_version    TEXT    NOT NULL,    -- 当前活跃版本；并发更新走单调守卫
    status          TEXT    NOT NULL,    -- blocked / pending / running / ok / failed
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    method          TEXT,                -- ray_batch / local / ...
    run_id          TEXT,                -- 外部编排器 run id，仅作 lineage 字段
    output_uri      TEXT,                -- 结果文件路径（GCS / S3 / 本地）
    error_message   TEXT,
    result_summary  JSONB NOT NULL DEFAULT '{}',  -- 算法特定元数据 + extra_fields
    updated_at      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (asset_id, algo_name)
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
| `run_id` | 可选 | 外部编排器 run id（算法 worker / k8s job），lineage + 幂等的命中键 |
| `output_uri` | ok 时按注册表要求 | 结果文件路径（registry `output.uri_required=true` 时必填） |
| `error_message` | failed 时必填 | 错误信息（来自 `FinishAlgoInput.reason`） |
| `result_summary` | 可选 | 算法特定元数据（帧数、置信度、`result_size_bytes` 等 extra_fields） |

> **不再双写 `cf_algo`**：Issue 2 修正后，`AlgoUsecase` 不再触碰 `assets.cf_algo`、不再 bump `assets.version`、不再写独立 `asset_algo_events` 表。所有算法状态读写都只走 `asset_algo_latest` + `asset_events`。

### 1.2 状态变更同事务追加 `asset_events`

任意一次算法状态变更必须在**同一个事务**里：
1. upsert `asset_algo_latest` 当前态行；
2. append 一条 `asset_events` 记录（语义事件 / outbox 起点）。

```sql
asset_events (
    event_seq               BIGSERIAL UNIQUE,        -- outbox 顺序号
    event_id                UUID PRIMARY KEY,
    asset_id                UUID NOT NULL,
    event_type              TEXT NOT NULL,           -- algo_started / algo_finished / algo_failed / algo_reset / ...
    event_payload           JSONB NOT NULL,
    payload_schema_version  TEXT NOT NULL,           -- 字符串版本，如 "1.0"
    actor                   TEXT,
    request_id              TEXT,
    created_at              TIMESTAMPTZ NOT NULL,
    published_at            TIMESTAMPTZ              -- outbox worker 标记发布完成
);
```

`event_type` 与算法相关的命名空间使用 `algo_*` 前缀：

| event_type | 触发条件 | payload 关键字段 |
|------------|----------|------------------|
| `algo_started` | `start_algo` 调用，`pending → running` | `algo_key`, `algo_name`, `algo_version`, `prev_status`, `new_status`, `run_id` |
| `algo_finished` | `finish_algo(status=ok)`，`running → ok` | `algo_key`, `prev_status`, `new_status=ok`, `run_id` |
| `algo_failed` | `finish_algo(status=failed)`，`running → failed` | `algo_key`, `prev_status`, `new_status=failed`, `run_id`, `reason` |
| `algo_reset` | `reset_algo`，`failed/ok → pending` | `algo_key`, `prev_status`, `new_status=pending` |
| `algo_unblocked` | 上游 finish-ok 触发依赖满足，`blocked → pending` | `algo_key`, `prev_status=blocked`, `new_status=pending` |

**为什么不再单独建 `asset_algo_events` 表**：算法事件、tag 事件、QA 事件、生命周期事件全部走统一 `asset_events`，下游 outbox / ES 同步 / 入湖 / 审计 / 回放只对接一个表，不需要按 event 类型扇出。完整设计见 `data-platform-design.md §5.2.7 / §5.6.2`。

### 1.4 并发安全：PK + 单调守卫，不锁 `assets`

`asset_algo_latest` 的并发写完全在投影表内部解决，不再像旧方案那样需要 CAS `assets.version`：

| 并发场景 | 处理方式 | 结果 |
|----------|----------|------|
| 同一资产、不同算法（`hand_tracking` + `env_analysis` + `head_tracking`）同时 finish | 写不同的 `(asset_id, algo_name)` 行 | 完全无冲突，互不阻塞 |
| 同一资产、同一算法、同一版本的两次 finish（重试） | `run_id` 命中投影行的 `run_id` → 直接返回 nil；否则 last-writer-wins，状态收敛 | 幂等，不放大写流量 |
| 同一资产、同一算法、不同版本（`hand_tracking@1.0.0` 旧请求迟到）| `INSERT ... ON CONFLICT DO UPDATE WHERE asset_algo_latest.algo_version <= EXCLUDED.algo_version` | 旧版本静默丢弃，新版本不会被覆盖回去 |

实现：

```sql
INSERT INTO asset_algo_latest (asset_id, algo_name, algo_version, status, ...)
VALUES (...)
ON CONFLICT (asset_id, algo_name) DO UPDATE SET
    status        = EXCLUDED.status,
    algo_version  = EXCLUDED.algo_version,
    finished_at   = EXCLUDED.finished_at,
    output_uri    = EXCLUDED.output_uri,
    ...
WHERE asset_algo_latest.algo_version <= EXCLUDED.algo_version;
```

修正前（Issue 2）的失败模式：三个算法并发 finish 同一资产 → 都拿同一个 `assets.version` 做 CAS → 只有一个成功，其余 `409 CONCURRENT_CONFLICT` → 客户端重试放大写流量。

修正后：吞吐随并发线性扩展；`assets` 行的 vacuum / MVCC 压力也降下来（算法路径不再产生 `UPDATE assets`）。

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
      required_fields: ["output_uri", "type"]
      uri_required: true

  head_tracking:
    description: "头部追踪"
    versions: ["1.0.0"]
    output:
      required_fields: ["output_uri", "type"]
      uri_required: true

  deface:
    description: "去人脸"
    versions: ["2.0.0"]
    output:
      required_fields: ["output_uri", "width", "height", "fps"]
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
- `finish_algo(ok)`：检查 `output_uri` / `required_fields` 是否齐全。
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
  "run_id": "abc-123-def"
}
```

行为（同一事务内）：
1. 校验 `algo_key` 在注册表中、版本合法；
2. upsert `asset_algo_latest`：`status = running`，写 `started_at / method / run_id`；
3. append `asset_events(event_type='algo_started', payload={...})`；
4. 返回 200。

### 3.2 完成处理（成功）

```
POST /api/v1/assets/:id/algo/:algo_key/finish
```

```json
{
  "status": "ok",
  "output_uri": "gs://bucket/results/xxx.npz",
  "run_id": "abc-123-def",
  "result_summary": { "frame_count": 3600, "confidence": 0.95 }
}
```

行为：
1. 校验 `required_fields`；
2. update `asset_algo_latest`：`status = ok`，写 `finished_at / output_uri / result_summary`；
3. append `asset_events(event_type='algo_finished', ...)`；
4. 返回 200。

### 3.3 完成处理（失败）

```
POST /api/v1/assets/:id/algo/:algo_key/finish
```

```json
{
  "status": "failed",
  "run_id": "abc-123-def",
  "error_message": "GPU OOM at frame 1234"
}
```

行为：update 投影表 `status = failed` + append `asset_events(event_type='algo_failed', ...)`。

### 3.4 重置为 pending

```
POST /api/v1/assets/:id/algo/:algo_key/reset
```

清空 `error_message / output_uri / result_summary`，`status → pending`，append `asset_events(event_type='algo_reset', ...)`。

### 3.5 查询算法事件历史

```
GET /api/v1/assets/:id/events?event_type=algo_*&algo_key=hand_tracking@1.2.0
```

直接查 `asset_events`，按 `created_at` 倒序。

### 3.6 批量查询待处理资产（外部 worker 用）

```
GET /api/v1/assets?algo=hand_tracking@1.2.0&algo_status=pending&page=1&page_size=100
```

后端用 `asset_algo_latest` 的二级索引 `(algo_name, algo_version, status)` 直接过滤；不再依赖 `cf_algo` JSONB GIN。

---

## 四、外部算法 worker 集成

平台**不绑定具体编排器**。任何算法 worker（k8s Job / Ray Cluster / 自研脚本 / 未来引入的 Temporal 等）都通过 SDK 调用上述 API 完成 start / finish 闭环。`run_id` 仅用作 lineage 字段，平台不解析其语义。

```python
# Worker 启动时：发现待处理 Segment
pending_assets = client.assets.list(
    algo="hand_tracking@1.2.0",
    algo_status="pending",
    page_size=100,
)

# 开始处理
client.assets.start_algo(
    asset_id=asset_id,
    algo_key="hand_tracking@1.2.0",
    method="ray_batch",
    run_id="my-job-2026-04-28-xxx",
)

# 处理完成
client.assets.finish_algo(
    asset_id=asset_id,
    algo_key="hand_tracking@1.2.0",
    status="ok",
    output_uri="gs://bucket/results/xxx.npz",
    run_id="my-job-2026-04-28-xxx",
    result_summary={"frame_count": 3600},
)
```

**Lineage 追溯**：
- 从 asset 反查 worker：`asset_algo_latest.run_id` 或 `asset_events.payload->>'run_id'`；
- 从 worker 正查 asset：worker 自身记录写入了哪些 `asset_id`（log / metadata），平台不强制约定。

> 1.0 / 2.0 阶段，"如何调度算法 worker"是各算法团队自管理的工程问题（k8s Job / 一次性脚本即可），平台只提供状态接口；如未来需要平台级 DAG 编排（链式触发 / 长事务重试 / lineage 可视化），独立选型 ADR（候选 Temporal / Dagster / Argo），与本设计文档解耦。

---

## 五、规模与存储路径

| 阶段 | Segment 规模 | 主库形态 | 检索 / 同步 |
|------|--------------|----------|--------------|
| **1.0（当前）** | < 100 万 | PostgreSQL 单库 + JSONB 过渡列 | 无 ES / 无湖仓，列表筛选直接走 PG |
| 2.0 | 100 万 – 1000 万 | 同 PG，热字段提升为真实列 + `asset_tags / asset_algo_latest / asset_events` 投影 | ES 经 outbox + Go worker（river）同步；湖仓由 PyIceberg CronJob 按事件序号增量入 Bronze |
| 3.0 | 1000 万 – 1 亿 | PG 主库 + Iceberg 历史 + 多模态 lake（候选） | ES + 列存湖仓 + Trino |
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
│         ├──► asset_algo_latest (算法当前态投影；PK = asset_id+algo_name) │
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

### 已落地

1. **`asset_algo_latest` 投影表**：已切换为唯一写入路径；`cf_algo` 保留为兼容/回滚路径（只读不写）。
2. **`asset_events` 统一事件表**：已建已用，作为审计事实源和 outbox 起点。

### 待建设（→ 2.0）

1. **Outbox Worker + ES / Iceberg Sink**：启用 Go Worker（30s 纯轮询），把 `asset_events` 同步到 ES 和湖仓。
2. **Dataset / DatasetSnapshot 实体**：筛选 → review → seal → 交付。
3. **MCAP 解析**：用 foxglove/mcap Go 库提取文件摘要写入 `mcap_files` 标量列。

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

**迁移状态**（当前已完成切换）：
1. ~~双写期~~：已结束——后端 `AlgoUsecase` 不再写 `cf_algo`。
2. ~~切读期~~：已完成——所有读路径已切到 `asset_algo_latest`。
3. **当前态**：`cf_algo` 列保留为只读回滚路径，投影表稳定 60 天后计划 drop。

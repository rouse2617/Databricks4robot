# 算法处理生命周期与数据模型设计

## 一、核心设计决策

### 1.1 算法生命周期嵌入 cf_algo

算法处理的生命周期信息直接存储在 `assets.cf_algo` JSONB 字段中，不单独建步骤表。

**键名规范**：`<algo>@<version>:<field>`

```json
{
  "hand_tracking@1.2.0:status": "ok",
  "hand_tracking@1.2.0:gcs_uri": "gs://bucket/results/xxx.npz",
  "hand_tracking@1.2.0:started_at": "2026-04-24T10:00:00Z",
  "hand_tracking@1.2.0:finished_at": "2026-04-24T10:05:00Z",
  "hand_tracking@1.2.0:method": "ray_batch",
  "hand_tracking@1.2.0:dagster_run_id": "abc-123-def",
  "hand_tracking@1.2.0:error_message": "",

  "deface@2.0.0:status": "running",
  "deface@2.0.0:started_at": "2026-04-24T10:10:00Z"
}
```

**状态机**：

```
blocked → pending → running → ok
                  → failed → pending (重试)
```

**每个算法实例的标准字段**：

| 字段后缀 | 类型 | 必填 | 说明 |
|----------|------|------|------|
| `:status` | string | 是 | blocked / pending / running / ok / failed |
| `:started_at` | RFC3339 | running 时写入 | 开始处理时间 |
| `:finished_at` | RFC3339 | ok/failed 时写入 | 完成时间 |
| `:method` | string | 是 | 处理方法标识（ray_batch / local 等） |
| `:dagster_run_id` | string | 否 | Dagster run ID，用于 lineage 追溯 |
| `:gcs_uri` | string | ok 时必填 | 结果文件的 GCS 路径 |
| `:error_message` | string | failed 时填写 | 错误信息 |
| `:result_meta` | JSON string | 否 | 算法特定的结果元数据（帧数、置信度等） |

### 1.2 为什么不单独建步骤表

cyber-grace 用独立的 `video_steps` 表管理处理步骤，适合 Video 级（万级）的场景。
我们的场景是 Segment 级（百万到千万级），每个 Segment 跑 5-10 个算法：

- 独立步骤表：1000 万 Segment × 10 算法 = 1 亿行，查询和维护成本高
- cf_algo 方案：算法信息和资产在同一行，一次读取拿到全部信息，零 JOIN

Phase 1 切 Bigtable 时，cf_algo 直接映射到 `cf:algo` 列族的独立列，天然支持宽表，无需额外迁移。

### 1.3 轻量事件日志

单独建一张 `asset_algo_events` 表记录状态变更，用于审计和问题排查。
只记录关键信息，不记录完整快照。

```sql
CREATE TABLE IF NOT EXISTS asset_algo_events (
    event_id       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id       UUID         NOT NULL REFERENCES assets(asset_id),
    algo_key       TEXT         NOT NULL,   -- "hand_tracking@1.2.0"
    prev_status    TEXT,                    -- 变更前状态（首次为 NULL）
    new_status     TEXT         NOT NULL,   -- 变更后状态
    dagster_run_id TEXT,
    error_message  TEXT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_algo_events_asset
  ON asset_algo_events (asset_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_algo_events_algo_status
  ON asset_algo_events (algo_key, new_status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_algo_events_dagster
  ON asset_algo_events (dagster_run_id) WHERE dagster_run_id IS NOT NULL;
```

Phase 1 切 Bigtable 时，这张表可以迁移到独立的 Bigtable 表（rowkey: `asset_id#timestamp`），
也可以保留在 PG 中作为分析库（事件日志的查询模式偏分析，PG 更合适）。

---

## 二、算法注册表

用 YAML 配置文件定义合法的算法和版本，backend 启动时加载，usecase 层做验证。

```yaml
# config/algo_registry.yaml

algorithms:
  hand_tracking:
    description: "手部追踪"
    versions: ["1.0.0", "1.2.0"]
    output:
      required_fields: ["gcs_uri", "type"]
      uri_required: true

  head_tracking:
    description: "头部追踪"
    versions: ["1.0.0"]
    output:
      required_fields: ["gcs_uri", "type"]
      uri_required: true

  body_tracking:
    description: "身体追踪"
    versions: ["1.0.0"]
    output:
      required_fields: ["gcs_uri", "type"]
      uri_required: true

  deface:
    description: "去人脸"
    versions: ["2.0.0"]
    output:
      required_fields: ["gcs_uri", "width", "height", "fps", "source_stream", "eye"]
      uri_required: true

  action_annotation:
    description: "自动动作标注"
    versions: ["1.0.0"]
    output:
      required_fields: ["gcs_uri"]
      uri_required: true

  env_analysis:
    description: "AI 场景分析"
    versions: ["1.0.0"]
    output:
      required_fields: []
      uri_required: false

  content_normalization:
    description: "内容归一化"
    versions: ["1.0.0"]
    output:
      required_fields: []
      uri_required: false
```

**加新算法**：编辑 YAML → 重启服务（或热加载），不需要改代码、重新编译。

**验证逻辑**（usecase 层）：
- start_algo：检查 algo_key 是否在注册表中，版本是否合法
- finish_algo（ok）：检查 result_ref 是否包含 required_fields
- finish_algo（failed）：检查 error_message 非空

---

## 三、API 设计

### 3.1 开始处理

```
POST /api/v1/assets/:id/algo/:algo_key/start
```

```json
{
  "method": "ray_batch",
  "dagster_run_id": "abc-123-def"
}
```

行为：
1. 验证 algo_key 格式（`<name>@<version>`）和注册表
2. 更新 `cf_algo` 中对应键的 status → running，写入 started_at
3. 插入 `asset_algo_events` 记录
4. 返回 200

### 3.2 完成处理（成功）

```
POST /api/v1/assets/:id/algo/:algo_key/finish
```

```json
{
  "status": "ok",
  "gcs_uri": "gs://bucket/results/xxx.npz",
  "dagster_run_id": "abc-123-def",
  "result_meta": {
    "frame_count": 3600,
    "confidence": 0.95
  }
}
```

行为：
1. 验证 required_fields
2. 更新 `cf_algo` 中对应键的 status → ok，写入 finished_at、gcs_uri 等
3. 插入 `asset_algo_events` 记录
4. 返回 200

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

### 3.4 重置为 pending

```
POST /api/v1/assets/:id/algo/:algo_key/reset
```

用于重试失败的算法。清除 error_message 和 result 字段，status → pending。

### 3.5 查询算法事件历史

```
GET /api/v1/assets/:id/algo-events?algo_key=hand_tracking@1.2.0
```

返回该 asset 该算法的所有状态变更记录，按时间倒序。

### 3.6 批量查询待处理资产（给 Dagster sensor 用）

```
GET /api/v1/assets?filter=cf_algo.hand_tracking@1.2.0:status:eq:pending&page=1&page_size=100
```

利用通用 filter 语法 + GIN 索引，查询所有 hand_tracking 待处理的 Segment。

---

## 四、Dagster 集成

Dagster 的 sensor 和 asset 通过 SDK 调用上述 API：

```python
# sensor: 发现待处理的 Segment
pending_assets = client.assets.list(
    filter=["cf_algo.hand_tracking@1.2.0:status:eq:pending"],
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
    gcs_uri="gs://bucket/results/xxx.npz",
    dagster_run_id=context.run_id,
    result_meta={"frame_count": 3600},
)
```

**Lineage 追溯**：通过 `dagster_run_id` 可以从 asset 的 cf_algo 反查 Dagster 的完整运行记录，
也可以从 Dagster 的 asset materialization metadata 正查到 Grace 的 asset。

---

## 五、是否需要元数据湖？

**当前阶段：不需要。**

元数据湖（如架构文档中的 HBase/Lindorm 方案）解决的是百亿级元数据的存储和检索问题。
我们的规模路径：

| 阶段 | Segment 规模 | 元数据存储 | 够用吗 |
|------|-------------|-----------|--------|
| Phase 0 | < 100 万 | PostgreSQL + JSONB GIN | ✅ |
| Phase 0.5 | 100 万 - 1000 万 | PG + 热字段提升为真实列 | ✅ |
| Phase 1 | 1000 万 - 1 亿 | Bigtable | ✅ |
| Phase 2 | > 1 亿 | Bigtable + Elasticsearch CDC | ✅ |

元数据湖的核心能力（宽表、高并发点查、列族隔离）在 Phase 1 由 Bigtable 提供。
元数据湖的高级能力（联邦检索、血缘追踪、生命周期管理）可以在 Phase 2 按需引入。

**什么时候需要元数据湖**：
- 当你需要跨多个数据源做联邦检索（PG + Elasticsearch + 向量库）
- 当你需要百亿级的元数据存储（远超 Bigtable 单表的合理规模）
- 当你需要复杂的血缘追踪（A 资产的算法结果被 B 资产引用）

这些都是 Phase 2+ 的需求。

---

## 六、资产管理的完整模型

当前的资产管理已经覆盖了核心需求：

```
┌─────────────────────────────────────────────────────────────┐
│                    资产管理模型                               │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  mcap_files (原始文件)                                       │
│  ├─ cf_meta: 文件级元数据（GCS 路径、hash、大小、设备信息）    │
│  └─ cf_process: 文件级处理状态（ingest_state）               │
│       │                                                     │
│       │ 1:N                                                 │
│       ▼                                                     │
│  assets (Segment 级资产)                                     │
│  ├─ cf_meta: 资产元数据（时间范围、时长、环境、任务、QA 状态） │
│  ├─ cf_algo: 算法结果 + 生命周期（本文档的核心）             │
│  └─ cf_tag: 自由标签（priority、quality 等）                 │
│       │                                                     │
│       │ M:N                                                 │
│       ▼                                                     │
│  datasets (数据集，交付的上游)          ← 待建设              │
│  ├─ cf_meta: 数据集元数据（名称、描述、查询快照、统计）       │
│  └─ 版本链: parent_id → 上一版本                             │
│       │                                                     │
│       │ 1:1                                                 │
│       ▼                                                     │
│  deliveries (交付批次)                                       │
│  ├─ cf_meta: 交付元数据（合同、清单、客户）                   │
│  └─ delivery_items: asset ↔ delivery M:N 关联               │
│                                                             │
│  asset_algo_events (算法事件日志)        ← 本文档新增         │
│  └─ 状态变更审计记录                                         │
│                                                             │
│  idempotency_keys (幂等性)                                   │
│  └─ API 级幂等保障                                           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 待建设的部分

1. **Dataset 实体**：筛选 Segment → 创建 Dataset → review → seal → 交付
2. **通用 filter 查询**：`GET /api/v1/assets?filter=cf_algo.xxx:eq:yyy`
3. **算法生命周期 API**：本文档描述的 start/finish/reset
4. **MCAP 解析**：用 foxglove/mcap Go 库提取文件摘要

### 不需要建设的部分（当前阶段）

- 元数据湖（HBase/Lindorm）— Phase 1 用 Bigtable 替代
- 统一 Catalog（Gravitino）— 只有一个数据源
- 联邦检索（Trino）— PG GIN 索引够用
- Lance 湖仓 — 不做训练，不需要
- 向量检索（Milvus）— 不做语义搜索

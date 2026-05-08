# API 使用指南

本文档提供 data-platform 后端 API 的完整使用说明，包含 curl 示例。

> OpenAPI 规范: [`api/openapi.yaml`](../../api/openapi.yaml)

## ID 格式约定（示例统一）

- `asset_id`：固定 8 位字母数字（`[0-9A-Za-z]{8}`），示例统一用 `aset0001`
- `mcap_file_id`：固定 8 位字母数字（`[0-9A-Za-z]{8}`），示例统一用 `mcap0001`
- `delivery_id` / `event_id` / `eval_result_id`：UUID
- `action_id`：固定 8 位字母数字（`[0-9A-Za-z]{8}`）

> 下文所有请求/响应示例默认遵循上述格式，避免将非法 ID 复制到真实请求中触发 `400 INVALID_ARGUMENT`。

## 资产字段（现行可交付）

对接时以 **`api/openapi.yaml`** 中 Asset 及相关 schema 为唯一契约来源。常用语义：

- **`lifecycle_state`**：资产生命周期（状态机见《数据平台方案设计》§5.2.4）
- **`asset_type`**：资产类型（如 `segment`）
- **`duration_ms`**：时长（毫秒），与 `start_timestamp_ns` / `end_timestamp_ns` 一致
- **算法投影**：`asset_algo_latest`（响应中多为 `algo_results` 等形态，见 OpenAPI）
- **Tag 投影**：`asset_tags`（响应字段 `tags`）
- **事件**：`asset_events`（通过 `GET /api/v1/assets/{id}/events` 查询）

Query API 的筛选 / 排序字段名与注册中心一致，见 §1.3。

## 基础信息

- 基础 URL: `http://localhost:8080`（本地开发）
- 认证: `/api/v1/*` 默认需要 `X-Grace-Token`；也支持先走 `POST /api/v1/auth/login` 写入 `grace_session` cookie，再访问受保护接口
- 响应格式: JSON
- 请求 ID: 每个响应包含 `X-Request-ID` header

```bash
# 设置环境变量方便后续使用
export BASE=http://localhost:8080
export TOKEN=dev-token
```

会话登录（可选）：

```bash
curl -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"token":"'"$TOKEN"'"}' \
  -c /tmp/grace.cookie

curl "$BASE/api/v1/auth/me" -b /tmp/grace.cookie
curl -X POST "$BASE/api/v1/auth/logout" -b /tmp/grace.cookie
```

一键按本文示例跑 **最小冒烟**（Lakehouse、注册表、列表、可选写入链路 `POST mcap-files` → `POST assets`）：仓库根目录脚本 [`scripts/api-guide-smoke.sh`](../../scripts/api-guide-smoke.sh)。

```bash
BASE=http://localhost:8080 TOKEN=dev-token bash scripts/api-guide-smoke.sh
# 含创建 MCAP + 资产（需唯一 raw_hash_md5，脚本自动随机）：
RUN_WRITES=1 BASE=http://localhost:8080 TOKEN=dev-token bash scripts/api-guide-smoke.sh
```

## Lakehouse / Trino 验证

> Lakehouse 链路在 1.0 不可用；2.0 起由 PyIceberg CronJob 通过 outbox 增量入湖（详见《数据平台方案设计》§5.6.2 / §5.11）。本节命令演示的是本地脚手架，仅用于 demo / 验证。

启动 Iceberg + Trino + Catalog 服务：

```bash
make iceberg-up

# 触发一次 PyIceberg 演示 MERGE（本地脚本，将 PG 当前快照灌入 Bronze）
make iceberg-mvp
```

检查 Trino 查询层状态：

```bash
curl "$BASE/api/v1/lakehouse/status" \
  -H "X-Grace-Token: $TOKEN"
```

查看 Iceberg MVP 表行数：

```bash
curl "$BASE/api/v1/lakehouse/tables" \
  -H "X-Grace-Token: $TOKEN"
```

五类湖仓查询：

```bash
# 某次训练当时用了哪些 asset？
curl "$BASE/api/v1/lakehouse/training-assets?snapshot_id=mvp_hand_tracking_quality_v1" \
  -H "X-Grace-Token: $TOKEN"

# 某个算法版本变更后，哪些历史 asset 要重算？
curl "$BASE/api/v1/lakehouse/recompute-candidates?algo_key=hand_tracking@1.2.0&target_version=1.3.0" \
  -H "X-Grace-Token: $TOKEN"

# 某个 tag 是什么时候被算法追加的？
curl "$BASE/api/v1/lakehouse/tag-timeline?tag_key=quality" \
  -H "X-Grace-Token: $TOKEN"

# 上个月/近 30 天所有 MCAP segment 的质量分布是什么？
curl "$BASE/api/v1/lakehouse/quality-distribution?window=30d" \
  -H "X-Grace-Token: $TOKEN"

# 某个客户交付过的数据是否能完整回放？
curl "$BASE/api/v1/lakehouse/customer-replay?customer_id=urn:grace:customer:A" \
  -H "X-Grace-Token: $TOKEN"
```

兼容的静态报告接口仍然保留：

```bash
curl "$BASE/api/v1/lakehouse/report" \
  -H "X-Grace-Token: $TOKEN"
```

注意：上述 Lakehouse 接口在 1.0 阶段未上线；2.0 起 Trino 负责查询 Iceberg，入湖由 outbox + PyIceberg CronJob 完成（不引入 Spark / Dagster / Kafka / Debezium）。

## 1. 资产查询工作台 / 资产管理 (Assets)

> **`asset_id`（资产主键）**：固定 **8 位** ASCII **字母与数字**（`[0-9A-Za-z]{8}`），与 `mcap_file_id`（同为 8 位字母数字 ID）无关。创建资产时通常由服务端随机分配；也可在请求体中传入自定义 `asset_id`（须符合格式且未被占用，冲突返回 `409`、`DUPLICATE_ASSET_ID`）。凡路径中的 `{asset_id}` 均指该字段。

> 前置条件：`POST /api/v1/assets` 里的 `mcap_file_id` 必须指向一个**已存在**的 MCAP 文件记录。当前后端会在写 `asset_events` 时校验 `mcap_file_id` 外键，因此不能再像早期文档那样随便传不存在的占位字符串。

最简单的真实链路是：

1. 先调用 `POST /api/v1/mcap-files` 创建一条 MCAP 记录
2. 再把返回/自带的 `mcap_file_id` 用在 `POST /api/v1/assets`

### 1.1 创建资产

```bash
MCAP_ID="mcap0001"

curl -X POST "$BASE/api/v1/mcap-files" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"mcap_file_id\": \"$MCAP_ID\",
    \"gcs_path\": \"gs://bucket/path/file.mcap\",
    \"raw_hash_md5\": \"d41d8cd98f00b204e9800998ecf8427e\",
    \"ingest_state\": \"summarized\",
    \"size_bytes\": 1048576,
    \"file_duration_ms\": 60000,
    \"start_timestamp_ns\": 1700000000000000000,
    \"end_timestamp_ns\": 1700000060000000000,
    \"channel_count\": 12,
    \"chunk_count\": 5,
    \"vendor_id\": \"vendor-a\",
    \"device_id\": \"device-001\",
    \"scene_id\": \"indoor\",
    \"owner\": \"team-a\"
  }"
```

> ⚠️ `raw_hash_md5` 有唯一约束（`uq_mcap_files_hash_md5`）。同一环境重复运行时需使用不同的值，或省略该字段（后端允许 NULL）。

再创建资产：

```bash
curl -X POST "$BASE/api/v1/assets" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"mcap_file_id\": \"$MCAP_ID\",
    \"start_timestamp_ns\": 1700000000000000000,
    \"end_timestamp_ns\":   1700000060000000000,
    \"reviewer\": \"alice\",
    \"owner\": \"team-a\",
    \"asset_type\": \"segment\",
    \"tags\": {
      \"priority\": \"high\",
      \"quality\": \"good\",
      \"scene\": \"indoor\"
    }
  }"
```

响应 `201`:
```jsonc
{
  "asset_id": "b9a5a281",
  "mcap_file_id": "mcap0001",
  "lifecycle_state": "ready",
  "asset_type": "segment",
  "duration_ms": 60000,
  "reviewer": "alice",
  "owner": "team-a",
  "version": 1,
  "storage_uri": "",
  "thumb_uri": "",
  "retention_tier": "standard",
  "expire_at": null,
  "tenant_id": null,
  "project_id": null,
  "asset_level": 0,
  "parent_asset_id": null,
  "root_asset_id": null,
  "delivery_count": 0,
  "last_delivered_at": null,
  "last_delivered_to": null,
  "algo_results": {
    "env_analysis@1.0.0:status": "pending",
    "hand_tracking@1.0.0:status": "pending",
    "hand_tracking@1.2.0:status": "pending",
    "action_annotation@1.0.0:status": "blocked"
  },
  "tags": {"priority": "high", "quality": "good", "scene": "indoor"},
  "files": {"raw_mcap": "mcap0001"},
  "metadata": {},
  "lifecycle_meta": {
    "retention_tier": "standard",
    "archive_after_days": 90,
    "delete_after_days": 365
  },
  "created_at": "2026-04-25T10:00:00Z",
  "updated_at": "2026-04-25T10:00:00Z"
}
```

必填字段:
- `mcap_file_id` — 关联的 MCAP 文件 ID
- `start_timestamp_ns` — 起始时间戳 (纳秒, 不能为 0)
- `end_timestamp_ns` — 结束时间戳 (必须 > start)
- `reviewer` — 审核人

可选字段: `owner`, `asset_type`, `env`, `task`, `tags`。可选 **`asset_id`**：若传入则须为 **8 位字母数字** 且全局唯一；不传则由服务端生成。

Tags 校验规则:
- `priority`: 枚举 `critical | high | medium | low`
- `quality`: 枚举 `excellent | good | acceptable | poor | unusable`
- `scene`: 枚举 `indoor | outdoor | warehouse | office | factory`
- `task`, `batch`: 自由字符串
- `notes`: 自由字符串, 最大 500 字节

### 1.2 获取资产

```bash
curl "$BASE/api/v1/assets/{asset_id}" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`: 完整 Asset JSON（与 OpenAPI Asset schema 一致；与创建响应同一形状）

响应 `404`:
```json
{"code": "ASSET_NOT_FOUND", "message": "asset not found", "request_id": "..."}
```

### 1.3 列表查询（已收口到 Query API）

`GET /api/v1/assets` 旧列表查询路径不再是正式查询入口。资产查询工作台与对外推荐用法统一走：

- `POST /api/v1/queries/validate`
- `POST /api/v1/queries/run`

历史上的 `filter=field:op:value`、`sort_by`、`page/page_size` 语义，如需继续使用，请在客户端封装为 Query IR，而不是继续直连旧列表接口。

### 1.3.1 Query IR（v1）

> 当前状态：**已实现并进入主路径（桥接收口中）**。  
> 现阶段 Query API 已承载工作台主查询路径：`validate -> plan -> ES recall(optional) -> PG refine`。内部仍存在部分桥接实现（例如 PG refine 继续复用旧 filter 编译），但对前端已是主接口。
>
> 目标形态（JSON Query IR、字段注册中心、planner、compiler/executor、多引擎路由）见：《查询平台设计（Query IR / Planner / Executors）》。

v1 当前范围固定为：`resource=assets`，执行路径以 `ES recall + PG refine` 为主；ES 不可用或编译不支持时回退纯 PG。

当前 Query API 要点：

- `where` 使用树形结构：`and / or / not / pred`
- `pred / and / or / not` 当前都可执行；每个节点必须四选一（不能在同一节点同时出现）
- `mode` 支持：`structured` / `keyword` / `semantic` / `similar`
- 分页仍使用 `page / page_size`；`offset/limit` 目前仅作为 bridge 语义，要求 `offset % limit == 0`

```bash
# 仅校验（不执行）
curl -sS -X POST "$BASE/api/v1/queries/validate" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "schema_version": "v1",
    "scope": {"resource": "assets"},
    "where": {
      "and": [
        { "pred": { "field": "owner", "op": "eq", "value": "alice" } },
        { "pred": { "field": "delivery_count", "op": "gte", "value": 1 } }
      ]
    },
    "sort": [{"field": "created_at", "direction": "desc"}],
    "page": {"page": 1, "page_size": 20}
  }'

# 校验 + 执行（structured）
curl -sS -X POST "$BASE/api/v1/queries/run" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "schema_version": "v1",
    "scope": {"resource": "assets"},
    "select": {"fields": ["asset_id", "owner", "created_at"]},
    "where": {
      "pred": { "field": "owner", "op": "eq", "value": "alice" }
    },
    "sort": [{"field": "created_at", "direction": "desc"}],
    "page": {"page": 1, "page_size": 20, "offset": 0, "limit": 20}
  }'

# keyword / semantic / similar 也统一从 Query API 进入
curl -sS -X POST "$BASE/api/v1/queries/run" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "schema_version": "v1",
    "mode": "keyword",
    "scope": {"resource": "assets"},
    "where": {
      "pred": { "field": "_fulltext", "op": "ilike", "value": "warehouse rain" }
    },
    "page": {"page": 1, "page_size": 20, "offset": 0, "limit": 20}
  }'
```

错误语义：
- `400 INVALID_ARGUMENT`：请求结构不合法（如 `scope.resource` 非法、排序方向非法）
- `422 UNSUPPORTED_FIELD`：字段不在当前能力范围
- `422 UNSUPPORTED_OPERATOR`：操作符不支持
- `422 UNPLANNABLE_QUERY`：当前 planner 无法生成安全执行计划（少数未覆盖场景）

备注（避免误解）：
- 当前执行链仍包含桥接实现（PG refine 复用旧 filter 编译），但工作台主路径已统一到 Query API。
- 当前 `queries/run` 已支持 `warnings / facets / debug_plan`；structured / keyword / semantic / similar 主查询统一从 Query API 进入。

### 1.4 更新资产

```bash
curl -X PATCH "$BASE/api/v1/assets/{asset_id}" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reviewer": "bob",
    "lifecycle_state": "rejected",
    "tags": {"quality": "poor"}
  }'
```

所有字段都是可选的，只更新传入的字段，不影响其他字段。

并发安全：服务端使用乐观锁（`assets.version` CAS）。如果在你 GET 之后有其他写入提交，PATCH 会返回 `409 CONCURRENT_CONFLICT`，请重新拉取最新资产后再重试。

### 1.5 软删除

```bash
curl -X DELETE "$BASE/api/v1/assets/{asset_id}" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{"deleted": true, "asset_id": "..."}
```

软删除将 `lifecycle_state` 设为 `archived`，资产仍可通过 GET 访问，但不会出现在列表查询中。

### 1.6 查询资产关联的交付

```bash
curl "$BASE/api/v1/assets/{asset_id}/deliveries?page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": ["delivery-id-1", "delivery-id-2"],
  "asset_id": "...",
  "page": 1,
  "page_size": 20,
  "next_token": ""
}
```

## 2. 算法生命周期 (Algo Lifecycle)

### 状态机

```
blocked ──→ pending ──→ running ──→ ok ──→ (reset) ──→ pending
                                    └──→ failed ──→ (reset) ──→ pending
```

- `blocked`: 依赖未满足（自动管理，不可手动操作）
- `pending`: 等待启动
- `running`: 正在处理
- `ok`: 处理成功
- `failed`: 处理失败

### 可用算法

| algo_key | 说明 | 依赖 | 完成时必填字段 |
|----------|------|------|----------------|
| `env_analysis@1.0.0` | AI 场景分析 | 无 | 无 |
| `hand_tracking@1.0.0` | 手部追踪 v1 | 无 | `output_uri`, `type`, `result_size_bytes` |
| `hand_tracking@1.2.0` | 手部追踪 v1.2 | 无 | `output_uri`, `type`, `result_size_bytes` |
| `head_tracking@1.0.0` | 头部追踪 | 无 | `output_uri`, `type`, `result_size_bytes` |
| `body_tracking@1.0.0` | 身体追踪 | 无 | `output_uri`, `type`, `result_size_bytes` |
| `deface@2.0.0` | 去人脸 | 无 | `output_uri`, `width`, `height`, `fps`, `source_stream`, `eye`, `result_size_bytes` |
| `action_annotation@1.0.0` | 动作标注 | hand_tracking@1.2.0 + head_tracking@1.0.0 + body_tracking@1.0.0 | `output_uri`, `result_size_bytes` |

### 2.1 启动算法

```bash
curl -X POST "$BASE/api/v1/assets/{asset_id}/algo/env_analysis@1.0.0/start" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "k8s_job",
    "run_id": "run-12345"
  }'
```

响应 `200`:
```json
{"asset_id": "...", "algo_key": "env_analysis@1.0.0", "status": "running"}
```

错误码:
- `400 INVALID_ALGO_KEY` — algo_key 格式错误或不在注册表中
- `404 ASSET_NOT_FOUND` — 资产不存在
- `409 ALGO_ALREADY_RUNNING` — 算法已在运行
- `409 INVALID_STATE_TRANSITION` — 当前状态不允许启动（如 blocked/ok/failed）

### 2.2 完成算法

成功完成（无额外字段要求的算法）:
```bash
curl -X POST "$BASE/api/v1/assets/{asset_id}/algo/env_analysis@1.0.0/finish" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "ok"}'
```

成功完成（需要 output 的算法）:
```bash
curl -X POST "$BASE/api/v1/assets/{asset_id}/algo/hand_tracking@1.2.0/finish" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "ok",
    "output_uri": "gs://bucket/hand_tracking/output.mcap",
    "run_id": "run-12345",
    "result_size_bytes": 12345678,
    "extra_fields": {
      "type": "hand_tracking_v1"
    }
  }'
```

失败完成:
```bash
curl -X POST "$BASE/api/v1/assets/{asset_id}/algo/env_analysis@1.0.0/finish" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "failed",
    "reason": "OOM: memory limit exceeded",
    "run_id": "run-12345"
  }'
```

错误码:
- `409 INVALID_STATE_TRANSITION` — 当前状态不是 running
- `422 MISSING_REQUIRED_FIELD` — 缺少算法要求的必填字段
- `422 MISSING_REASON` — status=failed 时缺少 reason

幂等性: 相同 `run_id` 重复 finish 会返回 `200`（幂等），不同 `run_id` 返回 `409`。

### 2.3 重置算法

```bash
curl -X POST "$BASE/api/v1/assets/{asset_id}/algo/env_analysis@1.0.0/reset" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{"asset_id": "...", "algo_key": "env_analysis@1.0.0", "status": "pending"}
```

只有 `ok` 或 `failed` 状态可以重置。

### 2.4 标签管理

```bash
# 新增 / 更新单个标签
curl -X POST "$BASE/api/v1/assets/{asset_id}/tags" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"key":"quality","value":"good"}'

# 删除单个标签
curl -X DELETE "$BASE/api/v1/assets/{asset_id}/tags/quality" \
  -H "X-Grace-Token: $TOKEN"

# 查询标签变更历史
curl "$BASE/api/v1/assets/{asset_id}/tags/history?limit=20" \
  -H "X-Grace-Token: $TOKEN"
```

说明：
- `POST /tags` 走 `tag_registry` 校验，未注册 key 或非法 enum value 返回 `422 INVALID_TAG`。
- `DELETE /tags/{key}` 对不存在 key 按幂等成功处理，但不会伪造 `tag_deleted` 事件。
- `GET /tags/history` 返回的仍是统一 `asset_events` 形状，只是固定过滤 `tag_upserted / tag_deleted`。

### 2.5 查询资产事件 / 算法事件子集

```bash
# 查询资产完整事件时间线（最新在前）
curl "$BASE/api/v1/assets/{asset_id}/events?limit=50" \
  -H "X-Grace-Token: $TOKEN"

# 只看算法事件
curl "$BASE/api/v1/assets/{asset_id}/events?event_type=algo_*&limit=50" \
  -H "X-Grace-Token: $TOKEN"

# 按算法过滤，并用 cursor 继续翻下一页
curl "$BASE/api/v1/assets/{asset_id}/events?event_type=algo_*&algo_key=env_analysis@1.0.0&cursor=12345&limit=20" \
  -H "X-Grace-Token: $TOKEN"
```

说明：`{asset_id}` 须为 **8 位字母数字**（与创建响应中的 `asset_id` 同格式）；非法格式返回 `400`（`INVALID_ARGUMENT`），资产不存在返回 `404`（`ASSET_NOT_FOUND`）。`GET /tags/history` 同理。

响应 `200`（统一 `asset_events` 格式）:
```json
{
  "items": [
    {
      "event_id": "550e8400-e29b-41d4-a716-446655440000",
      "event_seq": 12345,
      "asset_id": "b9a5a281",
      "event_type": "algo_started",
      "event_payload": {
        "algo_key": "env_analysis@1.0.0",
        "algo_name": "env_analysis",
        "algo_version": "1.0.0",
        "prev_status": "pending",
        "new_status": "running",
        "run_id": "run-12345"
      },
      "payload_schema_version": "v1",
      "event_source": "backend",
      "request_id": "req-xxx",
      "created_at": "2026-04-25T10:00:00Z"
    }
  ],
  "limit": 20,
  "next_cursor": 12345
}
```

说明：
- 事件按 `event_seq` **降序**排列；`cursor` 语义为“继续取 `event_seq < cursor` 的更老事件”。
- `event_type` 支持精确值，也支持前缀通配，如 `algo_*`、`tag_*`。
- `algo_key` 过滤的是 `event_payload.algo_key`，通常与 `event_type=algo_*` 搭配使用。
- 响应字段与 `asset_events` 表一一对应；算法特有字段如 `prev_status / new_status / reason` 均保留在 `event_payload` 内。
- 已软删除的资产返回 `404 ASSET_NOT_FOUND`。

### 2.6 依赖链自动 Unblock

`action_annotation@1.0.0` 依赖三个算法。当所有依赖都完成（status=ok）后，系统自动将 `action_annotation` 从 `blocked` 变为 `pending`。

```bash
# 1. 完成三个依赖
for algo in hand_tracking@1.2.0 head_tracking@1.0.0 body_tracking@1.0.0; do
  curl -X POST "$BASE/api/v1/assets/{id}/algo/$algo/start" \
    -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
    -d '{"method":"k8s_job"}'
  
  curl -X POST "$BASE/api/v1/assets/{id}/algo/$algo/finish" \
    -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
    -d "{\"status\":\"ok\",\"output_uri\":\"gs://b/$algo.mcap\",\"result_size_bytes\":100,\"extra_fields\":{\"type\":\"v1\"}}"
done

# 2. action_annotation 自动变为 pending，现在可以启动了
curl -X POST "$BASE/api/v1/assets/{id}/algo/action_annotation@1.0.0/start" \
  -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"method":"k8s_job"}'
```

## 2.7 Action 段（seg 内时间分段标注）

> 状态：**Phase 1 已上线**：`POST /assets/:id/actions`、`GET /assets/:id/actions`（含 `at` / `from` / `to` / `label` 过滤）。`PATCH` / `DELETE` / `GET /actions`（平台反查）/ `GET /lookup` 仍在 §2.7.2 / §2.7.4 / §2.7.5 标记为「待上线」，按现有 OpenAPI 描述实现，请勿提前调用。

业务模型：`mcap → seg → action`。一条 action 是 seg 内某段时间窗上的一组标注（label + 描述 + 多源溯源）。

**关键约定**
- action 永远挂在 `asset_type='segment'` 的 seg 上；
- action **不参与** `lifecycle_state`，**不进** `deliveries`；
- 写入同事务追加 `asset_events`（`action_upserted` / `action_deleted`，schema 见 `backend/schemas/events/`）；
- `primary_label` 与 `labels[]` 必须来自 `backend/config/action_label_registry.yaml` 预定义集合（不在注册表内返回 `422 INVALID_ACTION`）；
- 时间戳与 `assets.start_timestamp_ns / end_timestamp_ns` 同口径；
- CDC 反向投影到 ES seg 文档的 `actions[]` nested 字段。
- 创建时 `start_ns` / `end_ns` 必须落在父 seg 的 `start_timestamp_ns`…`end_timestamp_ns` 之内（否则 `422 INVALID_ACTION`）。请求体里 **`start_ns` 不要写 `0`**：当前 handler 使用 `binding:"required"` 绑定 `int64`，Gin 会把 `0` 当成未提供而返回 `400 INVALID_ARGUMENT`。

### 2.7.1 创建 action

将下面示例中的时间戳换成目标 seg 的 `GET /api/v1/assets/{asset_id}` 响应里真实区间内的值（可与父区间同量级，例如纳秒级绝对时间）。

```bash
curl -X POST "$BASE/api/v1/assets/{asset_id}/actions" \
  -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{
    "start_ns": 1640056114776298435,
    "end_ns":   1640056115776298435,
    "primary_label": "pickup",
    "labels": ["pickup", "left_hand"],
    "description": "操作员从料盒中取出零件",
    "source_type": "human",
    "source_name": "annotator-001"
  }'
```

幂等：当前实现支持 `external_id`——同 `(asset_id, source_name, external_id)` 已存在时返回 `409 CONCURRENT_CONFLICT`（不自动覆盖；后续切片再启用 PATCH 路径）。`Idempotency-Key` header **未**在 actions 端点强制（与 `POST /deliveries` 不同）。

### 2.7.2 修改 / 删除（待上线）

```bash
# 改 label / labels / description / 时间窗；If-Match 走乐观锁
curl -X PATCH "$BASE/api/v1/assets/{asset_id}/actions/{action_id}" \
  -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
  -H "If-Match: 1" \
  -d '{"primary_label":"pickup_left","end_ns":2100000000}'

# 软删（CDC 投递 action_deleted）
curl -X DELETE "$BASE/api/v1/assets/{asset_id}/actions/{action_id}" \
  -H "X-Grace-Token: $TOKEN" -H "If-Match: 2"
```

### 2.7.3 查询：seg 内时间轴 / 时间戳点查 / 区间查

```bash
# 列出 seg 的所有 action（默认按 start_ns 排序）
curl "$BASE/api/v1/assets/{asset_id}/actions" -H "X-Grace-Token: $TOKEN"

# 时间戳点查：哪些 action 覆盖时间点 t
curl "$BASE/api/v1/assets/{asset_id}/actions?at=1500000000" -H "X-Grace-Token: $TOKEN"

# 区间查：与 [from,to] 重叠的 action
curl "$BASE/api/v1/assets/{asset_id}/actions?from=0&to=5000000000&label=pickup" \
  -H "X-Grace-Token: $TOKEN"
```

### 2.7.4 平台级反查："含某 action 的 seg"（待上线）

```bash
curl "$BASE/api/v1/actions?label=overtake&from=&to=&page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"
```

主路径走 ES（seg 文档 `actions[]` nested 字段，CDC 反向投影）；PG 兜底走 `(primary_label)` 索引。

### 2.7.5 时间戳一站式 lookup（前端时间轴用，待上线）

```bash
# 返回该绝对时间戳落在哪条 seg + 该时间点上的所有 action
curl "$BASE/api/v1/lookup?at=1700000000000000000" -H "X-Grace-Token: $TOKEN"
```

### 2.7.6 错误码

| 状态 | code | 触发 |
|------|------|------|
| 400 | `INVALID_ARGUMENT` | 路径中的 seg `asset_id` 非 8 位字母数字 / `end_ns < start_ns` / `start_ns` 为 `0` 导致绑定失败 |
| 404 | `ASSET_NOT_FOUND` | 父 seg 不存在或已软删 |
| 404 | `ACTION_NOT_FOUND` | action 不存在或已软删 |
| 409 | `CONCURRENT_CONFLICT` | `If-Match: version` 不匹配 |
| 422 | `INVALID_ACTION` | seg 不是 `asset_type='segment'`，或时间窗超出 seg 范围 |
| 500 | `INTERNAL_ERROR` | 罕见；若 `message` 为 `actions schema mismatch: run migration 018_actions_id_to_short_id.sql`，表示数据库 `actions.action_id` 列仍未迁移到 8 位短 id 语义，请对齐 `schemas/pg-phase0.sql` 并执行 `backend/migrations/018_actions_id_to_short_id.sql` |

## 3. 交付管理 (Deliveries)

### 3.1 创建交付

```bash
curl -X POST "$BASE/api/v1/deliveries" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-12345" \
  -d '{
    "asset_ids": ["aset0001", "aset0002"],
    "customer_id": "customer-001",
    "contract_id": "contract-001",
    "note": "Q1 delivery batch",
    "owner": "team-a"
  }'
```

响应 `201`:
```json
{"delivery_id": "cccccccc-3333-4000-8000-000000000001", "status": "delivered", ...}
```

> 注：当前实现会在创建时同步执行交付逻辑，所以 `status` 直接为 `delivered`，不经过 `pending` 中间态。

`Idempotency-Key` header 是必须的:
- 相同 key + 相同 body → 返回之前的结果 (`201`)
- 相同 key + 不同 body → `409` 冲突
- 缺少 key → `400`

`asset_ids` 中的每一项须为合法 **资产 `asset_id`**（8 位字母数字）；非法格式 → `400`（避免写入 `delivery_items` 时数据库报错）。

### 3.2 获取交付详情

```bash
curl "$BASE/api/v1/deliveries/{delivery_id}" \
  -H "X-Grace-Token: $TOKEN"
```

### 3.3 按客户查询交付

```bash
curl "$BASE/api/v1/customers/{customer_id}/deliveries?page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`：

```json
{
  "delivery_ids": ["delivery-id-1", "delivery-id-2"],
  "total": 2,
  "page": 1,
  "page_size": 20,
  "next_token": ""
}
```

### 3.4 交付列表

```bash
# 基础分页
curl "$BASE/api/v1/deliveries?page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"

# 按状态过滤
curl "$BASE/api/v1/deliveries?page=1&page_size=20&status=delivered" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [
    {
      "delivery_id": "cccccccc-3333-4000-8000-000000000001",
      "customer_id": "customer-001",
      "contract_id": "contract-001",
      "status": "delivered",
      "owner": "team-a",
      "note": "Q1 delivery batch",
      "item_count": 5,
      "delivered_at": "2026-05-01T10:00:00Z",
      "created_at": "2026-04-25T10:00:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

## 4. 批量创建片段 (内部接口)

```bash
curl -X POST "$BASE/internal/commit-segments" \
  -H "Content-Type: application/json" \
  -d '{
    "mcap_file_id": "mcap0001",
    "ranges": [
      [1700000000000000000, 1700000010000000000],
      [1700000020000000000, 1700000030000000000]
    ],
    "reviewer": "alice",
    "owner": "team-a"
  }'
```

响应 `201`:
```json
{"created": ["aset0001", "aset0002"], "count": 2}
```

注意：`POST /internal/commit-segments` 在 Phase 0 **未**挂在 `/api/v1` 的 `X-Grace-Token` 中间件上（见 `backend/routes/routes.go` 中 Internal 路由注释）。文档里其余 **`/api/v1/*`** 示例仍需 `X-Grace-Token` 或有效 `grace_session` cookie。

## 5. MCAP 文件管理

### 5.1 创建 MCAP 文件

```bash
curl -X POST "$BASE/api/v1/mcap-files" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "mcap_file_id": "mcap0001",
    "gcs_path": "gs://bucket/path/file.mcap",
    "raw_hash_md5": "unique-hash-per-file",
    "ingest_state": "summarized",
    "size_bytes": 1048576,
    "file_duration_ms": 60000,
    "start_timestamp_ns": 1700000000000000000,
    "end_timestamp_ns": 1700000060000000000,
    "channel_count": 12,
    "chunk_count": 5,
    "vendor_id": "vendor-a",
    "device_id": "device-001",
    "scene_id": "indoor",
    "owner": "team-a",
    "metadata": {
      "source": "manual_import"
    },
    "process_state": {
      "hand_tracking": "completed"
    }
  }'
```

响应 `201`：返回完整 `McapFile` JSON。

### 5.2 列出 MCAP 文件

```bash
curl "$BASE/api/v1/mcap-files?page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [
    {
      "mcap_file_id": "mcap0001",
      "gcs_path": "gs://bucket/path/file.mcap",
      "size_bytes": 1048576,
      "ingest_state": "summarized",
      "channel_count": 12,
      "chunk_count": 5,
      "owner": "team-a",
      "created_at": "2025-01-15T10:00:00Z",
      "updated_at": "2025-01-15T10:05:00Z",
      "version": 2
    }
  ],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

### 5.3 获取单个 MCAP 文件

```bash
curl "$BASE/api/v1/mcap-files/{mcap_file_id}" \
  -H "X-Grace-Token: $TOKEN"
```

响应中时间字段说明：

- `start_timestamp_ns / end_timestamp_ns`：文件**物理**起止（含废段，由 ingest 时确定，不再变更）

### 5.4 按时间检索 MCAP（规划中，当前未开放查询参数）

`at` / `overlaps_from` / `overlaps_to` 仍处于规划阶段，当前 `GET /api/v1/mcap-files` 不支持这些参数。
如需按时间筛选，请先通过资产或算法侧条件过滤，再按 `mcap_file_id` 回查文件详情。

## 6. 注册表 (Registry)

### 6.1 算法注册表

```bash
curl "$BASE/api/v1/algo-registry" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [
    {
      "key": "hand_tracking@1.2.0",
      "name": "hand_tracking",
      "version": "1.2.0",
      "depends_on": []
    },
    {
      "key": "action_annotation@1.0.0",
      "name": "action_annotation",
      "version": "1.0.0",
      "depends_on": ["hand_tracking@1.2.0", "head_tracking@1.0.0", "body_tracking@1.0.0"]
    }
  ]
}
```

### 6.1.1 资产生命周期枚举（与 PG `chk_lifecycle_state` 对齐）

```bash
curl "$BASE/api/v1/lifecycle-states" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`：`items` 为写入 `assets.lifecycle_state` 时允许的取值（与 `migrations/009_lifecycle_state_check.sql` 一致）。

### 6.2 标签注册表

**`tag_registry.yaml` 是 tag 的「白名单字典」**：只有在这里声明过的 **key**（例如 `priority`、`scene`）才允许作为 `tag_key` 出现在受校验的写入路径里；**`enum` 类型还限制 value 必须在给出的列表中。** 源文件路径：`backend/config/tag_registry.yaml`；可在进程运行中热重载。

```bash
curl "$BASE/api/v1/tag-registry" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`（`items` 按 `key` 字母序；每项含 YAML 中的 `description`，便于前端字典页展示）:
```json
{
  "items": [
    {"key": "priority", "description": "处理优先级", "type": "enum", "values": ["critical", "high", "medium", "low"]},
    {"key": "quality", "description": "数据质量评级", "type": "enum", "values": ["excellent", "good", "acceptable", "poor", "unusable"]},
    {"key": "scene", "description": "采集场景", "type": "enum", "values": ["indoor", "outdoor", "warehouse", "office", "factory"]},
    {"key": "notes", "description": "自由备注", "type": "string", "max_length": 500},
    {"key": "batch", "description": "采集批次号", "type": "string"},
    {"key": "task", "description": "采集任务标识", "type": "string"}
  ]
}
```

## 7. Elasticsearch 检索 (Search)

### 7.1 全文检索资产
`GET /api/v1/search/assets` 已从正式查询主路径下线。  
全文、keyword、semantic、similar、facet 统一通过 Query API 进入：

- `POST /api/v1/queries/validate`
- `POST /api/v1/queries/run`

建议：

- keyword：`mode=keyword` + `_fulltext` predicate
- semantic：`mode=semantic`
- similar：`mode=similar`
- 结构化 tag / mcap / algo / action 条件：统一放进 Query IR 的 `where`

### 7.2 Admin：全量从 PG 重建 ES（不推进 outbox 游标）

当前先复用 **`X-Grace-Token`**；不再额外要求 `X-Admin-Token`。

```bash
curl -sS -X POST "$BASE/api/v1/admin/search/reindex" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"dry_run": true, "page_size": 200}' | jq .
```

`dry_run: true` 时只统计将要索引/删除的文档数，不写 ES。

响应 `200`:

```json
{
  "dry_run": true,
  "total_assets": 1200,
  "indexed": 1180,
  "deleted": 20,
  "failed": 0,
  "duration_ms": 842,
    "assets_scanned": 1200,
  "documents_indexed": 1180,
  "documents_deleted": 20
}
```

- `indexed` / `deleted` 在 `dry_run=true` 时表示“将会执行”的数量。
- `failed > 0` 时检查后端日志；ES `_bulk` 的局部失败不会被误报成全成功。

### 7.3 Prometheus 指标

同一进程暴露 `GET /metrics`（无认证；建议仅内网可达）。指标前缀包括 `gin_` (HTTP 请求时延/状态码)、`go_` (运行时) 等。

> ⚠️ `outbox_worker_*` 系列指标（`outbox_worker_pending_total` 等）已随 outbox polling worker 一同删除（CDC 架构取代了它）。如有监控依赖这些指标名，需更新告警规则。

`filter` 语法速查:

| op | 例子 | 含义 |
|----|------|------|
| `eq` / `ne` | `owner:eq:alice` | 等于 / 不等于 |
| `in` / `nin` | `tags_flat.scene:in:highway,urban` | 命中 / 不命中集合 |
| `gt` / `gte` / `lt` / `lte` | `duration_ms:gte:60000` | 数值 / 时间比较 |
| `between` | `duration_ms:between:30000,120000` | 闭区间 |
| `exists` | `tags_flat.weather:exists:true` | 字段存在 |
| `ilike` | `notes:ilike:%夜间%` | 大小写不敏感子串 |

字段前缀约定:

- `tags_flat.<key>` — 物化扁平字段，**首选**等值过滤路径（如 `tags_flat.scene`、`tags_flat.time_of_day`）。
- `mcap.<col>` — mcap 反范式属性（`vendor_id` / `device_id` / `scene_id` / `camera_model` 等）。
- `algos.<key>.status` / `algos.<key>.score` — 算法状态（nested，后端自动转 nested query）。
- 顶层字段：`lifecycle_state` / `asset_type` / `owner` / `duration_ms` / `created_at` / `updated_at`。

常见误区:

- ❌ 把 `scene=highway` 塞进全文搜索 → 应改成 Query IR 谓词 `tags_flat.scene = highway`。
- ❌ 想精确匹配 owner 却只做 `_fulltext` → 应改成 Query IR 谓词 `owner = alice`。
- ✅ 全文 + 多 tag 组合：`_fulltext` 谓词 + 多个结构化谓词一起放进 `where.and`。

注意: 需要 Elasticsearch 服务运行。当 Elasticsearch 不可用时返回 `503`。

> 当前 ES facet 聚合依赖 `*.keyword` 子字段。如果你本地索引是旧 mapping，`queries/run` 可能返回 warning 并回退纯 PG。先调用一次 `POST /api/v1/admin/search/reindex` 或重建本地 `assets` 索引。

### 7.1.1 按 tag 检索（三条路径）
推荐的正式写法：

- `tag.priority`
- `tag.quality`
- `tags_flat.<key>`（等值首选）
- `tags.<key>`（nested key/value）
- `tags.<inner>`（`source_type`、`confidence` 等元信息）

这些条件都通过 Query IR `where` 进入，由 planner 决定走 ES recall 还是 PG refine；不再单独依赖旧 `/search/assets` 参数协议。

## 8. 湖仓同步状态

### 8.1 查询同步状态

```bash
curl "$BASE/api/v1/lakehouse/sync-status" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "available": true,
  "source": "realtime",
  "data": {
    "dagster_run_id": "",
    "checked_at": "2026-05-15T08:30:00Z",
    "pg_total_count": 10000,
    "iceberg_total_count": 9998,
    "count_diff_pct": 0.0002,
    "pg_status_dist": {},
    "iceberg_status_dist": {},
    "status_diff": {},
    "is_alert": false,
    "iceberg_max_seq": 456789
  }
}
```

- `source = realtime` 表示直接比较 `asset_events.publish_state='published'` 与 Iceberg Bronze 的事件总数。
- `source = sync_reconciliation` 表示读取 Dagster 落表结果。
- `count_diff_pct` 是 **比值**，不是已经乘过 100 的百分数；例如 `0.001 = 0.1%`。
- 当 `available=false` 时，接口仍返回 `200`，并通过 `message` 说明是 Dagster 数据未生成还是 Bronze 尚未追平。

## 8.2 Eval / Metrics（Phase 1.5，已上线）

> 详细设计见《Eval Metrics 设计（Phase 1.5）》。以下端点**已实现并可用**。

端点清单：

- `POST /api/v1/assets/{asset_id}/eval-results`（K1）— 写入评估结果并展开指标
- `GET /api/v1/assets/{asset_id}/eval-results`（K2）— 查询评估历史
- `GET /api/v1/assets/{asset_id}/metrics`（K3）— 查询指标投影
- `GET /api/v1/metrics/registry`（K5）— 查询注册指标白名单
- `POST /api/v1/metrics:search`（K6）— 按指标过滤资产

写入示例（K1）：

```bash
ASSET_ID="b9a5a281"

curl -X POST "$BASE/api/v1/assets/$ASSET_ID/eval-results" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_type": "segment",
    "target_id": "",
    "eval_name": "qc_deliverable_frame_eval",
    "eval_version": "1.0.0",
    "parameter_version": "8",
    "run_id": "run-2026-05-05-001",
    "source_type": "algo",
    "source_name": "qc-worker",
    "status": "ok",
    "result_payload": {
      "good_frames_ratio": 0.0009407337723424271,
      "hand_good_frames": 244,
      "total_good_frames": 1,
      "total_video_frames": 1063,
      "weighted_avg_convex_hull_volume": 0
    }
  }'
```

按指标搜索示例（K6）：

```bash
curl -X POST "$BASE/api/v1/metrics:search" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "filters": {
      "lifecycle_state": "ready",
      "metrics": [
        { "metric_key": "good_frames_ratio", "op": ">=", "value": 0.8 }
      ]
    },
    "page": 1,
    "page_size": 20
  }'
```

语义约束：

- 服务端必须在单事务内完成：`asset_eval_results` 写入、`asset_metrics` 展开、`asset_events` 追加。
- `asset_metrics` 只接收注册表白名单 key；未注册 key 仅保留在 `result_payload`。
- 指标本身不直接修改 `lifecycle_state`，必须经过显式规则触发。

## 9. 错误码参考

| HTTP | Code | 说明 |
|------|------|------|
| 400 | `INVALID_ARGUMENT` | 请求格式错误 |
| 400 | `INVALID_FILTER` | 过滤条件无效 |
| 400 | `INVALID_ALGO_KEY` | 算法 key 无效 |
| 401 | `UNAUTHORIZED` | 认证失败 |
| 404 | `ASSET_NOT_FOUND` | 资产不存在 |
| 409 | `DUPLICATE_ASSET_ID` | `POST /assets` 指定了已存在的 `asset_id` |
| 409 | `ALGO_ALREADY_RUNNING` | 算法已在运行 |
| 409 | `CONCURRENT_CONFLICT` | 乐观锁冲突 — 算法路径重试 3 次后仍失败，或 PATCH /assets/:id 期间资产被并发修改 |
| 414 | `URI_TOO_LONG` | URL 超过 2048 字符 |
| 422 | `INVALID_STATE` | 业务状态错误 (如 start > end) |
| 422 | `INVALID_TAG` | Tag key 未注册或值不合法 |
| 422 | `MISSING_REQUIRED_FIELD` | 算法完成时缺少必填字段 |
| 422 | `MISSING_REASON` | failed 状态缺少 reason |
| 500 | `INTERNAL_ERROR` | 服务器内部错误 |

## 10. 典型工作流

### 资产入库 → 算法处理 → 交付

```bash
# 1. MCAP 文件上传后，创建资产
ASSET=$(curl -s -X POST "$BASE/api/v1/assets" \
  -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"mcap_file_id":"mcap0001","start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000060000000000,"reviewer":"alice"}')
ASSET_ID=$(echo $ASSET | python3 -c "import sys,json; print(json.load(sys.stdin)['asset_id'])")

# 2. 外部算法 worker 触发处理
curl -X POST "$BASE/api/v1/assets/$ASSET_ID/algo/env_analysis@1.0.0/start" \
  -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"method":"k8s_job","run_id":"run-001"}'

# 3. 算法完成回调
curl -X POST "$BASE/api/v1/assets/$ASSET_ID/algo/env_analysis@1.0.0/finish" \
  -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"status":"ok","run_id":"run-001"}'

# 4. 交付给客户
curl -X POST "$BASE/api/v1/deliveries" \
  -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
  -H "Idempotency-Key: delivery-$(date +%s)" \
  -d "{\"asset_ids\":[\"$ASSET_ID\"],\"customer_id\":\"cust-001\"}"
```

# API 使用指南

本文档提供 data-platform 后端 API 的完整使用说明，包含 curl 示例。

> OpenAPI 规范: [`api/openapi.yaml`](../../api/openapi.yaml)

## 字段命名口径（重要）

> **API 当前同时接受新旧字段名，但新代码 / SDK / 前端请优先使用"新字段"。** 旧字段在 1.0 → 2.0 迁移窗口内保留别名读写，2.0 之后会从响应中移除。

| 维度 | 旧字段（兼容保留） | 新字段（主线，优先用） | 说明 |
|------|-------------------|------------------------|------|
| 资产生命周期 | `status` | `lifecycle_state` | 旧 `status` 沿用 review 流程枚举（pending / approved / rejected …），新 `lifecycle_state` 是统一的资产生命周期状态机（见 `data-platform-design.md §5.2.4`） |
| 资产类型 | `type` | `asset_type` | 旧 `type` 名字过于宽泛，与 HTTP `Content-Type`、tag value type 容易混淆 |
| 时长 | `duration_sec` | `duration_ms` | 与 `start_timestamp_ns / end_timestamp_ns` 单位对齐到毫秒精度 |
| 算法结果存储 | `cf_algo.<algo>@<ver>:<field>` (JSONB key) | `asset_algo_latest` 投影表 + `asset_events` 事件表 | **已切换**——后端唯一写入路径 |
| Tag 存储 | `cf_tag` JSONB | `asset_tags` 投影表 | **已切换**——后端唯一写入路径 |
| 业务事件 | `asset_algo_events`（仅算法） | `asset_events`（统一事件 / outbox） | **已切换**——所有事件只入 `asset_events`；老表保留只读 |

筛选 / 排序字段同时接受新旧两种写法，详见 §1.3。



## 基础信息

- 基础 URL: `http://localhost:8080`（本地开发）
- 认证: 所有 `/api/v1/*` 端点需要 `X-Grace-Token` header
- 响应格式: JSON
- 请求 ID: 每个响应包含 `X-Request-ID` header

```bash
# 设置环境变量方便后续使用
export BASE=http://localhost:8080
export TOKEN=dev-token
```

## Lakehouse / Trino 验证

> Lakehouse 链路在 1.0 不可用；2.0 起由 PyIceberg CronJob 通过 outbox 增量入湖（详见 `data-platform-design.md §5.6.2 / §5.11`）。本节命令演示的是本地脚手架，仅用于 demo / 验证。

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

## 1. 资产管理 (Assets)

### 1.1 创建资产

```bash
curl -X POST "$BASE/api/v1/assets" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "mcap_file_id": "mcap-001",
    "start_timestamp_ns": 1700000000000000000,
    "end_timestamp_ns":   1700000060000000000,
    "reviewer": "alice",
    "owner": "team-a",
    "type": "task_demo",
    "env": "indoor",
    "task": "pick_and_place",
    "tags": {
      "priority": "high",
      "quality": "good",
      "scene": "indoor"
    }
  }'
```

响应 `201`:
```json
{
  "asset_id": "b9a5a281-9761-4389-a227-6a6d28dd0057",
  "mcap_file_id": "mcap-001",
  "status": "approved",
  "lifecycle_state": "ready",
  "type": "task_demo",
  "asset_type": "task_demo",
  "duration_sec": 60.0,
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
  "files": {"raw_mcap": "mcap-001"},
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

> **双写期间字段说明**: 响应同时包含旧字段（`status`, `duration_sec`, `type`）和新字段（`lifecycle_state`, `duration_ms`, `asset_type`）。新字段是权威来源，旧字段由后端自动映射生成。新代码请优先使用新字段。

必填字段:
- `mcap_file_id` — 关联的 MCAP 文件 ID
- `start_timestamp_ns` — 起始时间戳 (纳秒, 不能为 0)
- `end_timestamp_ns` — 结束时间戳 (必须 > start)
- `reviewer` — 审核人

可选字段: `owner`, `type`, `env`, `task`, `tags`

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

响应 `200`: 完整 Asset JSON（同创建响应，包含所有新旧字段）

响应 `404`:
```json
{"code": "ASSET_NOT_FOUND", "message": "asset not found", "request_id": "..."}
```

### 1.3 列表查询

```bash
# 基础分页
curl "$BASE/api/v1/assets?page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"

# 按字段过滤
curl "$BASE/api/v1/assets?filter=status:eq:approved&filter=owner:eq:alice" \
  -H "X-Grace-Token: $TOKEN"

# 按 tag 别名过滤
curl "$BASE/api/v1/assets?filter=tags.notes:ilike:night-run&sort_by=-tag.priority" \
  -H "X-Grace-Token: $TOKEN"

# 按算法状态过滤
curl "$BASE/api/v1/assets?filter=algo.hand_tracking@1.2.0:status:eq:failed" \
  -H "X-Grace-Token: $TOKEN"

# 排序 (- 前缀表示降序)
curl "$BASE/api/v1/assets?sort_by=-created_at&page=1&page_size=10" \
  -H "X-Grace-Token: $TOKEN"

# 按 MCAP 文件查询 (向后兼容)
curl "$BASE/api/v1/assets?mcap_file_id=mcap-001" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [...],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

过滤操作符:

| 操作符 | 含义 | 示例 |
|--------|------|------|
| `eq` | 等于 | `status:eq:approved` |
| `ne` | 不等于 | `status:ne:archived` |
| `lt` / `gt` | 小于/大于 | `version:gt:5` |
| `lte` / `gte` | 小于等于/大于等于 | `delivery_count:gte:1` |
| `like` | 模糊匹配 | `owner:like:%team%` |
| `ilike` | 不区分大小写模糊 | `reviewer:ilike:%alice%` |
| `in` | 包含在列表中 | `status:in:["approved","rejected"]` |
| `nin` | 不在列表中 | `status:nin:["archived"]` |

允许的过滤/排序字段（**优先使用新字段名**）:

- 标识：`asset_id`, `mcap_file_id`, `version`
- 生命周期：`lifecycle_state`（新）/ `status`（旧，兼容保留）, `qa_state`, `reviewer`, `owner`
- 资产属性：`asset_type`（新）/ `type`（旧）, `env`, `task`
- 时间：`created_at`, `updated_at`, `start_timestamp_ns`, `end_timestamp_ns`, `duration_ms`（新）/ `duration_sec`（旧）
- 交付汇总：`delivery_count`, `last_delivered_to`, `last_delivered_at`
- 留存策略：`retention_tier`, `archive_after_days`, `delete_after_days`, `total_size_bytes`, `last_accessed_at`
- Tag 前缀：`tag.<key>`（推荐）
- 算法结果前缀：`algo.<key>`（推荐，路由到 `asset_algo_latest`）
- 文件引用前缀：`files.<key>`

兼容别名（旧 → 新，仅过渡期支持）:

- `status` → `lifecycle_state`（语义不完全一致，详见上方"字段命名口径"表）
- `type` → `asset_type`
- `duration_sec` → `duration_ms`（单位变换由后端自动处理）
- `tags.<key>` → `tag.<key>`
- `cf_tag.<key>` → `tag.<key>`
- `algo_results.<key>` → `algo.<key>`
- `cf_algo.<key>` → `algo.<key>`
- `cf_files.<key>` → `files.<key>`

非法字段或排序字段会返回 `400 INVALID_FILTER`。

### 1.4 更新资产

```bash
curl -X PATCH "$BASE/api/v1/assets/{asset_id}" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reviewer": "bob",
    "status": "rejected",
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

软删除将 `status` 设为 `archived`，资产仍可通过 GET 访问，但不会出现在列表查询中。

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

响应 `200`（统一 `asset_events` 格式）:
```json
{
  "items": [
    {
      "event_id": "uuid",
      "event_seq": 12345,
      "asset_id": "uuid",
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

## 3. 交付管理 (Deliveries)

### 3.1 创建交付

```bash
curl -X POST "$BASE/api/v1/deliveries" \
  -H "X-Grace-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-12345" \
  -d '{
    "asset_ids": ["asset-id-1", "asset-id-2"],
    "customer_id": "customer-001",
    "contract_id": "contract-001",
    "note": "Q1 delivery batch",
    "owner": "team-a"
  }'
```

响应 `201`:
```json
{"delivery_id": "uuid", "status": "pending", ...}
```

`Idempotency-Key` header 是必须的:
- 相同 key + 相同 body → 返回之前的结果 (`201`)
- 相同 key + 不同 body → `409` 冲突
- 缺少 key → `400`

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

### 3.3 交付列表

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
      "delivery_id": "uuid",
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
    "mcap_file_id": "mcap-001",
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
{"created": ["asset-id-1", "asset-id-2"], "count": 2}
```

注意: 内部接口不需要 `X-Grace-Token`。

## 5. MCAP 文件管理

### 5.1 列出 MCAP 文件

```bash
curl "$BASE/api/v1/mcap-files?page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [
    {
      "mcap_file_id": "mcap-001",
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

### 5.2 获取单个 MCAP 文件

```bash
curl "$BASE/api/v1/mcap-files/{mcap_file_id}" \
  -H "X-Grace-Token: $TOKEN"
```

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

### 6.2 标签注册表

**`tag_registry.yaml` 是 tag 的「白名单字典」**：只有在这里声明过的 **key**（例如 `priority`、`scene`）才允许作为 `tag_key` 出现在受校验的写入路径里；**`enum` 类型还限制 value 必须在给出的列表中。** 源文件路径：`backend/config/tag_registry.yaml`；可在进程运行中热重载。

```bash
curl "$BASE/api/v1/tag-registry" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [
    {"key": "priority", "type": "enum", "values": ["critical", "high", "medium", "low"]},
    {"key": "quality", "type": "enum", "values": ["excellent", "good", "acceptable", "poor", "unusable"]},
    {"key": "scene", "type": "enum", "values": ["indoor", "outdoor", "warehouse", "office", "factory"]},
    {"key": "notes", "type": "string", "max_length": 500},
    {"key": "batch", "type": "string", "max_length": 255}
  ]
}
```

## 7. Elasticsearch 检索 (Search)

### 7.1 全文检索资产

```bash
# 1) 纯关键词（在 notes / owner / reviewer / asset_id 里搜）
curl "$BASE/api/v1/search/assets?q=corner+case&page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"

# 2) 关键词 + 结构化过滤（推荐：tag 走 filter，不要塞进 q）
curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "q=corner case" \
  --data-urlencode "filter=lifecycle_state:eq:ready" \
  --data-urlencode "filter=tags_flat.scene:eq:highway" \
  --data-urlencode "filter=duration_ms:between:30000,120000"

# 3) 纯结构化过滤（无关键词，等同于按 tag 资产发现）
curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags_flat.scene:in:highway,urban" \
  --data-urlencode "filter=mcap.vendor_id:eq:vendor-a" \
  --data-urlencode "filter=owner:ne:test-bot"
```

响应 `200`:
```json
{
  "items": [
    {
      "asset_id": "ast_01HX...",
      "lifecycle_state": "ready",
      "owner": "alice",
      "duration_ms": 87500,
      "tags_flat": {"scene": "highway", "time_of_day": "night"},
      "mcap": {"vendor_id": "vendor-a", "scene_id": "highway-01"},
      "_highlight": {"notes": ["典型的 <em>corner</em> <em>case</em> ..."]}
    }
  ],
  "total": 128,
  "page": 1,
  "page_size": 20,
  "facets": {
    "lifecycle_state_agg": {"buckets": [{"key": "ready", "doc_count": 120}]},
    "asset_type_agg":      {"buckets": [{"key": "mcap_segment", "doc_count": 110}]},
    "owner_agg":           {"buckets": [{"key": "alice", "doc_count": 32}]},
    "vendor_agg":          {"buckets": [{"key": "vendor-a", "doc_count": 58}]},
    "scene_agg":           {"buckets": [{"key": "highway", "doc_count": 58}]}
  }
}
```

查询参数:

- `q` — 全文搜索关键词，**仅匹配** `notes` / `owner.text` / `reviewer.text` / `asset_id` 四个字段（ES `multi_match`）。**Tag 值、vendor、device 等结构化字段不在这里**——走 `filter=`。
- `filter` — 结构化过滤，可重复多个，AND 关系。语法 `field:op:value`，与 `/api/v1/assets` 一致。
- `page` / `page_size` — 分页（`page_size ≤ 200`）。

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

- ❌ `q=highway` 想找 `scene=highway` 的资产 → 应改成 `filter=tags_flat.scene:eq:highway`。
- ❌ `q=alice` 想精确匹配 owner → 应改成 `filter=owner:eq:alice`（否则 `notes` 里出现 alice 的也会命中）。
- ✅ 关键词 + 多 tag 同时使用：`q=...` + 多个 `filter=tags_flat.<key>:eq:<value>`。

注意: 需要 Elasticsearch 服务运行。当 Elasticsearch 不可用时返回 `503`。

### 7.1.1 按 tag 检索（三条路径）

ES 提供 3 条互补的 tag 检索路径，前端按需挑一条即可。底层映射在 `internal/elasticsearch/client.go` 的 `nestedPath()` / `buildNestedClause()`。

| 路径 | API 写法 | ES 底层 | 适用场景 |
|------|---------|---------|---------|
| **A. `tags_flat.<key>`** | `filter=tags_flat.scene:eq:highway` | ES `term` on `flattened` 子键 | **90% 等值过滤**，最快，首选 |
| **B. `tags.<key>`** | `filter=tags.scene:eq:highway` | ES `nested` query（自动 pin `tags.key + tags.value`） | 复杂条件（带 source / confidence） |
| **C. `tags.<inner>`** | `filter=tags.source_type:eq:algo` | ES `nested` 内部条件 | 不挑 key，只问"有没有算法打的 tag" |

#### A. 等值（最常用）—— `tags_flat`

```bash
# scene=highway
curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags_flat.scene:eq:highway"

# 多 tag AND：高速 + 夜间 + ready
curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags_flat.scene:eq:highway" \
  --data-urlencode "filter=tags_flat.time_of_day:eq:night" \
  --data-urlencode "filter=lifecycle_state:eq:ready"

# OR 集合：高速 OR 城市
curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags_flat.scene:in:highway,urban"

# 排除 / 存在性
curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags_flat.scene:ne:highway"

curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags_flat.weather:exists:true"
```

#### B. 复杂条件 —— `tags.<key>` (nested)

```bash
# scene=highway 且必须是算法打的
curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags.scene:eq:highway" \
  --data-urlencode "filter=tags.source_type:eq:algo"
```

> ⚠️ 多个 `tags.*` filter **作用在不同 nested doc 上**——即"有一条 scene=highway"和"有一条 source_type=algo"，**不强制是同一条 tag**。如需"同一条 tag 内多条件捆绑"，目前不支持（候选 P2）。90% 场景下 A 路径足够。

#### C. 只问元信息 —— `tags.<inner>`

允许的 inner 字段：`source_type` / `source_name` / `confidence` / `value_num` / `value_bool` / `value` / `key`。

```bash
# 任何被算法打过 tag 的资产
curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags.source_type:eq:algo"

# 任何 confidence ≥ 0.9 的 tag（不限 key）
curl -G "$BASE/api/v1/search/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags.confidence:gte:0.9"
```

#### PG fallback

`/api/v1/assets`（或 `/search/assets` 在 ES 不可用时降级）也支持按 tag 过滤，走 `EXISTS` 子查询命中 `asset_tags(tag_key, tag_value, asset_id)` 索引：

```bash
curl -G "$BASE/api/v1/assets" \
  -H "X-Grace-Token: $TOKEN" \
  --data-urlencode "filter=tags.scene:eq:highway"
```

#### 能力边界

| 能力 | ES | PG fallback |
|------|----|-------------|
| 单 tag 等值 / IN / NOT / 存在性 | ✅ | ✅ |
| 多 tag AND | ✅ | ✅ |
| 数值 tag 范围（`value_num`、`confidence`） | ✅ B 路径 | ❌（仅 text 等值） |
| Tag 计数 facet（`scene_agg` / `vendor_agg` 等） | ✅ | ❌ |
| 同一条 tag 内多条件捆绑 | ❌ 候选 P2 | ❌ |
| Tag 时间范围（`tagged_at`） | ❌ 候选 P2 | ⚠️ PG `asset_tags.created_at` 可手写查 |

## 8. 湖仓同步状态

### 8.1 查询同步状态

```bash
curl "$BASE/api/v1/lakehouse/sync-status" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "last_sync_at": "2026-05-15T08:30:00Z",
  "postgres_count": 10000,
  "iceberg_count": 9998,
  "diff_count": 2,
  "diff_pct": 0.02,
  "status": "ok",
  "duration_sec": 45.2,
  "details": {
    "status_distribution_match": true,
    "env_distribution_match": true
  }
}
```

`status` 字段:
- `ok` — 对账通过，差异 < 0.1%
- `warning` — 差异 > 0.1%，需要关注
- `unknown` — 尚未执行对账

## 9. 错误码参考

| HTTP | Code | 说明 |
|------|------|------|
| 400 | `INVALID_ARGUMENT` | 请求格式错误 |
| 400 | `INVALID_FILTER` | 过滤条件无效 |
| 400 | `INVALID_ALGO_KEY` | 算法 key 无效 |
| 401 | `UNAUTHORIZED` | 认证失败 |
| 404 | `ASSET_NOT_FOUND` | 资产不存在 |
| 409 | `ALGO_ALREADY_RUNNING` | 算法已在运行 |
| 409 | `INVALID_STATE_TRANSITION` | 状态转换非法 |
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
  -d '{"mcap_file_id":"mcap-001","start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000060000000000,"reviewer":"alice"}')
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

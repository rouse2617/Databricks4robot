# API 使用指南

本文档提供 data-platform 后端 API 的完整使用说明，包含 curl 示例。

> OpenAPI 规范: [`api/openapi.yaml`](../api/openapi.yaml)

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

先启动 Iceberg + Trino，并运行 Spark 同步任务生成 Iceberg 表：

```bash
make iceberg-up

# Spark 读取 Docker 网络中的 Postgres
make iceberg-mvp

# 当前后端连接宿主机 localhost:5432 时，使用这个命令保持数据源一致
make iceberg-mvp-host
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

注意：当前仍是 Spark 批同步/准 CDC；Trino 负责查询 Iceberg，真正连续入湖后续由 RisingWave/Debezium 补齐。

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
  "reviewer": "alice",
  "owner": "team-a",
  "version": 1,
  "algo_results": {
    "env_analysis@1.0.0:status": "pending",
    "hand_tracking@1.0.0:status": "pending",
    "hand_tracking@1.2.0:status": "pending",
    "action_annotation@1.0.0:status": "blocked"
  },
  "tags": {"priority": "high", "quality": "good", "scene": "indoor"},
  "files": {"raw_mcap": "mcap-001"},
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

响应 `200`: 完整 Asset JSON（同创建响应）

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

允许的过滤/排序字段:

- 标量字段: `asset_id`, `mcap_file_id`, `status`, `reviewer`, `owner`, `type`, `env`, `task`, `created_at`, `updated_at`, `start_timestamp_ns`, `end_timestamp_ns`, `duration_sec`, `delivery_count`, `last_delivered_to`, `last_delivered_at`, `version`
- 生命周期字段: `retention_tier`, `archive_after_days`, `delete_after_days`, `total_size_bytes`, `last_accessed_at`
- Tag 前缀: `tag.<key>`
- 算法结果前缀: `algo.<key>`
- 文件引用前缀: `files.<key>`

兼容别名:

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
    "method": "dagster",
    "run_id": "dagster-run-12345"
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
    "run_id": "dagster-run-12345",
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
    "run_id": "dagster-run-12345"
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

### 2.4 查询算法事件

```bash
# 查询所有事件
curl "$BASE/api/v1/assets/{asset_id}/algo-events" \
  -H "X-Grace-Token: $TOKEN"

# 按算法过滤
curl "$BASE/api/v1/assets/{asset_id}/algo-events?algo_key=env_analysis@1.0.0" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [
    {
      "event_id": "uuid",
      "asset_id": "uuid",
      "algo_key": "env_analysis@1.0.0",
      "prev_status": "pending",
      "new_status": "running",
      "run_id": "dagster-run-12345",
      "created_at": "2026-04-25T10:00:00Z"
    }
  ]
}
```

事件按 `created_at` 降序排列（最新的在前）。

### 2.5 依赖链自动 Unblock

`action_annotation@1.0.0` 依赖三个算法。当所有依赖都完成（status=ok）后，系统自动将 `action_annotation` 从 `blocked` 变为 `pending`。

```bash
# 1. 完成三个依赖
for algo in hand_tracking@1.2.0 head_tracking@1.0.0 body_tracking@1.0.0; do
  curl -X POST "$BASE/api/v1/assets/{id}/algo/$algo/start" \
    -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
    -d '{"method":"dagster"}'
  
  curl -X POST "$BASE/api/v1/assets/{id}/algo/$algo/finish" \
    -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
    -d "{\"status\":\"ok\",\"output_uri\":\"gs://b/$algo.mcap\",\"result_size_bytes\":100,\"extra_fields\":{\"type\":\"v1\"}}"
done

# 2. action_annotation 自动变为 pending，现在可以启动了
curl -X POST "$BASE/api/v1/assets/{id}/algo/action_annotation@1.0.0/start" \
  -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"method":"dagster"}'
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
# 关键词搜索
curl "$BASE/api/v1/search/assets?q=warehouse+rain&page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"

# 关键词 + 结构化过滤
curl "$BASE/api/v1/search/assets?q=indoor&filter=status:eq:approved&filter=owner:eq:alice&page=1&page_size=20" \
  -H "X-Grace-Token: $TOKEN"

# 纯结构化过滤（无关键词）
curl "$BASE/api/v1/search/assets?filter=env:eq:warehouse&page=1&page_size=50" \
  -H "X-Grace-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [...],
  "total": 128,
  "page": 1,
  "page_size": 20,
  "aggregations": {
    "status": {"approved": 85, "rejected": 23, "archived": 20},
    "env": {"indoor": 60, "outdoor": 40, "warehouse": 28}
  }
}
```

查询参数:
- `q` — 全文搜索关键词，匹配 notes/owner/reviewer/task 字段
- `filter` — 结构化过滤，语法与 `/api/v1/assets` 相同（`field:op:value`）
- `page` / `page_size` — 分页参数

注意: 需要 Elasticsearch 服务运行。当 Elasticsearch 不可用时返回 `503`。

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

# 2. Dagster 触发算法处理
curl -X POST "$BASE/api/v1/assets/$ASSET_ID/algo/env_analysis@1.0.0/start" \
  -H "X-Grace-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"method":"dagster","run_id":"run-001"}'

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

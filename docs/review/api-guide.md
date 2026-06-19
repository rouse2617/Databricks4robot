# API 使用指南

本文档提供 data-platform 后端 API 的完整使用说明，包含 curl 示例。

> OpenAPI 规范: [`api/openapi.yaml`](../../api/openapi.yaml)

## ID 格式约定（示例统一）

- `asset_id`：固定 8 位字母数字（`[0-9A-Za-z]{8}`），示例统一用 `aset0001`
- `mcap_file_id`：固定 8 位字母数字（`[0-9A-Za-z]{8}`），示例统一用 `mcap0001`
- `delivery_id` / `event_id` / `eval_result_id`：UUID
- `action_id`：固定 8 位字母数字（`[0-9A-Za-z]{8}`）
- `customer_id`：客户 slug，`^[a-z][a-z0-9_-]{2,31}$`（小写字母开头），示例 `acme_corp`
- `run_id`：算法运行 ID，固定 16 位字母数字（`[0-9A-Za-z]{16}`），须先 `POST /api/v1/algo-runs` 登记后再写入 `asset_algo_latest` / per-asset algo API

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

### Asset type schema

注册型资产类型的 `metadata` 约束可通过 schema 端点发现：

```bash
curl "$BASE/api/v1/asset-types/dataset/schema" \
  -H "X-Databrew-Token: $TOKEN"

curl "$BASE/api/v1/asset-types/annotation_result/schema" \
  -H "X-Databrew-Token: $TOKEN"
```

当前注册类型：

- `dataset`: `format` (`parquet|csv|image|lidar|other`), `record_count`, `size_bytes`, `annotation_status` (`raw|annotated|validated`), `time_range`, `source`
- `annotation_result`: `tool`, `schema_version`, `annotators`, `quality_score`, `coverage`, `artifact_uri`

未知类型返回 `404 ASSET_NOT_FOUND` 风格错误体；旧资产类型仍走兼容校验。

## 基础信息

- 基础 URL: `http://localhost:8080`（本地开发）
- 认证: `/api/v1/*` 默认支持 `X-Databrew-Token`；浏览器端使用 `POST /api/v1/auth/email-login` 写入 `databrew_session` JWT cookie，再访问受保护接口；`POST /api/v1/auth/login` 保留给静态 token 会话登录和兼容场景
- 响应格式: JSON
- 请求 ID: 每个响应包含 `X-Request-ID` header

```bash
# 设置环境变量方便后续使用
export BASE=http://localhost:8080
export TOKEN=dev-token
```

邮箱会话登录（浏览器端实际路径）：

```bash
curl -X POST "$BASE/api/v1/auth/email-login" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@cyberorigin.ai"}' \
  -c /tmp/databrew.cookie

curl "$BASE/api/v1/auth/me" -b /tmp/databrew.cookie
curl -X POST "$BASE/api/v1/auth/logout" -b /tmp/databrew.cookie
```

静态 token 会话登录（SDK 兼容 / 调试可选）：

```bash
curl -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"token":"'"$TOKEN"'"}' \
  -c /tmp/grace.cookie

curl "$BASE/api/v1/auth/me" -b /tmp/grace.cookie
curl -X POST "$BASE/api/v1/auth/logout" -b /tmp/grace.cookie
```

### 接口冒烟测试（安装与运行）

**依赖（无需 pip/npm）**：本机已安装 **`curl`**、**`python3`**（解析 `queries/run` 列表里的 `asset_id`）。可选写入 `RUN_WRITES=1` 时优先使用 **`openssl rand`** 生成 `raw_hash_md5`，否则由 Python 回退生成。

仓库脚本：[`scripts/api-guide-smoke.sh`](../../scripts/api-guide-smoke.sh)（仓库根目录执行）。Makefile：`make api-guide-smoke`（默认本地 `BASE` / `TOKEN`，可用变量覆盖）。

**本地后端**：

```bash
BASE=http://localhost:8080 TOKEN=dev-token bash scripts/api-guide-smoke.sh
# 含创建 MCAP + 资产（需唯一 raw_hash_md5，脚本自动随机）：
RUN_WRITES=1 BASE=http://localhost:8080 TOKEN=dev-token bash scripts/api-guide-smoke.sh
```

**HTTPS dev 网关（API 无 IAP）**：`https://api-cyber-databrew-dev.cyberorigin.ai` 当前仅要求 **`X-Databrew-Token`**（`TOKEN` 环境变量），无需额外 IAP Bearer。

```bash
export BASE=https://api-cyber-databrew-dev.cyberorigin.ai
export TOKEN='<部署环境 DATABREW_TOKEN>'
bash scripts/api-guide-smoke.sh
```

说明：若未来重新启用 IAP，可通过 `IAP_TOKEN` 环境变量向脚本注入 Bearer，流程见 [IAP 程序化认证](https://cloud.google.com/iap/docs/authentication-howto)。

**GKE 集群内冒烟（绕过 IAP，推荐用于验证 dev 后端）**：对 `cyber-databrew-dev` 命名空间内的 `Service/cyber-databrew-backend` 执行同一脚本；`DATABREW_TOKEN` 来自 Secret **`cyber-databrew-secrets`**。一键：

```bash
bash scripts/api-guide-smoke-incluster.sh
# 或指定 context：
KUBE_CONTEXT=gke_green-valley-442103_us-central1_cyber-clust bash scripts/api-guide-smoke-incluster.sh
```

实现：将 [`scripts/api-guide-smoke.sh`](../../scripts/api-guide-smoke.sh) 同步为 ConfigMap，并 apply Job [`deploy/k8s/jobs/api-guide-smoke-job.yaml`](../../deploy/k8s/jobs/api-guide-smoke-job.yaml)。BigQuery 未配置时，部分 Lakehouse 接口会为 **503** / **WARN**，与 api-guide「1.0 未上线」一致。

## Lakehouse / BigQuery 验证

> Lakehouse 链路在 1.0 不可用；2.0 起由 PyIceberg CronJob 通过 outbox 增量入湖（详见《数据平台方案设计》§5.6.2 / §5.11）。本节命令演示的是本地脚手架，仅用于 demo / 验证。

启动 Lakehouse 本地脚手架（BigQuery + BigLake）：

```bash
make iceberg-up

# 触发一次 PyIceberg 演示 MERGE（本地脚本，将 PG 当前快照灌入 Bronze）
make iceberg-mvp
```

检查 BigQuery 查询层状态：

```bash
curl "$BASE/api/v1/lakehouse/status" \
  -H "X-Databrew-Token: $TOKEN"
```

查看 Iceberg 表行数：

```bash
curl "$BASE/api/v1/lakehouse/tables" \
  -H "X-Databrew-Token: $TOKEN"
```

响应至少包含 `bronze_asset_events`；当 Silver export job 已执行并创建外部表时，还会包含 `silver_asset_events_current`。Silver 表语义为每个 `asset_id` 保留 `last_event_seq` 最大的一行，适合下游审计分析避免直接读取 Bronze 重试重复行。未启用 lakehouse 或 Bronze 不可读时返回 `503`；Silver 尚未创建时 `/lakehouse/tables` 仍返回 `200`，只是不包含 Silver 行。

Lakehouse dashboard 读模型接口：

```bash
# 总览：Bronze/Silver/Gold 行数、Gold 最新日期，以及业务侧资产/新增/新鲜度
curl "$BASE/api/v1/lakehouse/overview" \
  -H "X-Databrew-Token: $TOKEN"

# 最近 N 天（默认 30，最大 180）每日新增资产与累计资产（Gold asset_created）
curl "$BASE/api/v1/lakehouse/asset-growth?days=30" \
  -H "X-Databrew-Token: $TOKEN"

# 最近 N 天（默认 30）每日事件统计
curl "$BASE/api/v1/lakehouse/event-daily?days=30" \
  -H "X-Databrew-Token: $TOKEN"

# 某日事件类型占比（date=latest 或 YYYY-MM-DD）
curl "$BASE/api/v1/lakehouse/event-type-share?date=latest" \
  -H "X-Databrew-Token: $TOKEN"
```

五类湖仓查询（当 BigQuery 中对应 Iceberg 表尚未建表或 `LAKEHOUSE_BQ_DATASET` 与仓库不一致时，这些接口仍返回 **HTTP 200**，`items` 为空并带 `note` 说明原因，避免前端按 500 处理）：

```bash
# 某次训练当时用了哪些 asset？
curl "$BASE/api/v1/lakehouse/training-assets?snapshot_id=mvp_hand_tracking_quality_v1" \
  -H "X-Databrew-Token: $TOKEN"

# 某个算法版本变更后，哪些历史 asset 要重算？
curl "$BASE/api/v1/lakehouse/recompute-candidates?algo_key=hand_tracking@1.2.0&target_version=1.3.0" \
  -H "X-Databrew-Token: $TOKEN"

# 某个 tag 是什么时候被算法追加的？
curl "$BASE/api/v1/lakehouse/tag-timeline?tag_key=quality" \
  -H "X-Databrew-Token: $TOKEN"

# 近 7/30/60/90 天资产质量分布（window 可选：7d/30d/60d/90d；默认 30d）
# 数据源：Lakehouse Silver 当前态表（口径与 PG `NOT is_deleted` 保持一致）
curl "$BASE/api/v1/lakehouse/quality-distribution?window=30d" \
  -H "X-Databrew-Token: $TOKEN"

# 某个客户交付过的数据是否能完整回放？
curl "$BASE/api/v1/lakehouse/customer-replay?customer_id=urn:grace:customer:A" \
  -H "X-Databrew-Token: $TOKEN"
```

兼容的静态报告接口仍然保留：

```bash
curl "$BASE/api/v1/lakehouse/report" \
  -H "X-Databrew-Token: $TOKEN"
```

注意：上述 Lakehouse 接口在 1.0 阶段未上线；2.0 起 BigQuery 负责查询 BigLake-managed Iceberg 表，入湖由 Cloud Run Job + PyIceberg 完成。

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
  -H "X-Databrew-Token: $TOKEN" \
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
  -H "X-Databrew-Token: $TOKEN" \
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
  "algo_inputs_uris": {},
  "annot_inputs_uris": {},
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
  -H "X-Databrew-Token: $TOKEN"
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
- **资产多版本（CYB-1016）**：默认只返回当前修订（`is_current=true`，或迁移前 `is_current` 为 NULL 的旧行）。需要历史修订时传 query `?include_history=true` 或 body `scope.include_history: true`。显式过滤可用 `logical_asset_id`、`revision`、`is_current` 字段（PG + ES）。

```bash
# 仅校验（不执行）
curl -sS -X POST "$BASE/api/v1/queries/validate" \
  -H "X-Databrew-Token: $TOKEN" \
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
  -H "X-Databrew-Token: $TOKEN" \
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

# 含历史修订（多版本资产族的全部 revision 行）
curl -sS -X POST "$BASE/api/v1/queries/run?include_history=true" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "schema_version": "v1",
    "scope": {"resource": "assets", "include_history": true},
    "where": {
      "pred": { "field": "logical_asset_id", "op": "eq", "value": "aaaaaaaa" }
    },
    "page": {"page": 1, "page_size": 20}
  }'

# keyword / semantic / similar 也统一从 Query API 进入
curl -sS -X POST "$BASE/api/v1/queries/run" \
  -H "X-Databrew-Token: $TOKEN" \
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
  -H "X-Databrew-Token: $TOKEN" \
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
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`:
```json
{"deleted": true, "asset_id": "..."}
```

软删除将 `lifecycle_state` 设为 `archived`，资产仍可通过 GET 访问，但不会出现在列表查询中。

### 1.6 查询资产关联的交付

```bash
curl "$BASE/api/v1/assets/{asset_id}/deliveries?page=1&page_size=20" \
  -H "X-Databrew-Token: $TOKEN"
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

### 1.7 获取资产溯源（版本历史 + 血缘）

`GET /api/v1/assets/{asset_id}/provenance`

返回同一 `logical_asset_id` 下的全部修订版本、`version_promoted` 时间线，以及与 `GET /assets/{id}/lineage` 相同结构的上下游血缘快照。

```bash
curl "$BASE/api/v1/assets/{asset_id}/provenance" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`（节选）：
```json
{
  "asset_id": "bbbbbbbb",
  "logical_asset_id": "aaaaaaaa",
  "revisions": [
    {"asset_id": "aaaaaaaa", "revision": 1, "is_current": false, "created_at": "2026-05-20T10:00:00Z"},
    {"asset_id": "bbbbbbbb", "revision": 2, "is_current": true, "created_at": "2026-05-21T10:00:00Z"}
  ],
  "version_history": [
    {"version": 1, "asset_id": "aaaaaaaa", "promoted_at": "2026-05-20T10:00:00Z"},
    {"version": 2, "asset_id": "bbbbbbbb", "promoted_at": "2026-05-21T10:00:00Z", "by_run_id": "run-99", "reason": "algo rerun"}
  ],
  "lineage": {
    "asset_id": "bbbbbbbb",
    "upstream": {"mcap_file_id": "z7zyx6sl"},
    "downstream": {"algo_results": [], "deliveries": [], "eval_results": []}
  }
}
```

错误：`404` + `ASSET_NOT_FOUND`（资产不存在）；`400` + `INVALID_ARGUMENT`（非法 `asset_id`）。

### 1.8 查询 logical asset 评分历史（CYB-1100）

`GET /api/v1/logical-assets/{logical_asset_id}/ratings-history`

返回同一 logical asset 下所有未删除 revision 的 `rating.*` 评分指标。评分来源是 `asset_metrics`，只包含 `metric_key` 以 `rating.` 开头的行；没有评分的 revision 仍会返回，`ratings: []`。如果 logical asset 不存在或所有 revision 都已删除，返回 `404 ASSET_NOT_FOUND`。

```bash
curl "$BASE/api/v1/logical-assets/aaaaaaaa/ratings-history" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`:
```json
{
  "logical_asset_id": "aaaaaaaa",
  "items": [
    {
      "asset_id": "aaaaaaaa",
      "revision": 1,
      "is_current": false,
      "lifecycle_state": "ready",
      "created_at": "2026-05-20T10:00:00Z",
      "ratings": [
        {
          "metric_key": "rating.quality_score",
          "metric_type": "float",
          "metric_unit": "stars",
          "metric_value": 4.5,
          "target_type": "asset",
          "target_id": "",
          "eval_name": "manual_rating",
          "eval_version": "v1",
          "run_id": "run1234567890123",
          "source_type": "human",
          "source_name": "reviewer_a",
          "confidence": 0.99,
          "recorded_at": "2026-05-20T10:15:00Z",
          "updated_at": "2026-05-20T10:15:00Z"
        }
      ]
    },
    {
      "asset_id": "bbbbbbbb",
      "revision": 2,
      "is_current": true,
      "lifecycle_state": "ready",
      "created_at": "2026-05-21T10:00:00Z",
      "ratings": []
    }
  ],
  "count": 2
}
```

错误路径：

| 状态 | 错误码 | 触发条件 |
|------|--------|----------|
| `400` | `INVALID_ARGUMENT` | `logical_asset_id` 不是 8 位 ASCII 字母数字 ID |
| `404` | `ASSET_NOT_FOUND` | logical asset 不存在，或没有未删除 revision |
| `503` | `PG_DISABLED` | Postgres 未配置 |

说明：
- 排序稳定：revision 按 `revision ASC, created_at ASC`；同一 revision 内 ratings 按 `metric_key, source_type, source_name, recorded_at` 升序。
- 本接口不返回 `asset_algo_latest.result_score/result_tag`；算法状态仍走算法/资产详情相关接口。
- `asset_metrics` 主键语义决定本接口展示每个 revision 当前保留的评分指标行，不是原始评审事件日志。

### 1.9 获取资产 MCAP 定位（用于预览/流式读取）

`GET /api/v1/assets/{asset_id}/mcap-locator`

为预览 / 流式服务（如 `mcap-preview-service`）提供单次拉取所需信息：资产的时间窗 + 底层 MCAP 对象引用。本接口**只读**、**不签名 URL**，调用方按需自行签名并对 GCS 发起 HTTP `Range` 读取。

```bash
curl "$BASE/api/v1/assets/{asset_id}/mcap-locator" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`：
```json
{
  "asset_id": "S86trk85",
  "lifecycle_state": "ready",
  "mcap": {
    "mcap_file_id": "z7zyx6sl",
    "mcap_uri": "gs://cyber-databrew-dev/uploads/....mcap",
    "size_bytes": 1334578288,
    "raw_hash_md5": "deadbeef..."
  },
  "window": {
    "start_timestamp_ns": 1775435036056908229,
    "end_timestamp_ns":   1775435056056908229,
    "duration_ms": 20000
  },
  "version": 1,
  "updated_at": "2026-05-08T15:32:52Z"
}
```

错误：

- `404 ASSET_NOT_FOUND` / `404 MCAP_FILE_NOT_FOUND`
- `409 ASSET_NOT_PREVIEWABLE`：当资产 `lifecycle_state` 不在 `{created, ready, delivered, archived, superseded}` 之中，或资产尚未关联 `mcap_file_id`
- `503 SERVICE_UNAVAILABLE`：服务端未配置 mcap 仓库（部署问题）

> **预览清单 (manifest)** 由独立部署的 **mcap-preview-service**（见 `services/mcap-preview/`）对外提供：
> `GET /api/v1/preview/assets/{asset_id}/manifest` —— 该服务读取本节的 `mcap-locator` 结果，
> 然后通过 GCS Range 读取仅解析 MCAP 摘要/索引，返回候选视频 topic 与窗口内 chunk 列表，
> **不**返回视频字节本身。该接口**不在** cyber-databrew 后端进程内，部署见 `deploy/k8s/mcap-preview/`。

## 1.10 分层子资产创建（CYB-1222）

分层 API 在父资产下创建子资产，自动写 `asset_relations` 边和 `asset_events`。

### 请求结构

```json
{
  "start_timestamp_ns": 1700000000000000000,
  "end_timestamp_ns":   1700000000100000000,
  "split_method":       "manual",
  "split_run_id":       "",
  "metadata":           {}
}
```

`split_method` 决定边类型：
| 值 | relation_type |
|---|---|
| `algo:*` | `derived_from` |
| 空 / `manual` / `rule:*` | `split_from` |

### 端点列表

| 端点 | 父类型 | 产物 asset_type | 说明 |
|---|---|---|---|
| `POST /api/v1/assets/:id/clips` | segment | clip | 从 segment 切 clip |
| `POST /api/v1/assets/:id/frames` | segment | frame | 从 segment 采样 frame |
| `POST /api/v1/assets/:id/tasks` | segment, task | task | 从 segment 创建 task，或创建 subtask |

> **注意**：`POST /api/v1/assets/:id/actions` 路由已被现有的 data-platform action 端点占用 (handlers/action)，暂不注册分层 action 端点。

### curl 示例

```bash
# 在 segment seg_001 下创建 clip
curl -s -X POST "http://localhost:8080/api/v1/assets/seg_001/clips" \
  -H "X-Databrew-Token: $DATABREW_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "start_timestamp_ns": 1700000000000000000,
    "end_timestamp_ns": 1700000000100000000,
    "split_method": "manual"
  }'

# 在 task task_001 下创建 subtask
curl -s -X POST "http://localhost:8080/api/v1/assets/task_001/tasks" \
  -H "X-Databrew-Token: $DATABREW_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "start_timestamp_ns": 1700000000000000000,
    "end_timestamp_ns": 1700000000200000000,
    "split_method": "algo:hand_track@2.0",
    "split_run_id": "R001"
  }'
```

### 错误响应

- `422 ASSET_HIERARCHY_VIOLATION`：父资产类型不允许创建该子类型（如 task 下创建 clip）
- `404 ASSET_NOT_FOUND`：父资产不存在

## 2. 算法生命周期 (Algo Lifecycle)

### 2.0 算法运行登记 (`algo_runs`，CYB-1018)

Worker 在批量跑算法前先登记 run，再在 per-asset `start`/`finish` 里带上同一 `run_id`（16 位）。
如传入 `input_asset_ids`，每个 ID 必须存在且未被软删除；重复或未知 ID 会返回 `400 INVALID_ARGUMENT`，`details.field` 为 `input_asset_ids`。

```bash
RUN_ID="R001abc123def456"

curl -X POST "$BASE/api/v1/algo-runs" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"run_id\": \"$RUN_ID\",
    \"algo_name\": \"hand_track\",
    \"algo_version\": \"2.0\",
    \"algo_kind\": \"processing\",
    \"triggered_by\": \"manual:ops\",
    \"params\": {\"confidence_threshold\": 0.8}
  }"

curl -X POST "$BASE/api/v1/algo-runs/$RUN_ID/start" \
  -H "X-Databrew-Token: $TOKEN"

# ... per-asset algo start/finish with same run_id ...

curl -X POST "$BASE/api/v1/algo-runs/$RUN_ID/finish" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "ok", "assets_processed": 10, "assets_succeeded": 9, "assets_failed": 1}'

curl "$BASE/api/v1/algo-runs/$RUN_ID" -H "X-Databrew-Token: $TOKEN"
```

Per-asset `finish` 在 `run_id` 为 16 位且已登记时，同事务追加 `algo_run_applied` 事件。

### 2.0.1 列表 / 取消 / 影响资产（CYB-1123）

列表分页统一为 `page/page_size`，响应为 `{items,total,page,page_size}`。非法或超限分页参数会按服务端默认值归一化，响应中的 `page/page_size` 表示实际生效的分页值。

```bash
curl "$BASE/api/v1/algo-runs?page=1&page_size=20&algo_name=hand_track&status=running" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`：
```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "page_size": 20
}
```

重复 `run_id` 创建返回 `409`（不再吞冲突返回旧记录）：

```bash
curl -X POST "$BASE/api/v1/algo-runs" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"run_id\": \"$RUN_ID\",
    \"algo_name\": \"hand_track\",
    \"algo_version\": \"2.0\",
    \"algo_kind\": \"processing\",
    \"triggered_by\": \"manual:ops\"
  }"
```

未知或重复 input asset 创建返回 `400`，并在 details 中列出对应 ID：

```bash
curl -X POST "$BASE/api/v1/algo-runs" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "run_id": "R1536badasset001",
    "algo_name": "hand_track",
    "algo_version": "2.0",
    "algo_kind": "processing",
    "triggered_by": "manual:ops",
    "input_asset_ids": ["DEAD1536"]
  }'

# 400:
# {
#   "code": "INVALID_ARGUMENT",
#   "message": "input_asset_ids contain unknown assets",
#   "details": {
#     "field": "input_asset_ids",
#     "missing_asset_ids": ["DEAD1536"]
#   }
# }
```

可取消 `pending/running` run：

```bash
curl -X POST "$BASE/api/v1/algo-runs/$RUN_ID/cancel" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason":"operator stop"}'
```

查看该 run 影响过的资产：

```bash
curl "$BASE/api/v1/algo-runs/$RUN_ID/affected-assets" \
  -H "X-Databrew-Token: $TOKEN"
```

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
  -H "X-Databrew-Token: $TOKEN" \
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
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "ok"}'
```

成功完成（需要 output 的算法）:
```bash
curl -X POST "$BASE/api/v1/assets/{asset_id}/algo/hand_tracking@1.2.0/finish" \
  -H "X-Databrew-Token: $TOKEN" \
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
  -H "X-Databrew-Token: $TOKEN" \
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
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`:
```json
{"asset_id": "...", "algo_key": "env_analysis@1.0.0", "status": "pending"}
```

只有 `ok` 或 `failed` 状态可以重置。

### 2.4 标签管理（多来源 — CYB-1015）

同一 `(asset_id, tag_key)` 上允许 `human` / `algo_sdk` / `rule_engine` / `system` / `compliance` 多源共存；身份字段由 `backend/config/tag_registry.yaml` `tag_sources[]` 治理。资产响应里：

- `tags`：扁平 `key→value` map（按 `applied_at` 取最新一行；保留兼容旧消费者）
- `tags_detailed[]`：多源真值，每行带 `source_type / source_name / source_version / run_id / applied_at`

```bash
# 1) human 写入 — 必须带 source_name
curl -X POST "$BASE/api/v1/assets/{asset_id}/tags" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"key":"quality","value":"good","source_type":"human","source_name":"labeler_007"}'

# 2) rule_engine 写入 — 必须带 source_name + source_version
curl -X POST "$BASE/api/v1/assets/{asset_id}/tags" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"key":"quality","value":"good","source_type":"rule_engine","source_name":"qc_check","source_version":"1.0"}'

# 3) 只删 human 那行（rule_engine 行保留）
curl -X DELETE "$BASE/api/v1/assets/{asset_id}/tags/quality?source_type=human" \
  -H "X-Databrew-Token: $TOKEN"

# 4) 不带 source_type 时删掉该 key 下所有来源
curl -X DELETE "$BASE/api/v1/assets/{asset_id}/tags/quality" \
  -H "X-Databrew-Token: $TOKEN"

# 5) 查询标签变更历史
curl "$BASE/api/v1/assets/{asset_id}/tags/history?limit=20" \
  -H "X-Databrew-Token: $TOKEN"
```

错误路径：

| 状态 | 错误码 | 触发条件 |
|------|--------|----------|
| `422` | `INVALID_TAG` | key 未在 `tags{}` 注册 / enum 值不合法 |
| `422` | `TAG_SOURCE_INVALID` | `source_type` 未在 `tag_sources[]` 注册，或缺 `requires_source_name` / `requires_source_version` 要求的字段 |
| `409` | `TAG_IMMUTABLE` | 对 `immutable: true` 来源（`algo_sdk` / `compliance`）的已有 `(key, source_type, source_version)` 再次写入 |
| `404` | `ASSET_NOT_FOUND` | asset_id 不存在 |

说明：
- `source_type` 缺省时按 `human` 处理（UI 流的默认）。
- `DELETE /tags/{key}` 对不存在的 key 按幂等成功处理，但不会伪造 `tag_deleted` 事件。
- `GET /tags/history` 仍是统一 `asset_events` 形状，过滤 `tag_upserted / tag_deleted`；event payload 含 `source_name` / `source_version` / `run_id`。

### 2.5 查询资产事件 / 算法事件子集

```bash
# 查询资产完整事件时间线（最新在前）
curl "$BASE/api/v1/assets/{asset_id}/events?limit=50" \
  -H "X-Databrew-Token: $TOKEN"

# timeline 别名端点（语义与 /events 一致）
curl "$BASE/api/v1/assets/{asset_id}/timeline?limit=50" \
  -H "X-Databrew-Token: $TOKEN"

# 只看算法事件
curl "$BASE/api/v1/assets/{asset_id}/events?event_type=algo_*&limit=50" \
  -H "X-Databrew-Token: $TOKEN"

# 按算法过滤，并用 cursor 继续翻下一页
curl "$BASE/api/v1/assets/{asset_id}/events?event_type=algo_*&algo_key=env_analysis@1.0.0&cursor=12345&limit=20" \
  -H "X-Databrew-Token: $TOKEN"
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
- `GET /assets/{id}/timeline` 与 `GET /assets/{id}/events` 完全同形，仅作为时间线命名别名保留给客户端。

### 2.5.0 资产事件 SSE stream（CYB-1099）

`GET /api/v1/assets/{asset_id}/events/stream` 为单个资产打开 `text/event-stream`。服务端按 `event_seq` 升序发送事件；客户端断线重连时可把最后收到的 `event_seq` 放到 `Last-Event-ID` header，服务端只发送 `event_seq > Last-Event-ID` 的事件。

```bash
# 首次订阅
curl -N "$BASE/api/v1/assets/{asset_id}/events/stream" \
  -H "X-Databrew-Token: $TOKEN"

# 断线后从 event_seq=12345 之后恢复
curl -N "$BASE/api/v1/assets/{asset_id}/events/stream" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Last-Event-ID: 12345"
```

SSE frame:

```text
id: 12346
event: asset_updated
data: {"event_id":"550e8400-e29b-41d4-a716-446655440000","event_seq":12346,"event_type":"asset_updated","asset_id":"b9a5a281","event_payload":{"field":"status"},"occurred_at":"2026-05-23T10:00:00Z"}
```

错误路径：

| 状态 | 错误码 | 触发条件 |
|------|--------|----------|
| `400` | `INVALID_ARGUMENT` | 非法 `{asset_id}` 或 `Last-Event-ID` 不是非负整数 |
| `404` | `ASSET_NOT_FOUND` | 资产不存在或已软删除 |

### 2.5.1 跨资产审计搜索（CYB-1097）

`GET /api/v1/audit/search` 在 `asset_events` 上做跨资产只读搜索，适合排查某个 actor、event type、run id 或时间窗口内的资产事件。结果按 `event_seq` 降序返回；`cursor` 语义为继续取 `event_seq < cursor` 的更老事件。

```bash
# 最新 50 条审计事件
curl "$BASE/api/v1/audit/search?limit=50" \
  -H "X-Databrew-Token: $TOKEN"

# 按事件类型 + run_id 搜索
curl "$BASE/api/v1/audit/search?event_type=algo_finished&run_id=run1234567890123&limit=20" \
  -H "X-Databrew-Token: $TOKEN"

# 按 actor + 时间窗口搜索
curl "$BASE/api/v1/audit/search?actor=alice&time_from=2026-05-01T00:00:00Z&time_to=2026-05-23T23:59:59Z" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [
    {
      "event_id": "550e8400-e29b-41d4-a716-446655440000",
      "event_seq": 12345,
      "event_type": "algo_finished",
      "aggregate_type": "asset",
      "asset_id": "b9a5a281",
      "mcap_file_id": "mcap0001",
      "event_source": "backend",
      "actor_type": "user",
      "actor_id": "alice",
      "run_id": "run1234567890123",
      "occurred_at": "2026-05-23T10:00:00Z",
      "created_at": "2026-05-23T10:00:00Z"
    }
  ],
  "limit": 20,
  "next_cursor": 12345
}
```

错误路径：

| 状态 | 错误码 | 触发条件 |
|------|--------|----------|
| `400` | `INVALID_ARGUMENT` | `limit <= 0` / 非整数 `limit` / 非 int64 `cursor` / 非 RFC3339 `time_from` 或 `time_to` / `time_from > time_to` |
| `503` | `SERVICE_UNAVAILABLE` | audit search 存储未配置 |

说明：
- `actor` 当前匹配 `asset_events.actor_id`，大小写不敏感且支持包含匹配。
- `event_type` 与 `run_id` 为精确匹配。
- `limit` 默认 `50`，最大 `200`；超过最大值会按 `200` 执行。

### 2.5.2 合规血缘搜索（CYB-1098）

`GET /api/v1/audit/lineage-search` 在 `asset_relations` 上做只读递归查询，适合合规审计时追踪某个资产的上游来源或下游影响面。默认 `direction=both`、`depth=10`，最大深度为 `50`。

方向语义：
- `upstream`: 从 `child_asset_id` 往 `parent_asset_id` 查。
- `downstream`: 从 `parent_asset_id` 往 `child_asset_id` 查。
- `both`: 先查 upstream，再查 downstream，并在响应中用 `nodes[].direction` 区分。

默认关系类型为依赖/结构血缘：`split_from`, `contains`, `derived_from`, `merged_from`, `sampled_from`。如需更窄范围，可传 `relation_types=derived_from,split_from`；`revision_of` 只在显式请求时包含。

```bash
# 查某个资产的上下游血缘，允许空结果
curl "$BASE/api/v1/audit/lineage-search?asset_id=b9a5a281&direction=both&depth=3" \
  -H "X-Databrew-Token: $TOKEN"

# 只查 derived/split 关系的下游影响面
curl "$BASE/api/v1/audit/lineage-search?asset_id=b9a5a281&direction=downstream&relation_types=derived_from,split_from" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`:
```json
{
  "asset_id": "b9a5a281",
  "direction": "downstream",
  "depth": 3,
  "relation_types": ["derived_from", "split_from"],
  "nodes": [
    {
      "asset_id": "clip001",
      "parent_asset_id": "b9a5a281",
      "child_asset_id": "clip001",
      "relation_type": "derived_from",
      "direction": "downstream",
      "depth": 1,
      "method": "algo",
      "algo_name": "hand_track",
      "algo_version": "2.0.0",
      "run_id": "run1234567890123",
      "created_at": "2026-05-23T10:00:00Z"
    }
  ],
  "count": 1
}
```

空结果仍返回 `200`，并保持 `nodes: []`:
```json
{
  "asset_id": "asset-without-relations",
  "direction": "both",
  "depth": 10,
  "relation_types": ["split_from", "contains", "derived_from", "merged_from", "sampled_from"],
  "nodes": [],
  "count": 0
}
```

错误路径：

| 状态 | 错误码 | 触发条件 |
|------|--------|----------|
| `400` | `INVALID_ARGUMENT` | 缺少 `asset_id` / `direction` 不是 `upstream,downstream,both` / `depth` 非正整数 / `relation_types` 包含不支持的值 |
| `500` | `INTERNAL` | 数据库查询或扫描失败 |
| `503` | `SERVICE_UNAVAILABLE` | audit lineage 存储未配置 |

说明：
- 递归查询带 path 防环，并受 `depth` 限制，避免环形关系导致重复遍历。
- 响应中的 `parent_asset_id` / `child_asset_id` 是命中的 `asset_relations` 原始边，`asset_id` 是本次 traversal 到达的相关资产。
- 当前接口只读，不写 `asset_events` 或其它审计副作用表。

### 2.6 依赖链自动 Unblock

`action_annotation@1.0.0` 依赖三个算法。当所有依赖都完成（status=ok）后，系统自动将 `action_annotation` 从 `blocked` 变为 `pending`。

```bash
# 1. 完成三个依赖
for algo in hand_tracking@1.2.0 head_tracking@1.0.0 body_tracking@1.0.0; do
  curl -X POST "$BASE/api/v1/assets/{id}/algo/$algo/start" \
    -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
    -d '{"method":"k8s_job"}'

  curl -X POST "$BASE/api/v1/assets/{id}/algo/$algo/finish" \
    -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
    -d "{\"status\":\"ok\",\"output_uri\":\"gs://b/$algo.mcap\",\"result_size_bytes\":100,\"extra_fields\":{\"type\":\"v1\"}}"
done

# 2. action_annotation 自动变为 pending，现在可以启动了
curl -X POST "$BASE/api/v1/assets/{id}/algo/action_annotation@1.0.0/start" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"method":"k8s_job"}'
```

## 2.7 Action 段（seg 内时间分段标注）

> 状态：**Phase 1 已上线**：`POST` / `GET` / `PATCH` / `DELETE /assets/:id/actions[/:action_id]`（含 `at` / `from` / `to` / `label` 过滤；PATCH/DELETE 走同事务发 `action_upserted` / `action_deleted`，可用 `expected_version` 做 CAS）。平台级反查 `GET /actions` 与 `GET /lookup` 仍在 §2.7.4 / §2.7.5 标记为「待上线」。

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
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
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

幂等：当前实现支持 `external_id`——同 `(asset_id, source_name, external_id)` 已存在时返回 `409 CONCURRENT_CONFLICT`（不自动覆盖；如需更新已有行请用 §2.7.2 的 PATCH）。`Idempotency-Key` header **未**在 actions 端点强制（与 `POST /deliveries` 不同）。

### 2.7.2 修改 / 删除

PATCH 是部分更新——未传字段保持不变。`expected_version` 可选；填了就走 CAS（不匹配返回 `409 CONCURRENT_CONFLICT`），不填则跳过版本守卫直接读最新版本。

```bash
# 改 label / labels / description / 时间窗；同事务追加 action_upserted
curl -X PATCH "$BASE/api/v1/assets/{asset_id}/actions/{action_id}" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"primary_label":"pickup_left","end_ns":2100000000,"expected_version":1}'

# 软删（同事务追加 action_deleted）
curl -X DELETE "$BASE/api/v1/assets/{asset_id}/actions/{action_id}?expected_version=2" \
  -H "X-Databrew-Token: $TOKEN"
```

错误映射：父 seg 或 action 不存在 → `404 ASSET_NOT_FOUND`；版本冲突 → `409 CONCURRENT_CONFLICT`；父非 segment / 时间窗超出 seg / label 不在注册表 → `422 INVALID_ACTION`；非法 source_type / `end_ns < start_ns` → `400 INVALID_ARGUMENT`。

### 2.7.3 查询：seg 内时间轴 / 时间戳点查 / 区间查

```bash
# 列出 seg 的所有 action（默认按 start_ns 排序）
curl "$BASE/api/v1/assets/{asset_id}/actions" -H "X-Databrew-Token: $TOKEN"

# 时间戳点查：哪些 action 覆盖时间点 t
curl "$BASE/api/v1/assets/{asset_id}/actions?at=1500000000" -H "X-Databrew-Token: $TOKEN"

# 区间查：与 [from,to] 重叠的 action
curl "$BASE/api/v1/assets/{asset_id}/actions?from=0&to=5000000000&label=pickup" \
  -H "X-Databrew-Token: $TOKEN"
```

### 2.7.4 平台级反查："含某 action 的 seg"（待上线）

```bash
curl "$BASE/api/v1/actions?label=overtake&from=&to=&page=1&page_size=20" \
  -H "X-Databrew-Token: $TOKEN"
```

主路径走 ES（seg 文档 `actions[]` nested 字段，CDC 反向投影）；PG 兜底走 `(primary_label)` 索引。

### 2.7.5 时间戳一站式 lookup（前端时间轴用，待上线）

```bash
# 返回该绝对时间戳落在哪条 seg + 该时间点上的所有 action
curl "$BASE/api/v1/lookup?at=1700000000000000000" -H "X-Databrew-Token: $TOKEN"
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

## 2.8 客户管理 (Customers)

客户主数据表 `customers`；`deliveries.customer_id` 为外键。创建客户时 `customer_id` 须满足 slug 规则（见文首 ID 约定）。

### 2.8.1 创建客户

```bash
curl -X POST "$BASE/api/v1/customers" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "acme_corp",
    "display_name": "Acme Robotics",
    "legal_name": "Acme Robotics Ltd",
    "status": "active",
    "region": "us-west",
    "sla_tier": "standard",
    "account_owner": "team-a"
  }'
```

响应 `201`：完整 `Customer` 对象（含 `row_version`、`created_at` 等）。

| 状态 | code | 触发 |
|------|------|------|
| 400 | `INVALID_ARGUMENT` | body 非法或 `customer_id` 不符合 slug |
| 409 | `INVALID_ARGUMENT` | `customer_id` 已存在 |

### 2.8.2 获取客户

```bash
curl "$BASE/api/v1/customers/acme_corp" \
  -H "X-Databrew-Token: $TOKEN"
```

| 状态 | code | 触发 |
|------|------|------|
| 404 | `CUSTOMER_NOT_FOUND` | 无此客户 |

### 2.8.3 更新客户

```bash
curl -X PATCH "$BASE/api/v1/customers/acme_corp" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"display_name": "Acme Robotics (US)", "sla_tier": "premium"}'
```

仅发送需修改字段。`409` `CONCURRENT_CONFLICT` 表示乐观锁冲突。

### 2.8.4 列出客户

`GET /api/v1/customers` 使用 cursor 分页（`limit` + `cursor`）：

```bash
curl "$BASE/api/v1/customers?status=active&sla_tier=premium&region=us-west&limit=20" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`：
```json
{
  "items": [],
  "limit": 20,
  "next_cursor": ""
}
```

---

## 2.9 交付规则 (`delivery-rules`，CYB-1020)

客户专属或全局（`customer_id` 省略）的 `query_dsl` 规则；`enforce_mode=block` 时在 **`POST /deliveries`** 提交前校验。`customers.exclude_tags` 同样会拦截（虚拟规则 `customer.exclude_tags`）。

### 2.9.1 创建规则

```bash
curl -X POST "$BASE/api/v1/delivery-rules" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "cust_no_pii",
    "owner": "ops@databrew",
    "customer_id": "acme_corp",
    "enforce_mode": "block",
    "query_dsl": {
      "where": [{"field": "tag.compliance.pii", "op": "eq", "value": "true"}]
    }
  }'
```

响应 `201`：`DeliveryRule`（含 `rule_id` UUID）。

### 2.9.2 列出规则

```bash
curl "$BASE/api/v1/delivery-rules?customer_id=acme_corp" \
  -H "X-Databrew-Token: $TOKEN"
```

### 2.9.3 与交付联动

当资产命中 block 规则时，`POST /deliveries` 返回 **`422`**、`code`: **`DELIVERY_RULE_FAILED`**，`details.violations` 列出 `asset_id`、`rule_id`、`rule_name`、`reason`。

---

## 3. 交付管理 (Deliveries)

交付前 **`customer_id` 必须在 `customers` 表中存在**（迁移 `029` 已为历史 `deliveries` 回填占位行）。不存在 → `422` `customer not found`。命中 **§2.9** 规则 → `422` `DELIVERY_RULE_FAILED`。

### 3.1 创建交付

```bash
curl -X POST "$BASE/api/v1/deliveries" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-12345" \
  -d '{
    "asset_ids": ["aset0001", "aset0002"],
    "customer_id": "acme_corp",
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
- 相同 key + 相同 body → 返回之前的结果 (`201`)，不会创建第二条 delivery
- 相同 key + 不同 body → `409` 冲突，且在写入 delivery 副作用前拒绝
- 缺少 key → `400`

`asset_ids` 中的每一项须为合法 **资产 `asset_id`**（8 位字母数字）；非法格式 → `400`（避免写入 `delivery_items` 时数据库报错）。

| 状态 | code | 触发 |
|------|------|------|
| 422 | `DELIVERY_RULE_FAILED` | 资产命中 block 规则或 `exclude_tags`（见 `details.violations`） |

### 3.1.1 C2 两阶段交付（draft → items → commit）

```bash
# 1) 创建草稿（pending）
curl -X POST "$BASE/api/v1/deliveries/draft" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "acme_corp",
    "asset_ids": ["aset0001"],
    "note": "batch draft"
  }'
```

```bash
# 2) 向 pending 草稿追加资产
curl -X POST "$BASE/api/v1/deliveries/{delivery_id}/items" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"asset_ids":["aset0002","aset0003"]}'
```

```bash
# 3) 提交草稿（expected_revision 做并发保护）
curl -X POST "$BASE/api/v1/deliveries/{delivery_id}/commit" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expected_revision":1,"approved_by":"ops@databrew"}'
```

语义说明（CYB-1123）：
- `draft` / `retry` 仅创建 `delivery_items`，不会提前计入资产 `delivery_count/last_delivered_*`
- 只有最终 `commit`（或一次式 `POST /deliveries`）才会刷新“已交付”索引与事件
- 向 pending 草稿重复追加已存在资产时，`asset_count` 按实际唯一 `delivery_items` 计算，不因重复请求膨胀
- `commit` 时若 `expected_revision` 不匹配返回 `409 CONCURRENT_CONFLICT`

### 3.2 获取交付详情

```bash
curl "$BASE/api/v1/deliveries/{delivery_id}" \
  -H "X-Databrew-Token: $TOKEN"
```

### 3.3 按客户查询交付

```bash
curl "$BASE/api/v1/customers/{customer_id}/deliveries?page=1&page_size=20" \
  -H "X-Databrew-Token: $TOKEN"
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
  -H "X-Databrew-Token: $TOKEN"

# 按状态过滤
curl "$BASE/api/v1/deliveries?page=1&page_size=20&status=delivered" \
  -H "X-Databrew-Token: $TOKEN"

# 按客户过滤（CYB-1014）
curl "$BASE/api/v1/deliveries?page=1&page_size=20&customer_id=acme_corp" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`:
```json
{
  "items": [
    {
      "delivery_id": "cccccccc-3333-4000-8000-000000000001",
      "customer_id": "acme_corp",
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

### 3.5 取消 / 重试 / 客户确认（CYB-1104/1105/1106）

```bash
# 取消：允许 pending / delivered / accepted
curl -X POST "$BASE/api/v1/deliveries/{delivery_id}/cancel" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cancelled_by":"ops@databrew","cancel_reason":"customer revoked"}'
```

```bash
# 重试：仅允许 failed / cancelled，返回一个新的 pending delivery
curl -X POST "$BASE/api/v1/deliveries/{delivery_id}/retry" \
  -H "X-Databrew-Token: $TOKEN"
```

```bash
# 客户确认：delivered -> accepted
curl -X POST "$BASE/api/v1/deliveries/{delivery_id}/ack" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"acknowledged_by":"customer.ops"}'
```

错误路径：

| 状态 | code | 触发 |
|------|------|------|
| 404 | `DELIVERY_NOT_FOUND` | delivery 不存在 |
| 422 | `INVALID_STATE` | 状态不允许该操作（如对 pending 执行 ack） |
| 422 | `INVALID_STATE_TRANSITION` | 状态机禁止的跃迁 |

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

注意：`POST /internal/commit-segments` 在 Phase 0 **未**挂在 `/api/v1` 的 `X-Databrew-Token` 中间件上（见 `backend/routes/routes.go` 中 Internal 路由注释）。文档里其余 **`/api/v1/*`** 示例仍需 `X-Databrew-Token` 或有效 `databrew_session` cookie。

## 5. MCAP 文件管理

### 5.1 创建 MCAP 文件

```bash
curl -X POST "$BASE/api/v1/mcap-files" \
  -H "X-Databrew-Token: $TOKEN" \
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
  -H "X-Databrew-Token: $TOKEN"
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
  -H "X-Databrew-Token: $TOKEN"
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
  -H "X-Databrew-Token: $TOKEN"
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
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`：`items` 为写入 `assets.lifecycle_state` 时允许的取值（与 `migrations/009_lifecycle_state_check.sql` 一致）。

### 6.2 标签注册表

**`tag_registry.yaml` 是 tag 的「白名单字典」**：只有在这里声明过的 **key**（例如 `priority`、`scene`）才允许作为 `tag_key` 出现在受校验的写入路径里；**`enum` 类型还限制 value 必须在给出的列表中。** 源文件路径：`backend/config/tag_registry.yaml`；可在进程运行中热重载。

```bash
curl "$BASE/api/v1/tag-registry" \
  -H "X-Databrew-Token: $TOKEN"
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
`GET /api/v1/search` / `GET /api/v1/search/assets` 是 ES 直连兼容入口；全文、keyword、semantic、similar、facet 的正式主路径仍建议通过 Query API 进入：

- `POST /api/v1/queries/validate`
- `POST /api/v1/queries/run`

建议：

- keyword：`mode=keyword` + `_fulltext` predicate
- semantic：`mode=semantic`
- similar：`mode=similar`
- 结构化 tag / mcap / algo / action 条件：统一放进 Query IR 的 `where`

#### 7.1.1 血缘过滤（ES lineage projection）

搜索兼容入口支持基于 ES 投影的直接血缘过滤。`lineage_with` 指定种子资产；`lineage_direction` 可为 `upstream`、`downstream`、`both`；`lineage_depth` 默认 `1`、最大 `3`；`relation_types` 可传逗号分隔或重复参数。

```bash
curl "$BASE/api/v1/search/assets?lineage_with=b9a5a281&lineage_direction=downstream&lineage_depth=2&relation_types=derived_from,pipeline_output" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`：`items[]` 为匹配的关联资产，每条结果保留 `lineage_upstream_ids` / `lineage_downstream_ids` / `lineage_relation_types`，并额外包含 `lineage_relation` 标记本次命中的种子资产、方向、深度和关系类型。

错误示例：

```bash
curl "$BASE/api/v1/search/assets?lineage_with=b9a5a281&lineage_direction=sideways" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `400`：`lineage_direction must be upstream, downstream, or both`。

### 7.2 Admin：全量从 PG 重建 ES（不推进 outbox 游标）

当 **`assets` 索引尚不存在** 时，清空索引步骤会对 `_delete_by_query` 的 **404** 视为「已删除 0 条」，从而允许首次全量写入。

当前先复用 **`X-Databrew-Token`**；不再额外要求 `X-Admin-Token`。

#### 7.2.0 只读：Outbox / DLQ 行数统计

用于排查「PG 资产数 vs ES 文档数」不一致时，`asset_events` 里各 `publish_state` 的分布以及 `outbox_dlq` 归档表行数：

```bash
curl -sS "$BASE/api/v1/admin/search/outbox-stats" \
  -H "X-Databrew-Token: $TOKEN" | jq .
```

响应示例：

```json
{
  "publish_state_counts": {
    "pending": 0,
    "published": 1234567,
    "processing": 0,
    "dlq": 42
  },
  "outbox_dlq_rows": 100
}
```

说明：`publish_state_counts["dlq"]` 是 **`asset_events` 表里标记为 dlq 的行**；`outbox_dlq_rows` 是 **`outbox_dlq` 表**里归档副本的行数（两者语义不同，可能都非零）。

#### 7.2.1 同步模式（兼容）

```bash
curl -sS -X POST "$BASE/api/v1/admin/search/reindex" \
  -H "X-Databrew-Token: $TOKEN" \
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

#### 7.2.2 异步任务模式（推荐）

> Cloud Run 场景下推荐使用任务模式，支持 **停止**、**断点续开**、**进度轮询**。

**创建任务：**

```bash
curl -sS -X POST "$BASE/api/v1/admin/search/reindex-jobs" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"dry_run": false, "page_size": 200}' | jq .
```

响应 `202`（示例）：

```json
{
  "id": "rj_4f85f0b8-9aa1-48e0-8b16-0dcb7a3b4a9d",
  "status": "queued",
  "dry_run": false,
  "page_size": 200,
  "next_page": 1,
  "progress_pct": 0
}
```

**查询任务进度：**

```bash
curl -sS "$BASE/api/v1/admin/search/reindex-jobs/$JOB_ID" \
  -H "X-Databrew-Token: $TOKEN" | jq .
```

关键字段：

- `status`: `queued|running|paused|succeeded|failed`
- `assets_scanned / total_assets`：进度分母分子
- `next_page`：续开时从该页继续
- `stop_requested`：是否已收到停止信号

**停止任务（安全点停机）：**

```bash
curl -sS -X POST "$BASE/api/v1/admin/search/reindex-jobs/$JOB_ID/stop" \
  -H "X-Databrew-Token: $TOKEN" | jq .
```

**断点续开：**

```bash
curl -sS -X POST "$BASE/api/v1/admin/search/reindex-jobs/$JOB_ID/resume" \
  -H "X-Databrew-Token: $TOKEN" | jq .
```

### 7.2.3 Internal：硬删除 assets / mcap_files

> ⚠️ 这是**物理删除**接口，与公共 `DELETE /api/v1/assets/:id`（soft delete）行为不同。仅在导入失控、需要彻底清理时使用。
> 鉴权同样走 `X-Databrew-Token`；CDC 会自动把删除事件传播到 Elasticsearch，不需要再单独 reindex。

**点删除**：

```bash
curl -sS -X DELETE "$BASE/api/v1/internal/assets/$ASSET_ID" \
  -H "X-Databrew-Token: $TOKEN" | jq .
```

```json
{
  "asset_id": "AbCd1234",
  "deleted": {
    "asset_metrics": 0,
    "asset_eval_results": 0,
    "asset_algo_events": 0,
    "asset_algo_latest": 0,
    "asset_tags": 3,
    "actions": 0,
    "delivery_items": 0,
    "asset_relations": 0,
    "asset_events_by_asset": 4,
    "assets": 1,
    "asset_events_by_mcap": 0,
    "asset_eval_results_by_mcap": 0,
    "mcap_files": 0
  }
}
```

- `404 ASSET_NOT_FOUND` — 资产不存在（包括已经被 hard-delete 过的情况）。
- `mcap_files` 始终保持 `0`：点删除不动 mcap 元数据。

**批量删除（先 dry-run 验证规模）**：

```bash
# 按 import_batch 标签
curl -sS -X POST "$BASE/api/v1/internal/assets:batch_delete" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{
    "import_batch": "collector-2026-05-11",
    "include_mcap_files": true,
    "dry_run": true
  }' | jq .

# 或按显式列表
curl -sS -X POST "$BASE/api/v1/internal/assets:batch_delete" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{
    "asset_ids": ["AbCd1234", "EfGh5678"],
    "dry_run": true
  }' | jq .
```

```json
{
  "dry_run": true,
  "include_mcap_files": true,
  "resolved": {
    "asset_ids_count": 267028,
    "mcap_file_ids_count": 124632,
    "import_batch": "collector-2026-05-11"
  },
  "deleted": {
    "asset_metrics": 0,
    "asset_eval_results": 0,
    "asset_algo_events": 0,
    "asset_algo_latest": 0,
    "asset_tags": 0,
    "actions": 0,
    "delivery_items": 0,
    "asset_relations": 0,
    "asset_events_by_asset": 267028,
    "assets": 267028,
    "asset_events_by_mcap": 124632,
    "asset_eval_results_by_mcap": 0,
    "mcap_files": 124632
  }
}
```

确认数字合理后把 `"dry_run": true` 改成 `false` 真删。

请求字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `asset_ids` | `string[]` | 显式列表。与 `import_batch` 二选一。 |
| `import_batch` | `string` | 匹配 `metadata->>'import_batch'`。与 `asset_ids` 二选一。 |
| `include_mcap_files` | `bool` | 仅在 `import_batch` 模式生效；同步清掉对应 `mcap_files`。 |
| `dry_run` | `bool` | 为 `true` 时只统计、不修改。 |
| `chunk_size` | `int` | 每个事务最多删除多少 id，默认 `1000`。 |

错误码：

| HTTP | code | 触发条件 |
|---|---|---|
| 400 | `INVALID_ARGUMENT` | `asset_ids` 和 `import_batch` 都给了或都没给；body 不是合法 JSON。 |
| 401 | `UNAUTHORIZED` | `X-Databrew-Token` 缺失或不匹配。 |
| 500 | `INTERNAL_ERROR` | 删除 mcap_files 时仍有 `assets` 行引用（说明 `asset_ids` 未覆盖全部使用方）。 |

### 7.3 Prometheus 指标

同一进程暴露 `GET /metrics`（无认证；建议仅内网可达）。指标前缀包括 `gin_` (HTTP 请求时延/状态码)、`go_` (运行时) 等。

> ⚠️ `outbox_worker_*` 系列指标（`outbox_worker_pending_total` 等）已随旧 polling worker 删除；当前链路为 `asset_events -> outbox relay -> Pub/Sub -> ES subscriber`。如有监控依赖旧指标名，需更新告警规则。

### 7.4 搜索索引同步状态

```bash
curl "$BASE/api/v1/search/sync-status" \
  -H "X-Databrew-Token: $TOKEN"
```

响应 `200`:

```json
{
  "elasticsearch_ok": true,
  "outbox_relay_enabled": true,
  "outbox_es_subscriber_enabled": true,
  "search_index_mode": "outbox_es_subscriber",
  "env": "development",
  "admin_search_enabled": true
}
```

字段说明：

- `search_index_mode` 枚举：`unavailable` / `outbox_es_subscriber` / `local_reconcile` / `manual`。
- `outbox_relay_enabled` 表示是否启用 PG `asset_events` 到 Pub/Sub 的 relay。
- `outbox_es_subscriber_enabled` 表示当前进程是否启用 Pub/Sub 到 Elasticsearch 的订阅消费。
- `admin_search_enabled` 表示当前环境是否开放搜索管理接口（重建索引、任务历史、PG↔ES 对账）。
- Pub/Sub EventSource 使用 `OUTBOX_TRANSPORT=pubsub`、`PUBSUB_PROJECT` 和 `OUTBOX_ES_SUBSCRIPTION`。消息 handler 成功时 ACK，handler 返回错误时 NACK，由 Pub/Sub 按订阅策略重试；缺少 project 或 subscription 时启动阶段会记录清晰配置错误并不启动该 subscriber。

#### OpenLineage / Marquez emitter（CYB-1109）

OpenLineage emitter 默认关闭，不新增 HTTP API。启用后它从独立 Pub/Sub subscription 消费 `asset_events`，把包含 `asset_id` 的事件映射为 OpenLineage JSON，并 `POST` 到 Marquez/OpenLineage endpoint。handler 成功返回 2xx 时 ACK；endpoint 非 2xx 或网络错误时返回错误给 Pub/Sub subscriber，消息会 NACK 并按订阅策略重试。

配置：

| 环境变量 | 默认 | 说明 |
|----------|------|------|
| `OPENLINEAGE_EMITTER_ENABLED` | `false` | `true` 时启动 emitter |
| `OPENLINEAGE_ENDPOINT` | 空 | Marquez/OpenLineage HTTP endpoint |
| `OPENLINEAGE_SUBSCRIPTION` | 空 | 独立 Pub/Sub subscription，避免与 ES consumer 竞争 |
| `OPENLINEAGE_NAMESPACE` | `cyber-databrew` | OpenLineage job/dataset namespace 前缀 |
| `OPENLINEAGE_PRODUCER` | `cyber-databrew` | OpenLineage `producer` |
| `OPENLINEAGE_TIMEOUT_MS` | `5000` | 单次 POST timeout |

#### 同步进度（PG / ES 与 Outbox）

```bash
curl "$BASE/api/v1/search/sync-progress" \
  -H "X-Databrew-Token: $TOKEN"
```

响应中与 Outbox 相关的字段含义（避免把「pending 总数」误判成积压卡死）：

- `outbox_pending_events`：`publish_state = pending` 的总数，**包含**仍在 relay safety lag 窗口内、relay 还不能认领的事件。
- `outbox_pending_claimable`：已满足时间条件、当前可被 relay 认领的 pending 数（与 `ClaimPendingSafe` 的 pending 分支一致）。
- `outbox_processing_events`：已被认领、`publish_state = processing` 的事件数。
- `outbox_relay_safety_lag_sec`：配置的 safety lag（秒）；`outbox_pending_events - outbox_pending_claimable` 多为该窗口内的写入缓冲。
- `oldest_pending_age_sec`：最老 pending 事件的年龄（秒）。
- `pg_max_event_seq` / `outbox_published_max_seq` / `seq_lag`：**PG → MQ** 段水位。`seq_lag = pg_max_event_seq - outbox_published_max_seq`，反映 PG 侧仍处于 pending/processing 的事件数量；O(1) 查询，规模到 50B 行也能高频轮询，是首选告警指标。
- `es_applied_min_seq` / `consumer_lag`：**MQ → ES** 段水位。订阅方每次 `_bulk` 成功后，按 `FNV(asset_id) % OUTBOX_ES_CHECKPOINT_SHARDS` 把该批的 max(event_seq) UPSERT 到 `es_sync_checkpoint`；`es_applied_min_seq = MIN(applied_seq)` 是跨分片保守水位（小于它的事件都已被 ES 应用）。`consumer_lag = outbox_published_max_seq - es_applied_min_seq`。当尚未集齐全部分片（新部署冷启动）时 `es_applied_min_seq=0`、`consumer_lag=0`，避免「乐观虚低」误导告警。
  - **空闲分片自动追平**：`updated_at` 距今超过 `OUTBOX_ES_CHECKPOINT_IDLE_AFTER_SEC`（默认 300s）的分片，在 MIN 计算时按 `outbox_published_max_seq` 处理，避免低吞吐 key space 的「僵尸分片」长期拉高 `consumer_lag`。设为 0 可禁用。

两段水位连起来即可定位瓶颈：

| 现象 | 解读 |
|---|---|
| `seq_lag` 高位上升 | relay / Pub/Sub publish 卡了 |
| `seq_lag` 平、`consumer_lag` 高位上升 | ES 消费端卡了（ES 拒写、subscriber 异常、单分片热点） |
| 两段都正常但 `oldest_pending_age_sec` 一直高 | DLQ / 卡死的单条消息，看 `publish_state_counts` |

仅有较大的 `outbox_pending_events` 而可认领接近 0，多为 safety lag 缓冲下的正常现象。

> 注意：`/api/v1/internal/assets/:id` 与 `/api/v1/internal/assets:batch_delete` 走的是硬删除，绕过 outbox。当前进程会在删除成功后**尽力**同步删除 ES 中对应文档（失败仅记日志，不影响接口返回）；PG 仍是真相源，必要时通过 reindex/reconcile 兜底。

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

> 当前 ES facet 聚合使用 keyword 维度字段本身（例如 `owner`、`lifecycle_state`、`mcap.vendor_id`）。如果你本地索引是旧 mapping，`queries/run` 可能返回 warning 并回退纯 PG。先调用一次 `POST /api/v1/admin/search/reindex` 或重建本地 `assets` 索引。

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
  -H "X-Databrew-Token: $TOKEN"
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
  -H "X-Databrew-Token: $TOKEN" \
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
  -H "X-Databrew-Token: $TOKEN" \
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

标准 JSON 错误体：

```json
{
  "code": "INVALID_ARGUMENT",
  "message": "invalid request body",
  "request_id": "req-xxx",
  "details": {"field": "asset_id"}
}
```

`details` 可省略；`request_id` 与响应头 `X-Request-ID` 对应。Python SDK 会将这些字段暴露为 `e.code`、`e.message`、`e.request_id`、`e.details`，并按 HTTP 状态码设置 `e.http_status` 和 typed exception。

| HTTP | Code | 说明 |
|------|------|------|
| 400 | `INVALID_ARGUMENT` | 请求格式错误 |
| 400 | `INVALID_FILTER` | 过滤条件无效 |
| 400 | `INVALID_ALGO_KEY` | 算法 key 无效 |
| 401 | `UNAUTHORIZED` | 认证失败 |
| 404 | `ASSET_NOT_FOUND` | 资产不存在 |
| 404 | `MCAP_FILE_NOT_FOUND` | MCAP 文件不存在（如 mcap-locator 找不到底层文件） |
| 409 | `ASSET_NOT_PREVIEWABLE` | 资产 lifecycle_state 不支持预览（如 `created` / `failed`） |
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
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"mcap_file_id":"mcap0001","start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000060000000000,"reviewer":"alice"}')
ASSET_ID=$(echo $ASSET | python3 -c "import sys,json; print(json.load(sys.stdin)['asset_id'])")

# 2. 外部算法 worker 触发处理
curl -X POST "$BASE/api/v1/assets/$ASSET_ID/algo/env_analysis@1.0.0/start" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"method":"k8s_job","run_id":"run-001"}'

# 3. 算法完成回调
curl -X POST "$BASE/api/v1/assets/$ASSET_ID/algo/env_analysis@1.0.0/finish" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"status":"ok","run_id":"run-001"}'

# 4. 交付给客户
curl -X POST "$BASE/api/v1/deliveries" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -H "Idempotency-Key: delivery-$(date +%s)" \
  -d "{\"asset_ids\":[\"$ASSET_ID\"],\"customer_id\":\"cust-001\"}"
```


## 11. Pipeline 编排

### 组件注册表（F1）

组件注册表管理可拖拽的 pipeline 组件（Docker 镜像）。

```bash
# 列出组件（支持搜索/筛选；/api/v1/components 为兼容别名）
curl -s "$BASE/api/v1/pipeline-components" \
  -H "X-Databrew-Token: $TOKEN"
# 按名称搜索: ?q=processor
# 按来源筛选: ?source=custom|system
# 组合: ?q=processor&source=custom

# 响应: {"items": [{...}, ...]}

# 创建组件
curl -X POST "$BASE/api/v1/pipeline-components" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{
    "name": "my-processor",
    "type": "container",
    "description": "My processing component",
    "image": "registry.example.com/my-processor",
    "tag": "v1",
    "command": ["python", "/app/main.py"],
    "args": ["--input", "{{inputs.asset}}"],
    "env": {"MODE": "batch"},
    "inputPorts": [{"name": "input", "type": "asset"}],
    "outputPorts": [{"name": "output", "type": "asset"}],
    "resources": {
      "cpu": "4000m",
      "memory": "16Gi",
      "disk": "50Gi",
      "gpu": "1",
      "computeTier": "gpu-l4"
    }
  }'
# 响应: 201 + Component 对象
# 必填字段: name, type(container|script|resource|suspend), image
# resources.gpu 会在部署时映射为 Argo/Kubernetes `limits.nvidia.com/gpu`；
# resources.computeTier 是 DataBrew 调度、配额、成本策略使用的元数据。
# 部署/运行/批量任务下发时，后端会按当前执行目标的资源上限校验 cpu/memory/disk/gpu；
# 超出 dev 能力会返回 400/invalid argument，提示最大可用规格并拒绝创建 Argo Workflow。

# 获取组件详情
curl -s "$BASE/api/v1/pipeline-components/<ID>" \
  -H "X-Databrew-Token: $TOKEN"

# 更新组件
curl -X PUT "$BASE/api/v1/pipeline-components/<ID>" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"name": "my-processor", "type": "container", "image": "registry.example.com/my-processor", "tag": "v2"}'

# 删除组件
curl -X DELETE "$BASE/api/v1/pipeline-components/<ID>" \
  -H "X-Databrew-Token: $TOKEN"
# 响应: 204

# 校验失败示例
curl -i -X POST "$BASE/api/v1/pipeline-components" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"missing-image","type":"container"}'
# 响应: 400 + 标准错误体
```

### 配置中心（ConfigManagement）

配置中心管理用户自己的单文件配置，不属于组件子对象。组件继续只描述运行镜像和运行参数；流水线需要配置时引用 `configId`。配置文件内容保存在 PostgreSQL，单版本内容上限为 1 MiB。

文件内容不可原地改写：查看旧版本时可看到当时的文件内容；编辑任意旧版本应提交为新版本。

```bash
# 列出当前用户的配置（普通用户默认只看自己的配置；legacy SDK/admin 可传 owner）
curl -s "$BASE/api/v1/pipeline-configs?q=detector&lifecycle=ready" \
  -H "X-Databrew-Token: $TOKEN"
# 响应: {"items": [{...}, ...]}

# 创建配置和第一个文件版本
curl -X POST "$BASE/api/v1/pipeline-configs" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{
    "name": "detector.yaml",
    "description": "Detector thresholds for validation",
    "tags": ["vision", "smoke"],
    "lifecycle": "ready",
    "content": "threshold: 0.82\nwindow: 5\n",
    "summary": "initial thresholds"
  }'
# 响应: 201 + Config 对象；versions 只返回摘要，不返回 content

# 查看配置详情和版本摘要
curl -s "$BASE/api/v1/pipeline-configs/<CONFIG_ID>" \
  -H "X-Databrew-Token: $TOKEN"

# 查看 v1 文件内容
curl -s "$BASE/api/v1/pipeline-configs/<CONFIG_ID>/versions/1" \
  -H "X-Databrew-Token: $TOKEN"
# 响应: Version 对象，包含 content

# 编辑文件：创建 v2，不改写 v1
curl -X POST "$BASE/api/v1/pipeline-configs/<CONFIG_ID>/versions" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{
    "status": "ready",
    "content": "threshold: 0.90\nwindow: 5\n",
    "summary": "raise detector threshold"
  }'
# 响应: 201 + Version 摘要，content 省略；currentVersion 会推进到 2

# 仅更新元数据，不改文件内容
curl -X PUT "$BASE/api/v1/pipeline-configs/<CONFIG_ID>" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{
    "name": "detector.yaml",
    "description": "Detector thresholds for validation",
    "tags": ["vision", "prod"],
    "fileType": "yaml",
    "lifecycle": "ready"
  }'

# 废弃配置（不硬删除，避免破坏历史流水线复现）
curl -X POST "$BASE/api/v1/pipeline-configs/<CONFIG_ID>/deprecate" \
  -H "X-Databrew-Token: $TOKEN"

# 校验失败示例：缺少文件内容
curl -i -X POST "$BASE/api/v1/pipeline-configs" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"bad.yaml","lifecycle":"ready"}'
# 响应: 400 + 标准错误体
```

### 组件版本库（ComponentRelease）

组件版本库记录由 CI/平台生成的 task 构建版本。普通 UI 应查询 `selectable=true`，只展示已经通过基础校验且 digest 固化的版本；repo、commit、image digest 等技术字段放在详情里。

CI 推荐使用专用 `X-Databrew-CI-Token: $DATABREW_CI_INGEST_TOKEN` 调用 sync；管理员/调试工具仍可使用普通 `X-Databrew-Token`。`source` 是 batch 级构建上下文，DataBrew 会把它作为每个 item 的默认 source metadata，并写入 technical metadata。

```bash
# 平台/CI 同步一个生成版本
curl -X POST "$BASE/api/v1/pipeline-component-releases/sync" \
  -H "X-Databrew-CI-Token: $DATABREW_CI_INGEST_TOKEN" -H "Content-Type: application/json" \
  -d '{
    "source": {
      "provider": "cloud-build",
      "repo": "CyberOrigin2077/automated-processing-gcloud",
      "ref": "refs/heads/main",
      "refType": "branch",
      "commit": "abc1234abc1234abc1234abc1234abc1234abc1234",
      "buildId": "1e86eab9-9250-41c5-b905-8ff3c6de23af",
      "trigger": "hand-detect-yolov26m-build-trigger"
    },
    "items": [{
      "componentId": "hand-detect-yolov26m",
      "taskName": "hand-detect-yolov26m",
      "taskPath": "tasks/hand_detect_yolov26m",
      "releaseLabel": "main-abc1234",
      "runtimeImage": "us-central1-docker.pkg.dev/my-project/video-proc-images/hand-detect-yolov26m@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "runtimeSnapshot": {
        "command": ["python", "src/main.py"],
        "inputPorts": [{"name": "input", "type": "asset"}],
        "outputPorts": [{"name": "output", "type": "asset"}],
        "resources": {"cpu": "14000m", "memory": "55Gi", "gpu": "1"}
      }
    }]
  }'
# 响应: {"items":[{...,"channel":"candidate","validationStatus":"passed","selectable":true}]}

# tag 构建可传 `refType=tag`，DataBrew 会归为 `prod`，UI 标识为“线上版本”。
# commit 构建可传 `refType=commit` 或只传 commit hash，普通用户用 q 搜 commit 即可找到测试版本。

# 普通 UI 列出可选版本
curl -s "$BASE/api/v1/pipeline-component-releases?selectable=true&q=abc1234" \
  -H "X-Databrew-Token: $TOKEN"
# q 支持 component/task/display/release label/source commit/ref/repo/build id/image tag/digest/runtime image。

# 查看技术详情
curl -s "$BASE/api/v1/pipeline-component-releases/<RELEASE_ID>" \
  -H "X-Databrew-Token: $TOKEN"

# digest 缺失时仍可入库审计，但不会出现在 selectable=true 的正常选择器里
curl -X POST "$BASE/api/v1/pipeline-component-releases/sync" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"items":[{"componentId":"demo","releaseLabel":"pr-1-abc123","runtimeImage":"registry/demo:abc123","runtimeSnapshot":{"command":["python","main.py"],"resources":{"cpu":"1"}}}]}'
# 响应: {"items":[{...,"validationStatus":"failed","selectable":false,"validationErrors":["imageDigest is required and must be sha256 pinned"]}]}
```

### Pipeline 版本对比（F2.12）

对比两个 pipeline template 版本的 node 和 edge 差异。

```bash
curl -s "$BASE/api/v1/pipelines/<ID1>/diff/<ID2>" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "added_nodes": [{"id": "new-step", "component": {"name": "...", "image": "..."}}],
#   "removed_nodes": [{"id": "old-step"}],
#   "modified_nodes": [{"id": "changed-step", "component": {"name": "...", "image": "..."}}],
#   "added_edges": [{"source": "a.out", "target": "b.in"}],
#   "removed_edges": [{"source": "c.out", "target": "d.in"}]
# }
# 404: template 不存在
```

### Pipeline 执行目标与资产驱动运行（CYB-1532/CYB-1534）

查询可用执行目标。CYB-1534 后执行目标会落库，默认目标会由 backend 当前 Argo
namespace 自动 bootstrap；后续可扩展为多个 cluster/namespace/service account。

```bash
curl -s "$BASE/api/v1/execution-targets" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "items": [
#     {
#       "id": "default",
#       "name": "Default Argo target",
#       "cluster": "default",
#       "namespace": "cyber-databrew-dev",
#       "argoServerConfigured": true,
#       "status": "available",
#       "isDefault": true
#     }
#   ]
# }
```

查询可挂载到节点的运行时资源。该接口只返回平台允许的资源目录，不返回 secret
内容；节点 DSL 只保存 `resourceId`、`mountPath`、`readOnly`，后端在部署时解析为
SecretProviderClass、PVC 或 emptyDir。

```bash
curl -s "$BASE/api/v1/pipeline/runtime-mounts" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "secrets": [
#     {
#       "id": "platform-db-secrets",
#       "name": "Platform DB 密钥",
#       "kind": "secretProviderClass",
#       "secretProviderClass": "databrew-platform-db-creds", # pragma: allowlist secret
#       "defaultMountPath": "/mnt/secrets",
#       "readOnly": true
#     }
#   ],
#   "storage": [
#     {
#       "id": "scratch-emptydir",
#       "name": "临时工作目录",
#       "kind": "emptyDir",
#       "defaultMountPath": "/workspace/scratch",
#       "readOnly": false,
#       "allowWrite": true
#     }
#   ]
# }
```

部署时如果节点引用未知 `resourceId`、目标环境不支持该资源、挂载路径不是绝对路径、
挂载到 `/tmp/outputs` 等保留目录、同节点挂载路径冲突，或只读资源请求写入，接口返回
`400 INVALID_ARGUMENT`，Workflow 不会提交到 Argo。

如果平台侧 catalog JSON 配置不合法，例如 secret 资源缺少 `secretProviderClass` 或 PVC
资源缺少 `pvcName`，`GET /api/v1/pipeline/runtime-mounts` 返回 `500`，前端应展示资源加载失败而不是允许用户保存空绑定。

按模板提交 first-class pipeline run。`target_id` 可省略，省略时使用默认执行目标；
`asset_ids` 可为空，但 UI 应把空资产运行标识为 no-asset run。显式 `asset_ids`
必须存在且未被软删除；重复或未知 ID 返回 `400 INVALID_ARGUMENT`，`details.field`
为 `asset_ids`。该接口会同时写入兼容 deployment 记录，旧前端 `/deployments`
仍可读取。模板保存是快照式版本管理：同名 pipeline 每次保存都会生成新的
`version`；保存模板前会按运行契约做校验：同一个目标输入端口不能被多个上游同时连接
（fan-in 需要给 join 节点配置不同输入端口），被下游消费的输出端口必须由生产节点写入
`/tmp/outputs/<port>`。校验失败返回 `400 INVALID_ARGUMENT`，避免无效 Workflow 提交到
Argo。列表默认只返回同名模板的最新版本，`/pipelines/<ID>/versions`
返回该模板名称下的全部历史版本。运行时可传 `version` 选择历史版本；响应会返回
实际绑定的 `templateId` 与 `templateVersion`。

```bash
# 保存同名模板两次，会得到 v1 / v2 两个 snapshot。
curl -X POST "$BASE/api/v1/pipelines" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "daily-ingest",
    "pipeline": {
      "name": "daily-ingest",
      "nodes": [
        {
          "id": "step-1",
          "component": {
            "name": "echo",
            "image": "alpine:3.18",
            "command": ["sh", "-c"],
            "args": [{"name": "script", "value": "echo ok"}]
          },
          "runtimeConfig": {
            "mode": "saved",
            "configId": "<PIPELINE_CONFIG_ID>",
            "version": 1,
            "fileName": "detector.yaml",
            "mountPath": "/workspace/configs",
            "targetFilename": "detector.yaml"
          },
          "inputs": [],
          "outputs": []
        }
      ],
      "edges": []
    }
  }'

curl -s "$BASE/api/v1/pipelines/<TEMPLATE_ID>/versions" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "items": [
#     {"id": "tpl-v2", "name": "daily-ingest", "version": 2, "nodeCount": 1},
#     {"id": "tpl-v1", "name": "daily-ingest", "version": 1, "nodeCount": 1}
#   ]
# }
```

`runtimeConfig` 是节点级配置绑定。它引用配置库里已经 ready 的配置版本，并随
pipeline template snapshot 保存。部署时后端会把该配置挂载到声明它的节点，不会挂到
其他节点；同一次运行选择的 `asset_ids` 仍作为 run 级上下文注入到所有节点 Pod。

```bash
curl -X POST "$BASE/api/v1/pipeline-runs/template/<TEMPLATE_ID>" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_id": "default",
    "version": 1,
    "asset_ids": ["SDKT0202", "SDKT0101"]
  }'

# 响应示例:
# {
#   "id": "run-123",
#   "templateId": "tpl-123",
#   "templateVersion": 1,
#   "pipelineName": "asset-pipeline",
#   "workflowName": "asset-pipeline-a1b2c3",
#   "executionTargetId": "default",
#   "status": "Pending",
#   "assetIds": ["SDKT0202", "SDKT0101"],
#   "assetCount": 2,
#   "noAssetRun": false,
#   "argoNamespace": "cyber-databrew-dev",
#   "executionTarget": {
#     "id": "default",
#     "namespace": "cyber-databrew-dev",
#     "status": "available"
#   },
#   "nodes": []
# }
# 400: asset 不存在或 target_id 不支持
# 404: template 不存在
# 503: Argo backend 未配置
```

`configSelection` 仍为可选兼容字段，适合旧调用方或明确需要全局 fallback 的高级场景。
正常产品模型应优先在 pipeline node 上保存 `runtimeConfig`。存在节点级
`runtimeConfig` 时，节点自己的配置优先；deploy-level `configSelection` 不会覆盖该节点。

兼容字段存在时，deploy/runtime 会把配置内容投影成一次性 ConfigMap。没有节点级配置的
旧 pipeline 会在所有 pipeline step 容器里同时注入：

- 挂载文件：`<mountPath>/<targetFilename>`
- 环境变量：`PIPELINE_CONFIG_PATH`、`PIPELINE_CONFIG_FILENAME`、`PIPELINE_CONFIG_SOURCE`
- 若来源是平台已保存配置（`mode=saved`），额外包含 `PIPELINE_CONFIG_ID` 与
  `PIPELINE_CONFIG_VERSION`

三种来源模式：

- `saved`：选择平台已保存配置，需传 `configId`，可选 `version`
- `upload`：上传本地文件，需传 `fileName + content`
- `inline`：在线编辑草稿，需传 `fileName + content`

查询 first-class run 列表和详情。详情会尽量刷新 Argo phase，并在 workflow
包含节点状态时返回 `nodes`；每个 pod 节点会带 `logRef`，供前端跳转日志。
当后端设置 `PRICING_CONFIG_PATH` 且节点存在 Argo `resourcesDuration` 时，
节点会返回 `estimatedCostUsd`，run 详情会返回 `totalEstimatedCost`。未配置
pricing、节点无资源耗时或迁移未填充历史数据时，这两个字段会省略或为 `null`，
接口不会失败。

```bash
curl -s "$BASE/api/v1/pipeline-runs" \
  -H "X-Databrew-Token: $TOKEN"

curl -s "$BASE/api/v1/pipeline-runs/<RUN_ID>" \
  -H "X-Databrew-Token: $TOKEN"

# 404: run 不存在
```

查询 run 事件时间线。事件由 DataBrew 的 `pipeline_run_events` 账本返回，
按 `sequence` 正序排列；刷新或 watcher 重启不会重复写入相同状态变化。
`cursor` 使用上一页返回的 `nextCursor`，`limit` 范围为 1-500。
常见事件包括 `run_submitted`、`run_scheduled`、`workflow_created`、
`workflow_observed`、`workflow_phase_changed`、`pod_created`、
`pod_phase_changed`、`node_started`、`node_succeeded`、`node_failed`、
`node_error`、`run_completed`、`run_failed`、`run_retry_requested`、
`run_resubmitted`、`run_delete_requested`、`run_deleted` 和
`run_delete_failed`。

```bash
curl -s "$BASE/api/v1/pipeline-runs/<RUN_ID>/events?limit=100" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "items": [
#     {
#       "id": "evt-123",
#       "runId": "run-123",
#       "workflowName": "asset-pipeline-a1b2c3",
#       "eventType": "run_submitted",
#       "subjectType": "run",
#       "subjectId": "run-123",
#       "status": "Pending",
#       "message": "pipeline run submitted",
#       "sequence": 1,
#       "occurredAt": "2026-06-03T10:01:00Z",
#       "observedAt": "2026-06-03T10:01:00Z",
#       "createdAt": "2026-06-03T10:01:00Z"
#     },
#     {
#       "id": "evt-124",
#       "runId": "run-123",
#       "workflowName": "asset-pipeline-a1b2c3",
#       "eventType": "node_failed",
#       "subjectType": "node",
#       "subjectId": "asset-pipeline-a1b2c3-123456",
#       "status": "Failed",
#       "message": "image pull failed",
#       "sequence": 2,
#       "occurredAt": "2026-06-03T10:03:00Z",
#       "observedAt": "2026-06-03T10:03:05Z",
#       "createdAt": "2026-06-03T10:03:05Z"
#     }
#   ],
#   "total": 2
# }

curl -s "$BASE/api/v1/pipeline-runs/<RUN_ID>/events?subjectType=node&eventType=node_failed" \
  -H "X-Databrew-Token: $TOKEN"

curl -s "$BASE/api/v1/pipeline-runs/<RUN_ID>/events?status=Failed&q=image" \
  -H "X-Databrew-Token: $TOKEN"

# 400: limit/cursor 非法
# 404: run 不存在
```

查询 watcher 健康状态。该状态用于判断 DataBrew run ledger 是否仍在从
Argo 同步运行状态；当 Argo Workflow 被 TTL 清理后，详情页仍应优先展示
DataBrew 已落库的 run/node/event 历史。

```bash
curl -s "$BASE/api/v1/pipeline-runs/watcher/status" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "id": "default",
#   "lastSyncedAt": "2026-06-03T10:03:10Z",
#   "lastScanStartedAt": "2026-06-03T10:03:10Z",
#   "lastScanFinishedAt": "2026-06-03T10:03:11Z",
#   "lastSuccessAt": "2026-06-03T10:03:11Z",
#   "activeScanLimit": 100,
#   "lastSyncedRunCount": 4,
#   "consecutiveFailures": 0,
#   "totalScans": 128,
#   "totalErrors": 1,
#   "scanLagSeconds": 10,
#   "healthy": true,
#   "stale": false,
#   "updatedAt": "2026-06-03T10:03:11Z"
# }
```

查询资产 × 节点明细。当前 P0.2 使用 `run.assetIds × pipeline_run_nodes`
派生明细；no-asset run 会返回 `assetId=no-asset`。成本字段是估算值，
`costSource=estimated_resource_duration` 表示来自 Argo resource duration
和 DataBrew pricing 配置，`not_available` 表示没有足够数据。

```bash
curl -s "$BASE/api/v1/pipeline-runs/<RUN_ID>/asset-nodes?limit=100&orderBy=cost" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "items": [
#     {
#       "id": "an-123",
#       "runId": "run-123",
#       "assetId": "SDKT0202",
#       "pipelineNodeId": "step-a",
#       "status": "Succeeded",
#       "podName": "asset-pipeline-step-a",
#       "logRef": "/api/v1/workflows/asset-pipeline/logs?podName=...",
#       "estimatedCostUsd": 0.0123,
#       "costSource": "estimated_resource_duration"
#     }
#   ],
#   "total": 1,
#   "summary": {
#     "assetCount": 1,
#     "nodeCount": 1,
#     "statuses": {"Succeeded": 1},
#     "totalEstimatedCostUsd": 0.0123,
#     "costSource": "estimated_resource_duration"
#   }
# }
```

查询 run 级成本汇总。该接口是估算和审计视图，不是 GCP Billing 对账。

```bash
curl -s "$BASE/api/v1/pipeline-runs/<RUN_ID>/cost-summary" \
  -H "X-Databrew-Token: $TOKEN"

# 404: run 不存在
```

旧 deployment API 仍可用。它返回兼容字段，并会从 first-class run 投影状态。

```bash
curl -X POST "$BASE/api/v1/deploy/template/<TEMPLATE_ID>" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_id": "default",
    "asset_ids": ["SDKT0202", "SDKT0101"]
  }'

# 响应会包含兼容 deployment 字段以及运行上下文:
# {
#   "id": "dep-123",
#   "pipelineName": "asset-pipeline",
#   "workflowName": "asset-pipeline-a1b2c3",
#   "status": "Pending",
#   "assetIds": ["SDKT0202", "SDKT0101"],
#   "assetCount": 2,
#   "executionTarget": {
#     "id": "default",
#     "namespace": "cyber-databrew-dev",
#     "status": "available"
#   }
# }
# 400: asset 不存在或 target_id 不支持
# 404: template 不存在
```

部署记录的 `status` 会尽量跟随 Argo workflow phase。若旧部署记录仍存在但
Argo Workflow CR 已被 TTL 清理，API 会返回 `Expired`，避免历史记录长期显示
过期的 `Running`/`Pending` 状态；这类记录可以通过 retry 重新提交。

### Workflow 节点 Pod 诊断（CYB-1559）

DataBrew 通过后端代理读取 GKE/Kubernetes Pod 诊断信息，前端不直接持有
kubeconfig 或 Kubernetes token。Cloud Run 后端需要配置可访问目标 GKE API 的
后端凭据和 namespace 级 RBAC，至少允许读取 `pods` 与 `events`。

```bash
curl -s "$BASE/api/v1/workflows/<WORKFLOW_NAME>/nodes/<NODE_ID>/pod" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "cluster": "dev-gke",
#   "namespace": "cyber-databrew-dev",
#   "podName": "my-workflow-step-a-123456",
#   "podIp": "10.2.3.4",
#   "serviceAccountName": "workflow",
#   "restartCount": 1,
#   "containers": [
#     {"name": "main", "image": "alpine:3.20", "ready": true, "restartCount": 1, "state": "Running"}
#   ],
#   "podConditions": [
#     {"type": "Ready", "status": "True", "reason": "ContainersReady"}
#   ],
#   "podEvents": [
#     {"type": "Normal", "reason": "Pulled", "message": "container image pulled", "count": 1}
#   ]
# }
```

错误语义：

- `404 WORKFLOW_NOT_FOUND`：Argo workflow 不存在。
- `404 NODE_NOT_FOUND`：workflow 存在，但节点不存在或不是 Pod 节点。
- `404 POD_NOT_FOUND`：节点解析出的 Pod 不存在，常见于尚未创建或已 TTL 清理。
- `403 K8S_FORBIDDEN`：后端 Kubernetes 凭据缺少当前 namespace 的 Pod/Event 读取权限。
- `503 K8S_UNAVAILABLE`：Cloud Run 到 GKE API 不通，或 Kubernetes client 未配置。

临时 dev bridge 可以使用后端 secret 注入 `K8S_API_ENDPOINT`、
`K8S_BEARER_TOKEN`、可选 `K8S_CA_FILE` 与 `K8S_CLUSTER_NAME`；长期生产方案
应使用 GCP identity/RBAC，而不是把长效 token 暴露给浏览器或前端配置。

### Workflow 节点 Pod 终端会话（CYB-1569）

Pod terminal 是后端持有 Kubernetes 凭据的受控调试入口。默认关闭；只有
pipeline run 的 execution target snapshot 显式配置 terminal policy 后，后端才会
创建 session；实际开启时通常配置 execution target 的
`resourceDefaults.terminal`，新建 run 会把它写入 target snapshot。浏览器只拿
DataBrew session id 和一次性 attach URL，不会拿
kubeconfig、ServiceAccount token 或 GKE 凭据。

创建会话：

```bash
curl -s -X POST "$BASE/api/v1/workflows/<WORKFLOW_NAME>/nodes/<NODE_ID>/terminal-sessions" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"command":"sh"}'

# 201:
# {
#   "id": "7f0d...",
#   "workflowName": "my-workflow",
#   "nodeId": "my-workflow-123",
#   "podName": "my-workflow-step-a-123456",
#   "namespace": "cyber-databrew-dev",
#   "command": "sh",
#   "status": "created",
#   "attachUrl": "/api/v1/pod-terminal/sessions/7f0d.../attach?token=...",
#   "expiresAt": "2026-06-03T03:00:00Z",
#   "createdAt": "2026-06-03T02:45:00Z"
# }
```

查询与终止：

```bash
curl -s "$BASE/api/v1/pod-terminal/sessions/<SESSION_ID>" \
  -H "X-Databrew-Token: $TOKEN"

curl -s -X POST "$BASE/api/v1/pod-terminal/sessions/<SESSION_ID>/terminate" \
  -H "X-Databrew-Token: $TOKEN"
```

WebSocket attach：

```text
GET /api/v1/pod-terminal/sessions/<SESSION_ID>/attach?token=<ONE_TIME_TOKEN>
```

`attach` 使用创建 session 时返回的一次性 token 授权；浏览器原生 WebSocket
不能设置 `X-Databrew-Token`，因此该握手不再要求额外 API header。session
创建、查询和终止接口仍要求正常 DataBrew API 认证。

Frame 是 JSON 文本：

```json
{"type":"status","status":"attached"}
{"type":"stdout","data":"..."}
{"type":"stderr","data":"..."}
{"type":"exit","exitCode":0,"reason":"completed"}
{"type":"error","code":"POD_EXEC_UNAVAILABLE","message":"..."}
```

当后端配置了 Kubernetes exec client 时，attach 会把允许命令的 stdout/stderr/exit
以 JSON frame 返回给前端。未配置或不可达时，attach 返回受控的
`POD_EXEC_UNAVAILABLE` / `POD_EXEC_FAILED` frame，而不是暴露底层 Kubernetes
凭据或原始敏感信息。

错误语义：

- `403 POD_EXEC_FORBIDDEN`：execution target 未启用 terminal，或命令不在 allowlist。
- `404 WORKFLOW_NOT_FOUND` / `NODE_NOT_FOUND` / `POD_NOT_FOUND`：无法定位 workflow、节点或 Pod。
- `409 POD_EXEC_UNAVAILABLE`：Pod 已完成、删除或不支持 exec。
- `503 K8S_UNAVAILABLE`：后端 Kubernetes exec path 未配置或不可达。

### 查询资源用量元数据（F5.8）

查询 workflow 各 pod 的 Argo resource duration、manifest request/limit，以及数据来源。
当前后端未接入 Kubernetes Metrics API，因此 `live_metrics_available=false`，
`source.metrics="unavailable"`；`cpu_usage` / `memory_usage` 是兼容字段，含义为
Argo resource duration，不是真实实时 CPU/Mem 样本或计费值。

```bash
curl -s "$BASE/api/v1/deployments/<DEPLOYMENT_ID>/resources" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "deployment_id": "dep-123",
#   "workflow_name": "my-pipeline-a1b2c3",
#   "status": "Running",
#   "observed_at": "2026-06-02T08:00:00Z",
#   "source": {"workflow": "argo-live", "metrics": "unavailable", "spec": "stored-manifest"},
#   "live_metrics_available": false,
#   "pods": [
#     {
#       "pod_name": "my-pipeline-a1b2c3-step-process-12345",
#       "live_metrics_available": false,
#       "requests": {"cpu": "500m", "memory": "256Mi"},
#       "limits": {"cpu": "1000m", "memory": "512Mi"},
#       "resource_duration": {"cpu": "30s", "memory": "45s"},
#       "cpu_usage": "30s",
#       "memory_usage": "45s",
#       "cpu_request": "500m",
#       "memory_request": "256Mi",
#       "cpu_limit": "1000m",
#       "memory_limit": "512Mi"
#     }
#   ]
# }
# 404: deployment 不存在
```

也可以直接按 workflow 或 workflow pod node 查询：

```bash
curl -s "$BASE/api/v1/workflows/<WORKFLOW_NAME>/resources" \
  -H "X-Databrew-Token: $TOKEN"

curl -s "$BASE/api/v1/workflows/<WORKFLOW_NAME>/nodes/<NODE_ID>/resources" \
  -H "X-Databrew-Token: $TOKEN"
```

### 从部署记录保存为 Template（F7.8）

把一次 deployment 的 pipeline 配置保存为新 template。

```bash
curl -X POST "$BASE/api/v1/deployments/<DEPLOYMENT_ID>/save-template" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{"name": "my-saved-template"}'

# 响应: 201 + Template 对象
# name 可选，默认 pipeline_name + "-from-deployment"
# 404: deployment 不存在
```

### 注册 Pipeline 产出资产（F4.3）

Pipeline 容器处理完数据后，通过回调 API 注册产出资产。

```bash
# 容器内部：PIPELINE_DEPLOYMENT_ID 由平台自动注入
curl -X POST "$BASE/api/v1/pipeline-assets" \
  -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
  -d '{
    "deployment_id": "'"$PIPELINE_DEPLOYMENT_ID"'",
    "node_id": "step-processor",
    "asset_id": "",
    "storage_uri": "gs://bucket/output/result.png",
    "asset_type": "image",
    "files": {"result": "gs://bucket/output/result.png"},
    "metadata": {"resolution": "1920x1080"}
  }'

# 正常响应: 201 + Asset 对象
# 404: deployment_id 不存在
```

### 查询产出血缘（F4.5）

从 asset 追溯是哪个 pipeline 的哪次 run 产生的。

```bash
curl -s "$BASE/api/v1/assets/<ASSET_ID>/pipeline-lineage" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "asset_id": "aset0001",
#   "deployment_id": "abc-123",
#   "pipeline_name": "my-pipeline",
#   "workflow_name": "my-pipeline-a1b2c3",
#   "node_id": "step-processor",
#   "input_assets": ["input-001", "input-002"],
#   "produced_at": "2026-05-27T12:00:00Z"
# }
```

缺失资产示例：

```bash
curl -X POST "$BASE/api/v1/deploy/template/<TEMPLATE_ID>" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"target_id":"default","asset_ids":["DEAD1536"]}'

# 400:
# {
#   "code": "INVALID_ARGUMENT",
#   "message": "asset_ids contain unknown assets",
#   "details": {
#     "field": "asset_ids",
#     "missing_asset_ids": ["DEAD1536"]
#   }
# }
```

### Workflow 监控与操作

直接查看和操作 Argo workflow。所有操作接口成功时返回 `{"message":"ok"}`；Argo 返回错误时，后端返回标准错误体。

```bash
# 列出 workflows
curl -s "$BASE/api/v1/workflows" \
  -H "X-Databrew-Token: $TOKEN"

# 查看 workflow 详情（含 labels、progress、estimatedDuration、nodes、Pod 节点 podName，以及标准化 DAG edges）
# nodes 会合并 Argo runtime 节点和 spec 中尚未启动的 DAG task；未启动 task 以 Pending 返回，便于前端立即渲染完整 DAG。
curl -s "$BASE/api/v1/workflows/<WORKFLOW_NAME>" \
  -H "X-Databrew-Token: $TOKEN"

# edges 示例：用于前端绘制 DAG 连线；Failed -> Omitted 以及 runtime 尚未创建的下游 Pending task 依赖也会保留
# {
#   "edges": [
#     {"id":"e-step-1-step-2","source":"step-1","target":"step-2","kind":"dag"}
#   ]
# }

# 查看节点日志。nodeId 传 workflow detail 返回的节点 id；后端会解析实际 Kubernetes podName。
# 默认 tailLines=200、limitBytes=262144；超过服务端最大值会 clamp。
# live Argo/Kubernetes 日志没有稳定历史 cursor，因此响应会明确标记 pagination.available=false。
curl -s "$BASE/api/v1/workflows/<WORKFLOW_NAME>/logs?nodeId=<NODE_ID>&tailLines=200&limitBytes=262144" \
  -H "X-Databrew-Token: $TOKEN"

# 响应示例:
# {
#   "workflowName": "wf-1",
#   "nodeId": "wf-1-step-123",
#   "podName": "wf-1-step-123",
#   "container": "main",
#   "source": "argo-live",
#   "logs": "...\n",
#   "lineCount": 42,
#   "truncated": false,
#   "nextCursor": null,
#   "pagination": {
#     "available": false,
#     "nextCursor": null,
#     "reason": "live Argo logs do not provide stable historical cursor pagination"
#   },
#   "window": {
#     "mode": "tail",
#     "tailLines": 200,
#     "limitBytes": 262144,
#     "scope": "bounded-live-window"
#   },
#   "truncation": {
#     "bounded": true,
#     "tailLines": 200,
#     "maxTailLines": 2000,
#     "limitBytes": 262144,
#     "maxLimitBytes": 2097152,
#     "bytesTruncated": false
#   }
# }

# 错误路径示例：live Argo 日志不支持稳定历史 cursor，传 cursor 返回 400。
curl -i "$BASE/api/v1/workflows/<WORKFLOW_NAME>/logs?nodeId=<NODE_ID>&cursor=older" \
  -H "X-Databrew-Token: $TOKEN"

# 流式日志。canonical endpoint 是 /logs/stream；旧 /log/stream 仅作为兼容 alias。
curl -N "$BASE/api/v1/workflows/<WORKFLOW_NAME>/logs/stream?nodeId=<NODE_ID>&tailLines=200&limitBytes=262144" \
  -H "X-Databrew-Token: $TOKEN"

# SSE 事件示例:
# event: log
# data: {"podName":"wf-1-step-123","container":"main","line":"hello"}
#
# event: heartbeat
# data: {}
#
# event: end
# data: {"reason":"stream-complete"}

# 重试 / 重新提交 / 暂停 / 恢复 / 终止
curl -X POST "$BASE/api/v1/workflows/<WORKFLOW_NAME>/retry" \
  -H "X-Databrew-Token: $TOKEN"
curl -X POST "$BASE/api/v1/workflows/<WORKFLOW_NAME>/resubmit" \
  -H "X-Databrew-Token: $TOKEN"
# 返回示例:
# {
#   "message": "ok",
#   "workflowName": "asset-pipeline-a1b2c3-x7k9p",
#   "pipelineRunId": "61bd4770-1b5a-4f8c-97c2-d8ff88f4b733"
# }
# 重新提交会在 Argo 创建新的 workflow；若原 workflow 有 DataBrew pipeline run，
# 后端会同步创建新的 pipeline_runs 记录，避免新 workflow 详情出现 pipeline run not found。
curl -X POST "$BASE/api/v1/workflows/<WORKFLOW_NAME>/suspend" \
  -H "X-Databrew-Token: $TOKEN"
curl -X POST "$BASE/api/v1/workflows/<WORKFLOW_NAME>/resume" \
  -H "X-Databrew-Token: $TOKEN"
curl -X POST "$BASE/api/v1/workflows/<WORKFLOW_NAME>/terminate" \
  -H "X-Databrew-Token: $TOKEN"

# 删除 workflow
curl -X DELETE "$BASE/api/v1/workflows/<WORKFLOW_NAME>" \
  -H "X-Databrew-Token: $TOKEN"

# 错误路径示例：缺少 nodeId 返回 400
curl -i "$BASE/api/v1/workflows/<WORKFLOW_NAME>/logs" \
  -H "X-Databrew-Token: $TOKEN"
```

## Pipeline Batch M1

创建 batch 可锁定模板版本、执行目标，并可指定 pilot 试跑数量。`targetId`
会保存到 batch job 的 filter JSON，并在 materialize、deploy、rerun、continue-full
产生的每个子任务中透传给 pipeline run；省略时使用默认执行目标。`assetIds` 单批上限为 10,000；超过上限返回
`400 INVALID_ARGUMENT`。

```bash
curl -X POST "$BASE/api/v1/backfill" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "daily-batch",
    "templateId": "tpl-123",
    "templateVersion": 4,
    "targetId": "video-proc-dev",
    "pilotCount": 50,
    "assetIds": ["asset-1", "asset-2"],
    "configSelection": {
      "mode": "saved",
      "configId": "cfg-123",
      "version": 2,
      "fileName": "feature-flags.yaml",
      "mountPath": "/workspace/configs",
      "targetFilename": "feature-flags.yaml"
    }
  }'

curl -s "$BASE/api/v1/backfill/<BATCH_ID>/node-summary" \
  -H "X-Databrew-Token: $TOKEN"

# 响应包含 subtasks、nodes[].counts、dataCoverage；Pending 表示该节点尚无账本行。

curl -s "$BASE/api/v1/backfill/<BATCH_ID>/node-failures?pipelineNodeId=step-extract&page=1&pageSize=20" \
  -H "X-Databrew-Token: $TOKEN"

curl -X POST "$BASE/api/v1/backfill/<BATCH_ID>/rerun" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope":"node_failed","pipelineNodeId":"step-extract","templateVersion":4,"dryRun":true}'

# 响应会返回 status / matchedCount / retriedCount / skipped；partial_success 表示部分项因唯一冲突或运行态跳过。
curl -X POST "$BASE/api/v1/backfill/<BATCH_ID>/rerun" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope":"failed","templateVersion":4}'

batch job 创建时附带的 `targetId` 和 `configSelection` 会原样保存在 job filter JSON 中，并在
每个子任务真正 materialize / deploy 时透传给 runtime；这样 pilot、rerun、
continue-full 与后续子任务能保持同一执行目标、挂载文件和环境变量语义。

curl -X POST "$BASE/api/v1/backfill/<BATCH_ID>/continue-full" \
  -H "X-Databrew-Token: $TOKEN"

# 错误路径：node_failed 缺少 pipelineNodeId 返回 400。
curl -i -X POST "$BASE/api/v1/backfill/<BATCH_ID>/rerun" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope":"node_failed"}'
```

## Backfill result upload (CYB-2097)

Register algorithm JSON against an asset + report manifest. Requires `report_manifests`
and `algo_run_results` tables on dev (legacy/dev schema).

```bash
curl -X POST "$BASE/api/v1/backfill/results" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "assetId": "<ASSET_ID>",
    "reportId": "report.project@1.0.0-backfill-left-eye-only",
    "version": "1.0.0",
    "manifest": { "assetId": "<ASSET_ID>" },
    "result": { "crop": "left_eye_only", "status": "ok" }
  }'

# 错误路径：manifest assetId 与 body 不一致 → 400
curl -i -X POST "$BASE/api/v1/backfill/results" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "assetId": "<ASSET_ID>",
    "reportId": "report.project@1.0.0-backfill-left-eye-only",
    "version": "1.0.0",
    "manifest": { "assetId": "other-asset" },
    "result": {}
  }'
```

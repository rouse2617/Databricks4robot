# Backend

Go 后端服务，当前运行时以 `postgres` 为唯一存储后端，提供资产管理、算法生命周期、交付管理全套 API。

## 架构概览

```
HTTP Request
  → middleware (RequestID → RequestGuard → StaticTokenAuth)
    → Handler (参数校验、响应格式化)
      → Usecase (业务逻辑、状态机、乐观锁重试)
        → Repository Interface
          └── PostgreSQL 实现 (internal/postgres/)
```

运行时仅支持 `STORAGE_BACKEND=postgres`。

## 快速开始

### Local Development with Postgres (推荐)

```bash
# 1. 启动 Postgres (首次会自动执行 migrations)
cd deploy/local
docker-compose up -d postgres

# 2. 配置环境变量
cd backend
cp .env.example .env
# 默认已配置 STORAGE_BACKEND=postgres，无需修改

# 3. 安装依赖
make deps

# 4. 启动服务
make run
```

Postgres 容器首次启动时会自动执行 `backend/migrations/` 下 **按文件名排序**的 SQL（仅 **DDL / 增量迁移**，不含演示 INSERT）：
- `001_init.sql` — 建表、索引、约束
- `004` … `016_*.sql` — 后续演进迁移

可选 **本地演示数据**（原 `002_seed.sql`，几条 MCAP、资产 `aset0001` 等）不再随 initdb 执行；需要时在仓库根目录：

```bash
make local-dev-seed
```

等价于 `bash backend/scripts/apply_dev_seed.sh`（若已存在 `aset0001` 会自动跳过）。

**已有 volume、但未跑到后续迁移时**（例如缺少 `asset_events.aggregate_type`，写入资产会 500），在 `backend` 目录执行增量回放：

```bash
make migrate-local   # 默认 DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:5432/cyber_databrew_dev
```

等价于 `bash scripts/apply_pg_deltas.sh`（跳过 `001`/`013`；可选 demo seed 见 `apply_dev_seed.sh`；`013` 会删除遗留 `cf_*` 列，仅在确认无依赖后设置 `APPLY_CF_LEGACY_DROP=1` 再执行）。

数据持久化在 Docker named volume `pgdata` 中，重启不丢失。如需重置数据：

```bash
docker-compose down -v   # 删除 volume
docker-compose up -d postgres  # 重新初始化（仅 DDL）
make local-dev-seed       # 可选：插入与本仓库一致的演示 MCAP/资产行
```

服务默认监听 `:8080`，健康检查: `GET /healthz`

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `STORAGE_BACKEND` | 存储后端 | `postgres` |
| `PORT` | 监听端口 | `8080` |
| `DATABREW_TOKEN` | API 认证 token (Phase 0) | `dev-token` |
| `DB_HOST/PORT/USER/PASSWORD/NAME` | PostgreSQL 连接 (postgres 模式) | localhost:5432 |
| `LAKEHOUSE_REPORT_PATH` | 本地 Lakehouse MVP 报告路径 | `../deploy/local/iceberg/notebooks/lakehouse_report.json` |
| `LAKEHOUSE_BACKEND` | 湖仓查询后端 (`bigquery` / `none`) | `bigquery` |
| `LAKEHOUSE_BQ_PROJECT` | BigQuery 项目 ID | 空 |
| `LAKEHOUSE_BQ_DATASET` | BigQuery 数据集（对应 Iceberg Bronze） | `lakehouse_bronze` |
| `ELASTICSEARCH_URL` | Elasticsearch 连接地址 | `http://localhost:9200` |
| `ELASTICSEARCH_USERNAME` | ES HTTP Basic 用户名（可选；有密码且为空时用 `elastic`） | 空 |
| `ELASTICSEARCH_PASSWORD` | ES HTTP Basic 密码（可选；非空则启用 Basic 认证） | 空 |
| `OUTBOX_RELAY_ENABLED` | 将 `asset_events` 投递到 Pub/Sub（需 `PUBSUB_PROJECT` + `TOPIC_ASSET_EVENTS`） | `false` |
| `OUTBOX_RELAY_BATCH_SIZE` | 每轮最多拉取的 pending 事件数 | `200` |
| `OUTBOX_RELAY_INTERVAL_MS` | relay 轮询间隔（毫秒） | `500` |
| `OUTBOX_RELAY_SAFETY_LAG_SEC` | 只投递 `occurred_at` 早于该秒数的事件 | `2` |
| `OUTBOX_RELAY_MAX_RETRIES` | 单条事件超过后进入 `outbox_dlq` | `20` |
| `OUTBOX_ES_SUBSCRIBER_ENABLED` | 本进程订阅 Pub/Sub 并写 ES | `false` |
| `OUTBOX_ES_SUBSCRIPTION` | 订阅短名（与 `PUBSUB_PROJECT` 同项目） | 空 |

| `ADMIN_TOKEN` | Admin/internal 路由（硬删、reindex 等）；**生产必填**，未设置则不挂载这些路由 | 空 |

## API 端点

### 资产 (Assets)

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/assets` | 创建资产 |
| `GET` | `/api/v1/assets/:id` | 获取单个资产 |
| `PATCH` | `/api/v1/assets/:id` | 部分更新 |
| `DELETE` | `/api/v1/assets/:id` | 软删除 (status→archived) |
| `GET` | `/api/v1/assets/:id/deliveries` | 资产关联的交付列表 |
| `GET` | `/api/v1/assets/:id/events` | 查询资产事件时间线；可用 `event_type=algo_*` 取算法事件子集 |
| `GET` | `/api/v1/assets/:id/timeline` | `/events` 别名端点，返回同形时间线数据 |
| `POST` | `/api/v1/assets/:id/tags` | 新增/更新单个标签 |
| `DELETE` | `/api/v1/assets/:id/tags/:key` | 删除单个标签 |
| `GET` | `/api/v1/assets/:id/tags/history` | 查询标签变更历史 |
| `POST` | `/api/v1/assets/:id/actions` | 创建 seg 内时间段标注（mcap → seg → action 第三层） |
| `GET` | `/api/v1/assets/:id/actions` | 列出 seg 的 action；`?at=` 时间点查、`?from=&to=` 区间、`?label=` 过滤 |

### 算法生命周期 (Algo Lifecycle)

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/assets/:id/algo/:algo_key/start` | 启动算法 |
| `POST` | `/api/v1/assets/:id/algo/:algo_key/finish` | 完成算法 (ok/failed) |
| `POST` | `/api/v1/assets/:id/algo/:algo_key/reset` | 重置算法 |
| `GET` | `/api/v1/assets/:id/algo` | 查询当前算法投影状态 |

算法状态机: `blocked → pending → running → ok/failed → (reset) → pending`

依赖链: `action_annotation` 依赖 `hand_tracking + head_tracking + body_tracking` 全部完成后自动 unblock。

### 交付 (Deliveries)

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/deliveries` | 创建交付 (需要 `Idempotency-Key` header) |
| `GET` | `/api/v1/deliveries` | 交付列表 (分页 + 可选 status 过滤) |
| `GET` | `/api/v1/deliveries/:id` | 获取交付详情 |
| `GET` | `/api/v1/deliveries/:id/items` | 获取交付关联资产明细 |
| `GET` | `/api/v1/customers/:cid/deliveries` | 按客户查询交付 |

### 注册表 (Registry)

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/algo-registry` | 已注册算法列表 (从 algo_registry.yaml) |
| `GET` | `/api/v1/tag-registry` | 已注册标签列表 (从 tag_registry.yaml) |

### 搜索与索引 (Elasticsearch)

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/search/sync-status` | 查询索引同步模式（工作台提示 / 运维观测） |
| `GET` | `/api/v1/search/sync-progress` | PG/ES 文档数；Outbox `pending` / `processing` / safety lag 秒；两段水位：PG→MQ (`pg_max_event_seq` / `outbox_published_max_seq` / `seq_lag`) 与 MQ→ES (`es_applied_min_seq` / `consumer_lag`)，均 O(1)，建议作主告警指标 |

### 查询工作台 / Query API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/queries/validate` | 校验并规范化 Query IR，返回 `normalized_query / warnings / field_capabilities / debug_plan` |
| `POST` | `/api/v1/queries/run` | 执行 Query IR；当前主路径为 `ES recall + PG refine`，返回 `items / total / facets / warnings / debug_plan` |
| `GET` | `/api/v1/saved-queries` | 列出已保存的 Query IR |
| `POST` | `/api/v1/saved-queries` | 创建 saved query |
| `GET` | `/api/v1/saved-queries/:id` | 获取单个 saved query |
| `PATCH` | `/api/v1/saved-queries/:id` | 更新 saved query |
| `DELETE` | `/api/v1/saved-queries/:id` | 删除 saved query |

`/api/v1/queries/*` 是当前资产查询工作台的主入口：

- Query IR 协议已切到树形 `where`（`and / or / not / pred`）
- structured / keyword / semantic / similar 主查询统一经由 `validate -> run`
- `semantic` / `similar` 为 Elasticsearch 驱动的模式，不是向量检索

**索引写入**：业务写入与 `asset_events` 同事务；可选 `OUTBOX_RELAY` 投递到 Pub/Sub，`OUTBOX_ES_SUBSCRIBER` 消费并 `BulkIndex`（`version_type=external`）。开发环境未启用 subscriber 时由本地周期 reconciler 从 PG 全量刷新 ES。

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/admin/search/reindex` | 全量从 PG 重建 ES 文档（当前复用 `X-Databrew-Token` 认证）；支持 `dry_run`，返回 `indexed / deleted / failed / duration_ms` 汇总 |
| `POST` | `/api/v1/admin/search/reindex-jobs` | 创建异步重建任务（支持 `dry_run`）；立即返回 job |
| `GET` | `/api/v1/admin/search/reindex-jobs` | 最近重建任务列表（用于前端恢复状态） |
| `GET` | `/api/v1/admin/search/reindex-jobs/:id` | 查询单个任务进度与计数 |
| `POST` | `/api/v1/admin/search/reindex-jobs/:id/stop` | 请求停止任务（在安全 checkpoint 停止） |
| `POST` | `/api/v1/admin/search/reindex-jobs/:id/resume` | 从 `next_page` 断点续开 paused/failed 任务 |
| `GET` | `/api/v1/admin/search/outbox-stats` | 只读：`asset_events` 按 `publish_state` 计数 + `outbox_dlq` 表行数（排查 PG/ES 缺口） |
| `GET` | `/metrics` | Prometheus 指标（无认证，建议内网暴露） |

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/lakehouse/status` | BigQuery 查询层状态 |
| `GET` | `/api/v1/lakehouse/tables` | Iceberg MVP 表行数 |
| `GET` | `/api/v1/lakehouse/training-assets` | 查询训练快照包含的 assets |
| `GET` | `/api/v1/lakehouse/recompute-candidates` | 查询算法版本变更后的历史重算候选 |
| `GET` | `/api/v1/lakehouse/tag-timeline` | 查询 tag current-state 更新时间线 |
| `GET` | `/api/v1/lakehouse/quality-distribution` | 查询 segment 质量分布 |
| `GET` | `/api/v1/lakehouse/customer-replay` | 查询客户交付 replay manifest |
| `GET` | `/api/v1/lakehouse/sync-status` | 最近同步状态；统一返回 `{ available, source, data }`，支持 realtime 与 Dagster reconciliation 两种来源 |
| `GET` | `/api/v1/lakehouse/report` | 兼容接口：读取 Spark 生成的静态 MVP 报告 |

部分 Lakehouse 只读接口在依赖表未入湖时仍返回 **200** 与空 `items`，并在 JSON 中带 `note`（详见 `docs/review/api-guide.md` 湖仓小节）。

### 内部接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/internal/commit-segments` | 批量创建资产片段 |

### 认证

所有 `/api/v1/*` 端点需要 `X-Databrew-Token` 或 `Authorization: Bearer <token>` header。

### Query Workbench 查询协议

工作台主查询统一走：

- `POST /api/v1/queries/validate`
- `POST /api/v1/queries/run`

核心约定：

- `Query IR` 使用树形 `where`：`and / or / not / pred`
- `mode` 支持：`structured` / `keyword` / `semantic` / `similar`
- `run` 返回：`items / total / facets / warnings / debug_plan`

示例：

```json
{
  "schema_version": "v1",
  "mode": "structured",
  "scope": { "resource": "assets" },
  "where": {
    "and": [
      { "pred": { "field": "lifecycle_state", "op": "eq", "value": "ready" } },
      { "pred": { "field": "owner", "op": "eq", "value": "alice" } }
    ]
  },
  "sort": [{ "field": "updated_at", "direction": "desc" }],
  "page": { "page": 1, "page_size": 20 }
}
```

### 错误响应格式

所有错误返回统一格式:
```json
{
  "code": "ASSET_NOT_FOUND",
  "message": "asset not found",
  "request_id": "uuid",
  "details": {}
}
```

## 目录结构

```
backend/
├── cmd/server/          # 服务入口 main.go
├── config/              # 算法注册表 & Tag 注册表 (YAML)
│   ├── algo_registry.yaml
│   └── tag_registry.yaml
├── internal/
│   ├── outbox/          # Pub/Sub relay + ES subscriber（asset_events → 搜索索引）
│   ├── postgres/        # PostgreSQL 仓库实现
│   ├── config/          # 环境变量加载
│   ├── filter/          # 过滤条件解析 (field:op:value)
│   ├── handlers/        # HTTP Handler 层
│   │   ├── asset/       # 资产 + 算法生命周期 handler
│   │   ├── delivery/    # 交付 handler
│   │   ├── mcap/        # MCAP 文件 handler
│   │   ├── registry/    # 算法注册表 + 标签注册表 handler
│   │   ├── search/      # Elasticsearch 检索 handler
│   │   └── lakehouse/   # 湖仓查询 handler
│   ├── httpresp/        # 统一错误响应
│   ├── middleware/       # RequestID, RequestGuard, StaticTokenAuth, RateLimit, CircuitBreaker
│   ├── models/          # 领域模型 (Asset, McapFile, Delivery, AlgoEvent)
│   ├── elasticsearch/      # Elasticsearch Go 客户端 (Search/BulkIndex)
│   ├── audit/           # 审计日志 (audit.Log)
│   ├── repository/      # 仓库接口定义
│   └── usecase/asset/   # 业务逻辑 (Usecase + AlgoUsecase)
├── routes/              # 路由注册
├── scripts/             # 测试脚本 & 工具
│   ├── test_edge_cases.sh      # 极端 Case 测试 (44 cases)
│   ├── test_edge_cases_2.sh    # 极端 Case 第二波 (41 cases)
│   ├── bench/main.go           # 压测工具
│   └── megaTest/               # 千级测试用例生成器 (1000 cases)
└── Makefile
```

## 开发指南

### 日常开发

```bash
make fmt          # 格式化代码
make vet          # 静态检查
make test         # 运行全部单元测试
make build        # 编译
```

### Outbox / 搜索索引

- 业务写入与 `asset_events` 同事务；`internal/outbox` 中的 relay 将 pending 行发布到 `TOPIC_ASSET_EVENTS`（需 GCP 与开启 message ordering 的 topic）。
- 可选在同一进程启用 `OUTBOX_ES_SUBSCRIBER`，从订阅拉取消息并用 `searchindex.Builder` 写 ES（`external` version = `event_seq`）。
- 开发环境未启用 subscriber 时，`cmd/server` 在非 production 下会启动本地周期 reconciler 从 PG 刷新 ES。

```bash
# 运行 outbox 相关包（当前无 *_test.go）
go test ./internal/outbox/...
```

### 添加新算法

1. 在 `config/algo_registry.yaml` 中添加算法定义
2. 指定 `versions`、`depends_on`、`output.required_fields`
3. 新资产创建时会自动初始化算法状态 (无依赖→pending, 有依赖→blocked)
4. 不需要改代码，状态机和依赖链逻辑是通用的

### 添加新 Tag

1. 在 `config/tag_registry.yaml` 中添加 tag 定义
2. 支持 `enum` (限定值列表) 和 `string` (自由文本, 可选 `max_length`) 两种类型
3. 创建/更新资产时自动校验

### 添加新 API 端点

1. 在 `internal/handlers/` 下添加 handler 方法
2. 在 `internal/usecase/` 下添加业务逻辑
3. 如需新的仓库方法，先在 `internal/repository/` 接口中定义，再在 `internal/postgres/` 中实现
4. 在 `routes/routes.go` 中注册路由
5. 在 `cmd/server/main.go` 中接线依赖注入

### 中间件

| 中间件 | 文件 | 说明 |
|--------|------|------|
| `RequestID` | `middleware/request_id.go` | 生成/透传请求 ID |
| `RequestGuard` | `middleware/request_guard.go` | 拒绝超长 URI (>2048 字符) |
| `StaticTokenAuth` | `middleware/auth.go` | Phase 0 静态 token 认证 |

## 测试

### 单元测试

```bash
make test                                    # 全部
go test ./internal/outbox/... -v             # Outbox 包
go test ./internal/handlers/asset/... -v     # Handler 包
```

### 富数据 Mock（MCAP + 资产 + Tag + Algo + Eval 指标）

用于全面验证指标检索、资产列表、评测写入等路径（`lifecycle_state` 与 DB `chk_lifecycle_state` 一致）。

```bash
# 在仓库根目录，需已启动本地后端（如 make all-up）
make seed-rich                              # 默认 SEED_TOTAL=100，可调 SEED_TOTAL / SEED_WORKERS / SEED_BASE
bash backend/scripts/verify_rich_seed.sh    # healthz + metrics/registry + metrics:search + assets 抽样
```

### 极端 Case 测试

```bash
bash scripts/test_edge_cases.sh      # 第一波: 44 cases
bash scripts/test_edge_cases_2.sh    # 第二波: 41 cases
```

### 千级测试

```bash
go run ./scripts/megaTest -report test_report.md
# 生成 Markdown 测试报告，覆盖 20 个维度 1000 个用例
```

### 压测

```bash
# 全场景 200 请求，10 并发
go run ./scripts/bench -c 10 -n 200

# 只压创建接口
go run ./scripts/bench -c 50 -n 1000 -s create

# 只压读取
go run ./scripts/bench -c 20 -n 500 -s get

# 可选参数: -url, -token, -keep (保留测试数据)
```

## PostgreSQL 表结构

| 表名 | 主键 | 用途 |
|------|------|------|
| `assets` | asset_id (8-char TEXT) | 资产主表，当前态字段 + metadata/files 扩展区 |
| `mcap_files` | mcap_file_id (8-char TEXT) | MCAP 文件元数据 |
| `deliveries` | delivery_id (UUID) | 交付记录 |
| `delivery_items` | (delivery_id, asset_id) | 交付明细 (junction table) |
| `asset_algo_events` | event_id (UUID) | 算法状态变更事件 |
| `idempotency_keys` | (scope, idem_key) | 幂等键存储 |
| `audit_events` | event_id (UUID) | 审计日志（操作人、操作类型、受影响资源、请求摘要） |
| `saved_queries` | saved_query_id (UUID) | 持久化 Query IR 查询对象 |

索引: `idx_assets_mcap_file_id`, `idx_assets_status`, `idx_assets_segment_locator`, `idx_assets_created_at`, `idx_delivery_items_asset_id`, `idx_algo_events_asset_created`, `idx_algo_events_algo_status`, `idx_algo_events_run_id` (partial)

初始化: `docker-compose up -d postgres` (自动执行 migrations)

## Elasticsearch 检索层

Elasticsearch 当前承担资产查询工作台的召回与 facet 计数，配合 PostgreSQL refine：

- Elasticsearch：全文召回、`semantic/similar` 模式、facet 聚合
- PostgreSQL：精过滤、排序、最终分页、单资产读模型

### 启动 Elasticsearch

```bash
# 全栈启动（包含 Elasticsearch）
cd deploy/local
docker compose --profile full up -d

# 验证 Elasticsearch 健康
curl http://localhost:9200/_cluster/health
```

### 索引初始化

```bash
bash deploy/local/elasticsearch/init-index.sh
```

创建 `assets` 索引，mapping 包含 keyword/text/date/numeric 字段。

### Query Workbench 主链

- `POST /api/v1/queries/validate`
- `POST /api/v1/queries/run`

当前默认执行链：

1. Query IR validate / normalize
2. Elasticsearch recall（可用时）
3. PostgreSQL refine / sort / paginate
4. 返回 `facets / warnings / debug_plan`

### 数据同步

PostgreSQL `asset_events`（outbox）→ Pub/Sub（可选 relay）→ Elasticsearch subscriber（可选）；本地 / 兜底可用 `POST /api/v1/admin/search/reindex` 全量重建索引。

## 本地监控栈（Prometheus + Grafana）

### 启动监控栈

监控服务集成在全栈 compose 文件中：

```bash
cd deploy/local
docker compose --profile full up -d prometheus grafana postgres-exporter elasticsearch-exporter blackbox-exporter
```

或启动完整栈（含 backend、frontend 等）：

```bash
cd deploy/local
docker compose --profile full up -d
```

### 访问 Grafana

- 地址：http://localhost:3000
- 默认账号：`admin` / `admin`
- Prometheus 数据源已预配置，指向 `http://prometheus:9090`
- Dashboard 目录：`deploy/local/monitoring/grafana/dashboards/`

### 可用指标

**HTTP 请求指标**（由 `internal/middleware/metrics.go` 采集）：
- `http_request_duration_seconds` — 请求耗时 histogram（label: `method`, `path`, `status`）
- `http_requests_total` — 请求计数 counter

**Outbox**：`asset_events` 投递失败会累加 `retry_count`；超过 `OUTBOX_RELAY_MAX_RETRIES` 的行由 `OutboxDLQRepo.MoveToDLQ` 迁入 `outbox_dlq` 表（relay 周期触发）。

**Outbox 水位 Gauge**（由 `GET /api/v1/search/sync-progress` 触发刷新；Frontend Dashboard / Cloud Monitoring 周期性轮询即可）：
- `outbox_pg_max_event_seq` / `outbox_published_max_seq` / `outbox_seq_lag` — PG→MQ 段水位；`outbox_seq_lag` 是首选告警。
- `outbox_es_applied_min_seq` / `outbox_consumer_lag` — MQ→ES 段水位；冷启动期 `=0`，所有分片报齐后才推进。低吞吐场景下空闲分片（`updated_at` 早于 `OUTBOX_ES_CHECKPOINT_IDLE_AFTER_SEC`，默认 300s）会被视为已追平 `outbox_published_max_seq`，避免长尾分片虚拉高 lag。
- 推荐告警阈值与 PromQL 规则示例见 [`docs/review/outbox-watermark-alerts.md`](../docs/review/outbox-watermark-alerts.md)。

**依赖健康指标**（由 blackbox-exporter 采集）：
- BigQuery API 可达性
- Iceberg REST TCP 连通性
- Frontend HTTP 可达性
- PgBouncer TCP 连通性

### 验证 Backend Metrics 被 Scrape

```bash
# 1. 确认 backend /metrics 端点正常
curl http://localhost:8080/metrics | head -20

# 2. 在 Prometheus UI 查询 backend 指标
open http://localhost:9090
# 搜索: http_request_duration_seconds

# 3. 检查 Prometheus scrape 状态
open http://localhost:9090/targets
# 确认 backend job 状态为 UP
```

> **注意**：Prometheus 通过 `host.docker.internal:8080` 访问宿主机上运行的 backend（`make run` 模式）。
> 若 backend 以 docker 服务方式运行（`docker compose --profile full`），需将 prometheus.yml 中 backend target 改为 `backend:8080`。

---

## 审计日志

Phase 2 新增 `audit_events` 表，记录核心写操作的审计日志：

```sql
-- 查询最近审计事件
SELECT * FROM audit_events ORDER BY created_at DESC LIMIT 20;

-- 按操作类型查询
SELECT * FROM audit_events WHERE action = 'batch_tag' ORDER BY created_at DESC;
```

记录的操作：批量打标签、创建交付、资产删除、批量重试算法。

## 已知限制

- `mcap/:id/messages` 是占位接口，尚未实现消息解码
- Phase 0 使用静态 token 认证，Phase 0.5 将切换到 OIDC/JWT

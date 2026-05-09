# Backend

Go 后端服务，当前运行时以 `postgres` 为唯一存储后端，提供资产管理、算法生命周期、交付管理全套 API。`internal/bigtable/` 仅保留为历史参考与测试编译目标。

## 架构概览

```
HTTP Request
  → middleware (RequestID → RequestGuard → StaticTokenAuth)
    → Handler (参数校验、响应格式化)
      → Usecase (业务逻辑、状态机、乐观锁重试)
        → Repository Interface
          ├── Bigtable 实现 (internal/bigtable/)
          └── PostgreSQL 实现 (internal/postgres/)
```

运行时默认且仅支持 `STORAGE_BACKEND=postgres`。`bigtable` 模式已从运行期下线。

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
make migrate-local   # 默认 DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:5432/data4cyber
```

等价于 `bash scripts/apply_pg_deltas.sh`（跳过 `001`/`013`；可选 demo seed 见 `apply_dev_seed.sh`；`013` 会删除遗留 `cf_*` 列，仅在确认无依赖后设置 `APPLY_CF_LEGACY_DROP=1` 再执行）。

数据持久化在 Docker named volume `pgdata` 中，重启不丢失。如需重置数据：

```bash
docker-compose down -v   # 删除 volume
docker-compose up -d postgres  # 重新初始化（仅 DDL）
make local-dev-seed       # 可选：插入与本仓库一致的演示 MCAP/资产行
```

### Bigtable Notes (历史保留)

```bash
# 运行时已不支持 bigtable。
# 相关代码仅保留给历史测试与参考，不再作为本地开发路径。
```

服务默认监听 `:8080`，健康检查: `GET /healthz`

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `STORAGE_BACKEND` | 存储后端 | `postgres` |
| `BIGTABLE_PROJECT` | GCP 项目 ID | — |
| `BIGTABLE_INSTANCE` | Bigtable 实例名 | `poc-datainfra` |
| `PORT` | 监听端口 | `8080` |
| `GRACE_TOKEN` | API 认证 token (Phase 0) | `dev-token` |
| `DB_HOST/PORT/USER/PASSWORD/NAME` | PostgreSQL 连接 (postgres 模式) | localhost:5432 |
| `LAKEHOUSE_REPORT_PATH` | 本地 Lakehouse MVP 报告路径 | `../deploy/local/iceberg/notebooks/lakehouse_report.json` |
| `TRINO_ENABLED` | 是否启用 Trino 湖仓查询层 | `true` |
| `TRINO_URL` | Trino SQL driver URL | `http://data-platform@localhost:8082` |
| `TRINO_CATALOG` | Trino Iceberg catalog | `iceberg` |
| `TRINO_SCHEMA` | Trino Iceberg schema/namespace | `robot` |
| `ELASTICSEARCH_URL` | Elasticsearch 连接地址 | `http://localhost:9200` |
| `CDC_ENABLED` | 是否启用 CDC runtime | `false` |
| `CDC_SOURCE_DRIVER` | CDC source (`debezium-kafka` / `in-memory`) | 空 |
| `CDC_KAFKA_BROKERS` | Kafka broker 列表 | `localhost:19092` |
| `CDC_KAFKA_GROUP_ID` | Kafka consumer group | `cyber-databrew-cdc` |
| `CDC_TOPIC_ACTIONS` | actions 表 CDC topic | `actions` |
| `CDC_MAX_CONSECUTIVE_FAILURES` | 连续失败阈值（超过后 fail-fast） | `20` |
| `CDC_DLQ_ENABLED` | 是否启用失败记录 DLQ | `true` |
| `CDC_DLQ_DIR` | 本地 DLQ JSONL 目录 | `/tmp/cdc-dlq` |
| `CDC_ES_BATCH_SIZE` | ES 投影 bulk 分块大小 | `200` |
| `CDC_ES_RATE_LIMIT_PER_SEC` | ES 投影批次速率（0=不限速） | `0` |

| `ADMIN_TOKEN` | 预留配置；当前 `reindex` 先复用 `X-Grace-Token` | 空 |

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
| `GET` | `/api/v1/search/sync-status` | 查询索引同步状态（工作台提示 / 运维观测） |

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

**索引写入**：当前分支仅保留 CDC 路径，ES 文档由 CDC consumer 投影写入。

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/admin/search/reindex` | 全量从 PG 重建 ES 文档（当前复用 `X-Grace-Token` 认证）；支持 `dry_run`，返回 `indexed / deleted / failed / duration_ms` 汇总 |
| `GET` | `/metrics` | Prometheus 指标（无认证，建议内网暴露） |

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/lakehouse/status` | Trino 查询层状态 |
| `GET` | `/api/v1/lakehouse/tables` | Iceberg MVP 表行数 |
| `GET` | `/api/v1/lakehouse/training-assets` | 查询训练快照包含的 assets |
| `GET` | `/api/v1/lakehouse/recompute-candidates` | 查询算法版本变更后的历史重算候选 |
| `GET` | `/api/v1/lakehouse/tag-timeline` | 查询 tag current-state 更新时间线 |
| `GET` | `/api/v1/lakehouse/quality-distribution` | 查询 segment 质量分布 |
| `GET` | `/api/v1/lakehouse/customer-replay` | 查询客户交付 replay manifest |
| `GET` | `/api/v1/lakehouse/sync-status` | 最近同步状态；统一返回 `{ available, source, data }`，支持 realtime 与 Dagster reconciliation 两种来源 |
| `GET` | `/api/v1/lakehouse/report` | 兼容接口：读取 Spark 生成的静态 MVP 报告 |

### 内部接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/internal/commit-segments` | 批量创建资产片段 |

### 认证

所有 `/api/v1/*` 端点需要 `X-Grace-Token` 或 `Authorization: Bearer <token>` header。

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
│   ├── bigtable/        # Bigtable 仓库实现 + 测试
│   │   ├── client.go    # btTable 接口、连接、编码工具
│   │   ├── repos.go     # AssetRepo, McapFileRepo, DeliveryRepo, AlgoEventRepo, IdempotencyRepo
│   │   ├── repos_test.go # 单元测试 + 属性测试 (fakeTable)
│   │   └── e2e_test.go  # 端到端测试 (httptest + fakeTable)
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
│   ├── bootstrap_bigtable.sh   # Bigtable 表结构初始化
│   ├── test_bigtable_e2e.sh    # 集成测试 (真实 Bigtable)
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

### Bigtable 仓库开发

所有 Bigtable 仓库代码在 `internal/bigtable/` 下。测试使用 `fakeTable`/`fakeDataClient` 模拟，不依赖真实 Bigtable。

关键约定:
- 列族常量在 `client.go` 中定义，必须与 `scripts/bootstrap_bigtable.sh` 一致
- `btTable` 接口抽象了 Bigtable 表操作，生产用 `realTable`，测试用 `fakeTable`
- `fakeTable` 支持 `storeOnApply: true` 模式，可以做完整的 Insert→Read 往返测试
- 整数字段用 `PackInt64`/`UnpackInt64` 大端序编码（兼容旧数据的字符串格式）
- 乐观锁通过 `CheckAndMutateRow` + `ValueRangeFilter` 实现

```bash
# 运行 Bigtable 包测试
go test ./internal/bigtable/... -v

# 覆盖率
go test ./internal/bigtable/... -coverprofile=/tmp/bt.cov
go tool cover -func=/tmp/bt.cov
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
3. 如需新的仓库方法，先在 `internal/repository/` 接口中定义，再分别在 `bigtable/` 和 `postgres/` 中实现
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
go test ./internal/bigtable/... -v           # Bigtable 包
go test ./internal/handlers/asset/... -v     # Handler 包
```

### 集成测试 (真实 Bigtable)

需要先启动后端连接真实 Bigtable:

```bash
# 终端 1: 启动后端
make run

# 终端 2: 运行集成测试
bash scripts/test_bigtable_e2e.sh

# 跳过全表扫描测试 (表数据多时)
SKIP_LIST_SCAN=1 bash scripts/test_bigtable_e2e.sh
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

## Bigtable 表结构

| 表名 | 列族 | 用途 |
|------|------|------|
| `assets` | meta, algo, tag, files | 资产主表 |
| `mcap_files` | meta, process | MCAP 文件元数据 |
| `deliveries` | meta | 交付记录 |
| `asset_algo_events` | meta | 算法状态变更事件 |
| `idx_segments_by_file` | ref | 二级索引: mcap_file → assets |
| `idx_asset_deliveries` | ref | 二级索引: asset → deliveries |
| `idx_customer_deliveries` | ref | 二级索引: customer → deliveries |
| `idempotency_keys` | meta | 幂等键存储 |

初始化: `make bt-bootstrap` 或 `bash scripts/bootstrap_bigtable.sh`

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

当前仅保留 CDC 路径：PostgreSQL current-state tables → CDC consumer → Elasticsearch。
本地 / 兜底场景可使用 `POST /api/v1/admin/search/reindex` 全量重建索引。

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

**CDC 指标**（由 `internal/cdc/metrics.go` 采集，`CDC_ENABLED=true` 时生效）：
- `cdc_batch_processed_total` — 已处理 batch 数（label: `consumer_kind`）
- `cdc_consumer_lag_events` — 消费者 lag（label: `consumer_kind`）
- `cdc_decode_errors_total` — Debezium 解码错误计数
- `cdc_es_rebuild_duration_ms` — ES 重建耗时 histogram
- `cdc_kafka_retries_total` — Kafka poll 暂时性错误重试计数
- `cdc_unknown_topic_total` — 未注册 topic 被丢弃计数
- `cdc_handler_failures_total{stage=decode|route|commit}` — 各处理阶段失败计数
- `cdc_dlq_writes_total` / `cdc_dlq_write_failures_total` — DLQ 写入成功/失败
- `cdc_consecutive_failures` — 当前连续失败计数（用于熔断观测）

### CDC 故障隔离与回放

- 默认开启 `DLQ`（`CDC_DLQ_ENABLED=true`）：解码失败、路由失败、提交失败会写入 `${CDC_DLQ_DIR}/failed_records_YYYYMMDD.jsonl`。
- 毒消息被记录后会尝试提交 offset，避免单条坏消息长期阻塞消费。
- 当连续失败达到 `CDC_MAX_CONSECUTIVE_FAILURES` 时，runtime 会 fail-fast 退出，等待外部拉起并排障。

回放示例：

```bash
cd backend
go run ./cmd/cdc-dlq-replay --file /tmp/cdc-dlq/failed_records_20260507.jsonl --brokers localhost:19092
```

或使用脚本：

```bash
cd backend
./scripts/replay_cdc_dlq.sh /tmp/cdc-dlq/failed_records_20260507.jsonl
```

**依赖健康指标**（由 blackbox-exporter 采集）：
- Trino HTTP 可达性
- Iceberg REST TCP 连通性
- Frontend HTTP 可达性
- PgBouncer TCP 连通性

### 验证 Backend Metrics 被 Scrape

```bash
# 1. 确认 backend /metrics 端点正常
curl http://localhost:8080/metrics | head -20

# 2. 在 Prometheus UI 查询 backend 指标
open http://localhost:9090
# 搜索: http_request_duration_seconds 或 cdc_batch_processed_total

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

## 性能参考 (真实 Bigtable)

| 接口 | avg | p50 | p95 | QPS |
|------|-----|-----|-----|-----|
| POST /assets (创建) | 499ms | 519ms | 591ms | ~19 |
| GET /assets/:id (读取) | 234ms | 227ms | 271ms | ~42 |
| algo start→finish→reset | 1.0s | 900ms | 1.7s | ~3 ops/s |
| GET /events | 459ms | 443ms | 579ms | ~21 |

测试环境: macOS → GCP Bigtable (green-valley-442103/poc-datainfra), 10 并发

## 已知限制

- `ListWithFilters` 在 Bigtable 模式下使用全表扫描 + 内存过滤，数据量大时较慢
- `mcap/:id/messages` 是占位接口，尚未实现消息解码
- Phase 0 使用静态 token 认证，Phase 0.5 将切换到 OIDC/JWT

# Backend

Go 后端服务，支持 `bigtable` 和 `postgres` 双存储后端，提供资产管理、算法生命周期、交付管理全套 API。

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

存储后端通过 `STORAGE_BACKEND` 环境变量切换，两种模式功能完全对等。

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

Postgres 容器首次启动时会自动执行 `backend/migrations/` 下的 SQL 文件：
- `001_init.sql` — 创建全部 6 张表、索引、外键
- `002_seed.sql` — 插入测试数据

数据持久化在 Docker named volume `pgdata` 中，重启不丢失。如需重置数据：

```bash
docker-compose down -v   # 删除 volume
docker-compose up -d postgres  # 重新初始化
```

### Local Development with Bigtable

```bash
# 1. 安装依赖
make deps

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env: STORAGE_BACKEND=bigtable，填入 GCP 项目信息

# 3. 初始化 Bigtable 表结构（首次）
make bt-bootstrap

# 4. 启动服务
make run
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

## API 端点

### 资产 (Assets)

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/assets` | 创建资产 |
| `GET` | `/api/v1/assets` | 列表查询 (支持 filter/sort/分页) |
| `GET` | `/api/v1/assets/:id` | 获取单个资产 |
| `PATCH` | `/api/v1/assets/:id` | 部分更新 |
| `DELETE` | `/api/v1/assets/:id` | 软删除 (status→archived) |
| `GET` | `/api/v1/assets/:id/deliveries` | 资产关联的交付列表 |

### 算法生命周期 (Algo Lifecycle)

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/assets/:id/algo/:algo_key/start` | 启动算法 |
| `POST` | `/api/v1/assets/:id/algo/:algo_key/finish` | 完成算法 (ok/failed) |
| `POST` | `/api/v1/assets/:id/algo/:algo_key/reset` | 重置算法 |
| `GET` | `/api/v1/assets/:id/algo-events` | 查询算法事件 |

算法状态机: `blocked → pending → running → ok/failed → (reset) → pending`

依赖链: `action_annotation` 依赖 `hand_tracking + head_tracking + body_tracking` 全部完成后自动 unblock。

### 交付 (Deliveries)

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/deliveries` | 创建交付 (需要 `Idempotency-Key` header) |
| `GET` | `/api/v1/deliveries/:id` | 获取交付详情 |
| `GET` | `/api/v1/customers/:cid/deliveries` | 按客户查询交付 |

### Lakehouse / Trino

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/lakehouse/status` | Trino 查询层状态 |
| `GET` | `/api/v1/lakehouse/tables` | Iceberg MVP 表行数 |
| `GET` | `/api/v1/lakehouse/training-assets` | 查询训练快照包含的 assets |
| `GET` | `/api/v1/lakehouse/recompute-candidates` | 查询算法版本变更后的历史重算候选 |
| `GET` | `/api/v1/lakehouse/tag-timeline` | 查询 tag current-state 更新时间线 |
| `GET` | `/api/v1/lakehouse/quality-distribution` | 查询 segment 质量分布 |
| `GET` | `/api/v1/lakehouse/customer-replay` | 查询客户交付 replay manifest |
| `GET` | `/api/v1/lakehouse/report` | 兼容接口：读取 Spark 生成的静态 MVP 报告 |

### 内部接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/internal/commit-segments` | 批量创建资产片段 |

### 认证

所有 `/api/v1/*` 端点需要 `X-Grace-Token` 或 `Authorization: Bearer <token>` header。

### 过滤与排序

```
GET /api/v1/assets?filter=status:eq:approved&filter=owner:eq:alice&sort_by=-created_at&page=1&page_size=20
```

支持的操作符: `eq`, `ne`, `lt`, `gt`, `lte`, `gte`, `like`, `ilike`, `in`, `nin`

字段不是自由输入。`filter` 和 `sort_by` 只接受白名单字段:

- 标量字段: `asset_id`, `mcap_file_id`, `status`, `reviewer`, `owner`, `type`, `env`, `task`, `created_at`, `updated_at`, `start_timestamp_ns`, `end_timestamp_ns`, `duration_sec`, `delivery_count`, `last_delivered_to`, `last_delivered_at`, `version`
- 生命周期字段: `retention_tier`, `archive_after_days`, `delete_after_days`, `total_size_bytes`, `last_accessed_at`
- 动态前缀: `tag.<key>`, `algo.<key>`, `files.<key>`
- 虚拟字段: `algo_status` (扫描所有算法的 :status 值), `has:delivery` (是否有交付)
- 兼容别名: `tags.<key>`, `cf_tag.<key>`, `algo_results.<key>`, `cf_algo.<key>`, `cf_files.<key>`

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
│   │   └── mcap/        # MCAP 文件 handler
│   ├── httpresp/        # 统一错误响应
│   ├── middleware/       # RequestID, RequestGuard, StaticTokenAuth
│   ├── models/          # 领域模型 (Asset, McapFile, Delivery, AlgoEvent)
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
| `assets` | asset_id (UUID) | 资产主表，JSONB 列族: cf_meta, cf_algo, cf_tag, cf_files |
| `mcap_files` | mcap_file_id (UUID) | MCAP 文件元数据 |
| `deliveries` | delivery_id (UUID) | 交付记录 |
| `delivery_items` | (delivery_id, asset_id) | 交付明细 (junction table) |
| `asset_algo_events` | event_id (UUID) | 算法状态变更事件 |
| `idempotency_keys` | (scope, idem_key) | 幂等键存储 |

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

## 性能参考 (真实 Bigtable)

| 接口 | avg | p50 | p95 | QPS |
|------|-----|-----|-----|-----|
| POST /assets (创建) | 499ms | 519ms | 591ms | ~19 |
| GET /assets/:id (读取) | 234ms | 227ms | 271ms | ~42 |
| algo start→finish→reset | 1.0s | 900ms | 1.7s | ~3 ops/s |
| GET /algo-events | 459ms | 443ms | 579ms | ~21 |

测试环境: macOS → GCP Bigtable (green-valley-442103/poc-datainfra), 10 并发

## 已知限制

- `ListWithFilters` 在 Bigtable 模式下使用全表扫描 + 内存过滤，数据量大时较慢
- `mcap/:id/messages` 是占位接口，尚未实现消息解码
- Phase 0 使用静态 token 认证，Phase 0.5 将切换到 OIDC/JWT

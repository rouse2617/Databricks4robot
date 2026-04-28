# CLAUDE.md

Practical project guidance for coding agents working in `Databricks4robot`.

## What This Repo Is

`Databricks4robot` is a single-backend-process asset platform for MCAP-oriented metadata, delivery tracking, search, lakehouse analysis, and SDK/frontend integration.

- Backend: Go + Gin (`backend/`)
- SDK: Python + httpx + pydantic (`sdk/`)
- Frontend: React + TypeScript (`Frontend/`)
- Orchestration: Dagster (`dagster/`)
- Online store: PostgreSQL by default, Bigtable retained as an alternate backend
- Search: Elasticsearch (`backend/internal/elasticsearch`, `/api/v1/search/assets`)
- Lakehouse query layer: Trino over Iceberg (`backend/internal/trino`, `/api/v1/lakehouse/*`)

## Current Architecture (Source of Truth)

### Backend runtime

- **Single process only**: `backend/cmd/server/main.go`
- **Router entry**: `backend/routes/routes.go`
- **Storage backend**: PostgreSQL only. `STORAGE_BACKEND=postgres` (default).
  - `STORAGE_BACKEND=bigtable` is **DEPRECATED**: the server fails fast at startup. The `internal/bigtable` package is kept as historical reference and is scheduled for removal; do not add new features against it.

### Backend layering

- `handler -> usecase -> repository`
- Handlers: `backend/internal/handlers/{asset,mcap,delivery,registry,search,lakehouse}`
- Usecases: `backend/internal/usecase/`
- Repositories:
  - Postgres (active): `backend/internal/postgres/client.go`, `backend/internal/postgres/repos.go`
  - Bigtable (DEPRECATED, retained as reference): `backend/internal/bigtable/`
- Optional service clients:
  - Elasticsearch: `backend/internal/elasticsearch/client.go`
  - Trino: `backend/internal/trino/client.go`

### Data schema sources

- Bigtable/logical schema source: `docs/sql.md`
- OpenAPI source: `api/openapi.yaml`
- PG schema: `schemas/pg-phase0.sql`

### 文档索引

| 文档 | 路径 | 内容 |
|------|------|------|
| API 使用指南 | `docs/api-guide.md` | 全部端点 curl 示例、错误码、工作流 |
| OpenAPI 规范 | `api/openapi.yaml` | 机器可读 API 定义 |
| 后端 README | `backend/README.md` | 架构、开发指南、测试、性能 |
| 算法注册表 | `backend/config/algo_registry.yaml` | 算法定义、依赖、输出要求 |
| Tag 注册表 | `backend/config/tag_registry.yaml` | Tag 类型、枚举值 |
| 数据模型 | `docs/algo-lifecycle-and-data-model.md` | 算法生命周期设计 |
| Bigtable Schema | `docs/sql.md` | 表结构、行键、列族 |
| Lakehouse 查询边界 | `docs/lakehouse-query-and-tag-filtering.md` | Postgres / Trino / Iceberg 查询职责 |
| 后训练平台架构 | `docs/advanced-training-data-platform-architecture.md` | 长期架构演进 |

## Removed / Disabled Capabilities

These are intentionally removed and should not be reintroduced unless explicitly requested:

- `GCSRawBucket` capability
- `/api/v1/mcap/upload/init`
- `/api/v1/mcap/:id/download-url`
- Multi-process backend split (`asset-service`, `mcap-gateway`, `delivery-service`)
- OpenSearch runtime/code paths (Elasticsearch is the current search backend)
- Bigtable storage backend at runtime (`STORAGE_BACKEND=bigtable` fails fast; package retained as historical reference, do not extend)

## API Conventions (Must Keep)

- Auth: `X-Grace-Token` (temporary phase-0 auth)
- Request tracing: `X-Request-ID` middleware
- Error envelope: `code`, `message`, `request_id`, `details`
- Idempotency:
  - `POST /api/v1/deliveries` requires `Idempotency-Key`
- Pagination:
  - `page`, `page_size`, `next_token` style must stay consistent

## Development Rules

1. **Keep interfaces stable** between handlers/usecases/repos.
2. **Update OpenAPI when API behavior changes**.
3. **No dead endpoints**: if route is removed, remove handler/docs/sdk usage together.
4. **Respect source-of-truth schema** (`sql.md`) for table/key/CF naming.
5. **Prefer simple structure**:
   - keep storage package layout as `client.go + repos.go` unless complexity requires split.

## Verification Checklist

Run after backend changes:

```bash
cd backend
make fmt
make vet
go test ./...
```

Storage package coverage targets:

```bash
go test ./internal/bigtable -coverprofile=/tmp/bt.cov && go tool cover -func=/tmp/bt.cov
go test ./internal/postgres -coverprofile=/tmp/pg.cov && go tool cover -func=/tmp/pg.cov
```

## Local Commands

### Backend

```bash
cd backend
make deps
make run
make bt-bootstrap
```

### SDK

```bash
cd sdk
uv sync --dev
uv run pytest tests/unit/
```

### Frontend

```bash
cd Frontend
npm install
npm run dev
```

## Environment Variables (Backend)

From `backend/.env.example`:

- `ENV`, `PORT`, `STORAGE_BACKEND`
- `BIGTABLE_PROJECT`, `BIGTABLE_INSTANCE`
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `TRINO_ENABLED`, `TRINO_URL`, `TRINO_CATALOG`, `TRINO_SCHEMA`, `LAKEHOUSE_REPORT_PATH`
- `ELASTICSEARCH_URL`
- `GCS_PROJECT`, `GCS_DERIVED_BUCKET`
- `PUBSUB_PROJECT`, `TOPIC_MCAP_FINALIZED`, `TOPIC_ASSET_EVENTS`
- `GRACE_TOKEN`
- `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`
- `CB_ENABLED`, `CB_WINDOW_SEC`, `CB_THRESHOLD`, `CB_COOLDOWN_SEC`
- `LOG_LEVEL`, `LOG_FORMAT`, `LOG_FILE`
- `GOOGLE_APPLICATION_CREDENTIALS` (optional local)

## 功能开发后必须更新的文件清单

每次开发新功能或修改现有功能后，按以下 checklist 逐项检查并更新：

### 新增 API 端点

| 文件 | 说明 |
|------|------|
| `backend/internal/handlers/*/` | 新增 handler 方法 |
| `backend/internal/usecase/*/` | 新增业务逻辑 |
| `backend/internal/repository/*.go` | 接口定义（如需新仓库方法） |
| `backend/internal/bigtable/repos.go` | Bigtable 实现 |
| `backend/internal/postgres/repos.go` | PostgreSQL 实现 |
| `backend/internal/elasticsearch/client.go` | 搜索客户端变更（如影响 search API） |
| `backend/internal/trino/client.go` | 湖仓查询客户端变更（如影响 lakehouse API） |
| `backend/routes/routes.go` | 注册路由 |
| `backend/cmd/server/main.go` | 依赖注入接线 |
| `api/openapi.yaml` | OpenAPI 规范 |
| `docs/api-guide.md` | API 使用指南（curl 示例） |
| `backend/README.md` | API 端点表格 |

### 新增算法

| 文件 | 说明 |
|------|------|
| `backend/config/algo_registry.yaml` | 算法定义（versions, depends_on, output） |
| `docs/api-guide.md` | 可用算法表格 |

不需要改代码 — 状态机和依赖链逻辑是通用的。

### 新增 Tag

| 文件 | 说明 |
|------|------|
| `backend/config/tag_registry.yaml` | Tag 定义（type, values, max_length） |
| `docs/api-guide.md` | Tags 校验规则说明 |

### 新增 Bigtable 表或列族

> **DEPRECATED — do not add.** Bigtable is no longer a supported runtime backend.
> If new storage requirements arise, extend `internal/postgres` and the
> repository interfaces in `internal/repository`.

### 修改 Model 字段

| 文件 | 说明 |
|------|------|
| `backend/internal/models/*.go` | 模型定义 |
| `backend/internal/bigtable/repos.go` | `rowToAsset()` / `Set()` 等转换函数 |
| `backend/internal/postgres/repos.go` | SQL 查询 |
| `api/openapi.yaml` | Schema 定义 |
| `docs/api-guide.md` | 响应示例 |

### 修改中间件

| 文件 | 说明 |
|------|------|
| `backend/internal/middleware/*.go` | 中间件实现 |
| `backend/routes/routes.go` | 中间件注册顺序 |
| `backend/README.md` | 中间件表格 |

### 修改错误码

| 文件 | 说明 |
|------|------|
| `backend/internal/httpresp/response.go` | 错误响应函数 |
| `api/openapi.yaml` | 错误响应 schema |
| `docs/api-guide.md` | 错误码参考表 |

### 测试相关

| 场景 | 需要更新的测试文件 |
|------|---------------------|
| Bigtable 仓库变更 | `backend/internal/bigtable/repos_test.go` |
| Handler 变更 | `backend/internal/handlers/*/handler_test.go` |
| 路由变更 | `backend/routes/routes_test.go` |
| 端到端流程变更 | `backend/internal/bigtable/e2e_test.go` |
| 集成测试 | `backend/scripts/test_bigtable_e2e.sh` |

## Definition of Done (Backend Changes)

每次提交前必须满足：

- [ ] `go build ./...` 编译通过
- [ ] `go test ./...` 全部通过
- [ ] `go vet ./...` 无警告
- [ ] 如果改了 API → 更新 `api/openapi.yaml`
- [ ] 如果改了 API → 更新 `docs/api-guide.md`
- [ ] 如果改了架构/流程 → 更新 `backend/README.md`
- [ ] 如果改了约定/规则 → 更新 `CLAUDE.md`
- [ ] **Do not** add new code against `internal/bigtable` (deprecated)
- [ ] 如果新增了算法/Tag → 更新对应 YAML 注册表
- [ ] 新代码有对应的单元测试

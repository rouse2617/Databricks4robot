# CLAUDE.md

Practical project guidance for coding agents working in `data-platform`.

## What This Repo Is

`data-platform` is a single-backend-process asset platform for MCAP-oriented metadata, delivery tracking, and SDK/frontend integration.

- Backend: Go + Gin (`backend/`)
- SDK: Python + httpx + pydantic (`sdk/`)
- Frontend: React + TypeScript (`Frontend/`)
- Orchestration skeleton: Dagster (`dagster/`)

## Current Architecture (Source of Truth)

### Backend runtime

- **Single process only**: `backend/cmd/server/main.go`
- **Router entry**: `backend/routes/routes.go`
- **Storage backend switch**: `STORAGE_BACKEND=bigtable|postgres`

### Backend layering

- `handler -> usecase -> repository`
- Handlers: `backend/internal/handlers/{asset,mcap,delivery}`
- Usecases: `backend/internal/usecase/`
- Repositories:
  - Bigtable: `backend/internal/bigtable/client.go`, `backend/internal/bigtable/repos.go`
  - Postgres: `backend/internal/postgres/client.go`, `backend/internal/postgres/repos.go`

### Data schema sources

- Bigtable schema source: `schemas/sql.md` in the `data4cyber` repository
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

## Removed / Disabled Capabilities

These are intentionally removed and should not be reintroduced unless explicitly requested:

- `GCSRawBucket` capability
- `/api/v1/mcap/upload/init`
- `/api/v1/mcap/:id/download-url`
- Multi-process backend split (`asset-service`, `mcap-gateway`, `delivery-service`)

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
- `GCS_PROJECT`, `GCS_DERIVED_BUCKET`
- `PUBSUB_PROJECT`, `TOPIC_MCAP_FINALIZED`, `TOPIC_ASSET_EVENTS`
- `GRACE_TOKEN`
- `GOOGLE_APPLICATION_CREDENTIALS` (optional local)

## 功能开发后必须更新的文件清单

每次开发新功能或修改现有功能后，按以下 checklist 逐项检查并更新：

### 新增 API 端点

| 文件 | 说明 |
|------|------|
| `backend/internal/handlers/*/` | 新增 handler 方法 |
| `backend/internal/usecase/*/` | 新增业务逻辑 |
| `backend/internal/repository/repository.go` | 接口定义（如需新仓库方法） |
| `backend/internal/bigtable/repos.go` | Bigtable 实现 |
| `backend/internal/postgres/repos.go` | PostgreSQL 实现 |
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

| 文件 | 说明 |
|------|------|
| `backend/scripts/bootstrap_bigtable.sh` | 表/列族创建脚本 |
| `backend/internal/bigtable/client.go` | 表名常量 + 列族常量 |
| `backend/internal/bigtable/repos.go` | 仓库实现 |
| `backend/internal/bigtable/repos_test.go` | 单元测试（fakeTable） |
| `backend/README.md` | Bigtable 表结构表格 |

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
- [ ] 如果改了 Bigtable 表结构 → 更新 `scripts/bootstrap_bigtable.sh` + `client.go` 常量
- [ ] 如果新增了算法/Tag → 更新对应 YAML 注册表
- [ ] 新代码有对应的单元测试

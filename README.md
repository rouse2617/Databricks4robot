# data-platform

面向视频 / 多模态**资产元数据、算法状态、检索与交付**的单进程后端 + Web / SDK 工程骨架。

> **架构基线与评审文档**以 [`docs/review/README.md`](docs/review/README.md) 为准（当前 **1.0：PostgreSQL + Backend**；**Bigtable 已不作为运行时存储**，`STORAGE_BACKEND=bigtable` 会启动失败）。本页只负责仓库导航与本地启动。

## Runtime snapshot

| 组件 | 状态 |
|------|------|
| 存储 | **PostgreSQL**（默认 `STORAGE_BACKEND=postgres`） |
| 服务 | Go 单进程（`backend/cmd/server`） |
| 检索 | 可选 **Elasticsearch**（`ELASTICSEARCH_URL`，未配置则搜索接口不可用） |
| 分析 | 可选 **Trino / Iceberg**（lakehouse 路由与本地脚手架） |
| Bigtable | **保留代码与测试作历史参考**，不再支持生产运行时 |

## Repository layout

| 路径 | 说明 |
|------|------|
| `backend/` | Go API（assets / mcap / deliveries / algo / search / lakehouse） |
| `sdk/` | Python SDK（`grace_sdk`） |
| `Frontend/` | React 前端 |
| `dagster/` | Dagster 编排骨架 |
| `deploy/` | 本地与部署（Docker Compose、脚本） |
| `schemas/` | SQL / 阶段 schema |
| `api/openapi.yaml` | HTTP 契约（与实现一致的源） |
| `docs/review/` | **评审与设计主文档包**（整体方案、schema 速查、API 指南） |
| `docs/archive/` | 历史调研与旧版前端规格（仅供参考） |

各子模块细节见对应目录内的 README。

## Quick start

### 1) 本地依赖（最小：Postgres + 可选模拟器）

```bash
make dev-up
```

默认会拉起 **PostgreSQL**（及本地 Bigtable / Pub/Sub 模拟器；后者仅在不走 GCP 时的可选依赖）。数据库会执行 `backend/migrations` 初始化。

### 2) 后端配置与启动

```bash
cp backend/.env.example backend/.env
# 按需编辑 DB_*、GRACE_TOKEN、ELASTICSEARCH_URL 等
make backend-run
```

等价于 `cd backend && make run-server`。详见 [`backend/README.md`](backend/README.md)。

### 3) 全栈（Postgres + 后端 + 前端 + Iceberg + Trino + ES）

```bash
make all-up
```

- 前端: http://localhost:5173  
- 后端: http://localhost:8080  
- Trino: http://localhost:8082  
- Elasticsearch: http://localhost:9200  

停止：`make all-down`。

### 4) 仅 Iceberg / Trino 湖仓脚手架

```bash
make iceberg-up
```

Notebook 与示例脚本在 `deploy/local/iceberg/notebooks/`。大规模本机数据示例（可选）：

```bash
ROW_COUNT=100000 BATCH_ID=scale_100k make pg-generate-scale
make iceberg-mvp-host
make trino-smoke
```

### 5) 前端

```bash
make frontend-install
make frontend-dev
```

## Environment

- 后端：`backend/.env.example` → `backend/.env`
- 前端：`Frontend/.env.example`

## Commit 规范

- 提交信息遵循 [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)：`<type>[optional scope]: <description>`
- 推荐类型：`feat`、`fix`、`docs`、`refactor`、`test`、`chore`
- 例如：`feat(backend): add delivery retry endpoint`

首次拉取仓库后，建议执行一次以下命令启用本仓库的 commit 校验 hook：

```bash
git config core.hooksPath .githooks
chmod +x .githooks/commit-msg
```

启用后，不符合规范的 `git commit` 会被拦截并提示修正。

## 常用 Makefile 目标

| 目标 | 作用 |
|------|------|
| `make dev-up` / `dev-down` | 本地 `docker-compose`（含 Postgres） |
| `make all-up` / `all-down` | 全栈 compose |
| `make backend-run` | 启动 API |
| `make backend-test` | `go test ./...` |
| `make test` | 后端 + SDK 单测 |
| `make iceberg-up` / `iceberg-down` | 独立湖仓 compose |

## README 模板（子模块建议）

子目录 README 建议包含：What · How to Run · Config · API/Interfaces · Directory Structure · Dev Workflow · Known Limitations · Next Milestones。

## Current status（高层）

- [x] PostgreSQL 仓储与单进程路由（含算法 `start` / `finish` / `reset`、`asset_events` 写入）
- [x] 可选 Elasticsearch 资产搜索、可选 Trino lakehouse 查询
- [x] 前端资产发现工作台、`sdk` 单测骨架
- [ ] 设计文档中的 **Outbox Worker → ES/Iceberg** 异步投递（表与写侧已有，消费侧按 2.0 规划）
- [ ] 生产级身份认证（当前 Phase 0：`X-Grace-Token`）

历史 Bigtable 实现仍存在于 `backend/internal/bigtable/`（测试与参考），**新功能不要依赖其扩展**。

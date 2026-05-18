# cyber-databrew（data-platform）

面向视频 / 多模态**资产元数据、算法状态、检索与交付**的单进程后端 + Web / SDK 工程骨架；GitHub 主仓库为 [`CyberOrigin2077/cyber-databrew`](https://github.com/CyberOrigin2077/cyber-databrew)。

> **架构基线与评审文档**以 [`docs/review/README.md`](docs/review/README.md) 为准（当前 **2.0：PostgreSQL + Elasticsearch + BigQuery + Outbox CDC**）。本页只负责仓库导航与本地启动。

## Runtime snapshot

| 组件 | 状态 |
|------|------|
| 存储 | **PostgreSQL**（默认 `STORAGE_BACKEND=postgres`） |
| 服务 | Go 单进程（`backend/cmd/server`） |
| 检索 | **Elasticsearch**（Outbox relay/subscriber 驱动同步） |
| 湖仓 | **BigQuery + BigLake-managed Iceberg**（Cloud Run Job 入湖） |
| 异步同步 | **Outbox**：`asset_events` + relay/subscriber（Pub/Sub → ES） |
| 预览 | **mcap-preview** 服务（HEVC/H.264 转码 + fMP4 流式） |

## Repository layout

| 路径 | 说明 |
|------|------|
| `backend/` | Go API（assets / mcap / deliveries / algo / search / lakehouse / admin） |
| `services/mcap-preview/` | MCAP 预览服务（独立 Go 服务，HEVC/H.264 转码 + fMP4 流式） |
| `Frontend/` | React + TypeScript 前端（card view / facets / preview pane / dashboard） |
| `sdk/` | Python SDK（`asset_sdk`，httpx + pydantic，CRUD 已完成，OpenAPI 类型生成待迭代）；见 [`sdk/README.md`](sdk/README.md) |
| `deploy/local/` | Docker Compose：最小依赖、全栈、Iceberg；说明见 [`deploy/local/README.md`](deploy/local/README.md) |
| `deploy/k8s/` | Kubernetes 清单（backend / frontend / mcap-preview / monitoring / gateway）；说明见 [`deploy/k8s/README.md`](deploy/k8s/README.md) |
| `deploy/cloudrun/` | Cloud Run 部署脚本与配置 |
| `deploy/iac/terraform/` | Terraform IaC（GCP 数据基础设施、Gateway、服务身份） |
| `.tekton/` | Tekton CI/CD 流水线（构建、部署、推送） |
| `schemas/` | SQL / 阶段 schema |
| `api/openapi.yaml` | HTTP 契约（与实现一致的源） |
| `docs/review/` | **评审与设计主文档包**（整体方案、schema 速查、API 指南、路线图） |
| `docs/archive/` | 历史调研与旧版设计（仅供参考） |
| `scripts/` | 运维与冒烟测试脚本 |

各子模块细节见对应目录内的 README。

## Quick start

### 1) 本地依赖（最小：Postgres + 可选模拟器）

```bash
make dev-up
```

默认会拉起 **PostgreSQL**（及本地模拟器等可选依赖）。数据库会执行 `backend/migrations` 初始化。

### 2) 后端配置与启动

```bash
cp backend/.env.example backend/.env
# 按需编辑 DB_*、GRACE_TOKEN、ELASTICSEARCH_URL 等
make backend-run
```

等价于 `cd backend && make run-server`。详见 [`backend/README.md`](backend/README.md)。

### 3) 全栈（Postgres + 后端 + 前端 + ES + Iceberg）

```bash
make all-up
```

- 前端: http://localhost:5173
- 后端: http://localhost:8080
- Elasticsearch: http://localhost:9200

停止：`make all-down`。

### 4) 湖仓脚手架（Iceberg + MinIO）

```bash
make iceberg-up
```

Notebook 与示例脚本在 `deploy/local/iceberg/notebooks/`。大规模本机数据示例（可选）：

```bash
ROW_COUNT=100000 BATCH_ID=scale_100k make pg-generate-scale
make iceberg-mvp-host
```

### 5) 前端

```bash
make frontend-install
make frontend-dev
```

## Environment

- 后端：`backend/.env.example` → `backend/.env`
- 前端：`Frontend/.env.example`
- 湖仓：`LAKEHOUSE_BACKEND`（`bigquery` / `none`）、`LAKEHOUSE_BQ_PROJECT`、`LAKEHOUSE_BQ_DATASET`

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
| `make dev-up` / `dev-down` | 最小 compose：Postgres + PgBouncer + 模拟器 |
| `make all-up` / `all-down` | 全栈 compose（前后端、PG、ES、Iceberg、监控） |
| `make backend-run` | 启动 API |
| `make backend-test` | `go test ./...` |
| `make test` | 后端 + SDK 单测 |
| `make iceberg-up` / `iceberg-down` | 独立湖仓 compose |

## Current status（高层）

- [x] PostgreSQL 仓储与单进程路由（含算法 `start` / `finish` / `reset`、`asset_events` 写入）
- [x] Elasticsearch 资产搜索（Outbox relay/subscriber 驱动同步）
- [x] BigQuery 湖仓查询（PG→Bronze→Silver→Gold 增量入湖）
- [x] 前端资产发现工作台（card view / facets / preview pane / dashboard）
- [x] MCAP 预览服务（HEVC/H.264 转码 + fMP4 流式播放）
- [x] Outbox CDC 同步主线（`asset_events` → relay → Pub/Sub → ES）
- [ ] 生产级身份认证（当前 Phase 0：`X-Grace-Token`）

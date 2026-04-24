# data-platform

`data-platform` 是一个面向视频/多模态资产管理与交付的工程骨架，当前以 **Bigtable + GCS + Pub/Sub** 为核心。

## Repository Layout

- `backend/`：Go 单进程服务（统一暴露 assets / mcap / deliveries API）
- `sdk/`：Python SDK（`grace_sdk`）
- `Frontend/`：React 前端
- `dagster/`：Dagster 任务编排骨架
- `deploy/`：本地与部署脚本（emulator、k8s 等）
- `schemas/`：数据模型与 schema 文档
- `docs/`：架构、ADR、设计文档

当前 Bigtable 写入结构以 `data4cyber` 仓库内的 `schemas/sql.md` 为准，`backend/internal/bigtable/` 已对齐表名、列族与索引 rowkey 规则。

## Module READMEs

- Backend: `backend/README.md`
- SDK: `sdk/README.md`
- Frontend: `Frontend/README.md`
- Dagster: `dagster/README.md`

## Quick Start

### 1) 启动本地依赖（Emulator）

```bash
make dev-up
```

### 2) 启动后端服务

```bash
make backend-run
```

### 2.5) 初始化 Bigtable 表结构（首次）

```bash
make -C backend bt-bootstrap
```

### 3) 启动 Frontend

```bash
make frontend-install
make frontend-dev
```

## Environment

- Backend env: `backend/.env.example`
- Frontend env: `Frontend/.env.example`

## README Framework (建议统一模板)

后续每个子目录建议都使用下面这个 README 模板，方便团队协作：

1. **What**：模块职责（这个模块做什么）
2. **How to Run**：本地启动命令
3. **Config**：环境变量说明（最小可运行配置）
4. **API/Interfaces**：对外接口（HTTP/SDK/事件）
5. **Directory Structure**：核心目录解释
6. **Development Workflow**：测试、lint、构建命令
7. **Known Limitations**：当前未完成能力/约束
8. **Next Milestones**：下一阶段计划

## Current Status

- [x] 工程骨架已创建
- [x] Bigtable repository 层已落地
- [x] 服务入口与路由已接线
- [ ] 端到端 smoke test
- [ ] 更完整的业务逻辑与测试覆盖


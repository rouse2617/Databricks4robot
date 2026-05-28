# Proposal — CYB-1254

## Why

`databrew-pipeline/` 是一个独立的流水线设计器应用，基于 React Flow（@xyflow/react）构建，支持拖拽编排 pipeline 节点、组件注册管理、部署到 Argo Workflow。它之前被移出 PR scope，现已在 `feat/pipeline-integration` 分支上恢复。

当前它作为一个独立的 Vite 应用运行，有自己的路由、布局、样式系统。目标是把它合入主 `Frontend/`，成为主应用的一个页面 /pipeline，并补齐组件注册管理能力，避免用户只能使用少量硬编码组件。

## What Changes

### New Capabilities

1. 将 databrew-pipeline/ui 的 React Flow canvas 作为新页面集成到 Frontend
2. 保留拖拽编排、组件注册、部署面板三个核心功能
3. 复用主应用的 antd 组件和设计系统
4. 保持 React Flow canvas 自身的交互体验（拖拽、连线、配置面板）
5. 新增独立组件管理页 `/components`，支持搜索、列表、创建、编辑、删除自定义 pipeline 组件
6. 组件表单支持 name、image、type、command、args、env、description 字段，并通过后端持久化

### Modified Capabilities

- Pipeline component registry 从画布内的轻量注册面板扩展为全站侧边栏可达的后端持久化管理页面。
- 后端组件注册 API 对齐前端需要的字段和 `TASK.md` 指定路径，保留已有组件读取能力。

## Impact

- **Affected code**: `backend/internal/handlers/pipeline_component`, `backend/internal/usecase/pipeline_component`, `backend/internal/repository`, `backend/routes/routes.go`, `Frontend/src/api/pipelineComponentApi.ts`, `Frontend/src/pages/ComponentListPage.tsx`, `Frontend/src/components/AppLayout.tsx`, `Frontend/src/App.tsx`
- **New APIs**: compatibility CRUD surface under `GET/POST/PUT/DELETE /api/v1/pipeline-components`
- **Dependencies**: no new runtime dependencies expected

## Scope

- **In scope**: full component registry CRUD page, Ant Design table/modal form, sidebar route, backend route compatibility and validation fixes, API contract sync
- **Out of scope**: pipeline transpiler changes, Argo workflow runtime changes, database schema migration unless existing persistence cannot represent required fields

## Success Criteria

- [ ] 用户可以从侧边栏进入 `/components` 查看所有组件并按名称搜索
- [ ] 用户可以创建、编辑、删除包含 image/type/command/args/env/description 的自定义组件
- [ ] 列表展示 Name、Image、Type、Description、CreatedAt、Actions
- [ ] 后端提供并测试 component registry CRUD API
- [ ] API contract artifacts, local tests, frontend lint, and deploy verification are completed before commit

## Goals (SLO)

- **Latency**: component list p99 < 300ms for normal dev registry size
- **Concurrency**: CRUD endpoints remain safe under normal admin usage
- **Quality**: handler tests cover list/create/update/delete and at least one validation/error path

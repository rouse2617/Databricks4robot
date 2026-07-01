# Proposal — BatchJobDetailPage 加载性能优化 (CYB-2872)

> **Supersedes PR #271**（已关闭）。P0/P1a/P1c 方案原样保留；P1b 前端懒加载改用 callback ref
> 实现，原实现的 `useRef` + `useEffect` 组合存在竞态——组件首次渲染时 `job` 未加载完成会
> 提前 `return <Skeleton/>`，此时 ref 容器未被渲染、`ref.current` 为 `null`；`useEffect`
> 只执行这一次，`job` 到位后重新渲染出真实容器时不会重新触发，导致 `IntersectionObserver`
> 永远不会被创建。详见 `decisions.md` 与 `design.md`。

## Why

`/pipeline/batch/{id}` 首屏加载对包含 1000+ 子任务的批次尤为缓慢，瓶颈位于三处聚合接口，与一个未受视口守卫的列表组件：

1. `GET /backfill/{id}/node-summary` 每次调用都跑 `refreshBatchReadModel`（无节流）+ 复杂 `AggregateNodeStatusByBatchJobID` JOIN，**接口层无缓存**
2. `GET /runs/{id}/children` 的 `listDurableRunChildren` → `ListByParentRunID` 一次性拉所有 relation，**不分页**
3. `WorkflowExecutionList` mount 即 fetch，**未受视口守卫**

开发者代码注释（`BatchJobDetailPage.tsx:538`）已明确指出「这两个接口会聚合 1000+ run，是慢加载的主要瓶颈」，本次变更把瓶颈点一一解决。

## What Changes

### Modified Capabilities

- **`backfill.node-summary`**：接口新增 in-memory cache (TTL 5s) 减少 DB 压力；`refreshBatchReadModel` 改为 30s 节流（复用既有 `shouldSyncProgress` 钩子）。`AggregateNodeStatusByBatchJobID` SQL 改写，复用既有 `backfill_items_summary` 物化行（若已存在），否则保持现状
- **`runs.children`**：`listDurableRunChildren` 改为内存分页（基于已 ListByParentRunID 拉到的 relations 做切片，避免一次查全量）
- **`frontend.batch-detail`**：`WorkflowExecutionList` 改为 IntersectionObserver 触发的视口懒加载。**懒加载触发器用 callback ref 实现**（不用 `useRef` + `useEffect` 组合），确保不管容器 DOM 节点在哪一次渲染出现，挂载时都能正确创建 observer

### New Capabilities

无新增端点，无新 schema 列，无新增依赖。

## Impact

- **Affected code**
  - `backend/internal/handlers/backfill/handler.go` — `GetNodeSummary` 加缓存
  - `backend/internal/usecase/backfill/usecase.go` — `GetBatchNodeSummary` 调用 `shouldSyncProgress` 节流
  - `backend/internal/postgres/backfill_repo.go` — `AggregateNodeStatusByBatchJobID`（P1a 跳过，无改动）
  - `backend/internal/repository/pipeline_repository.go` — `RunRelationRepository` 接口新增 `ListByParentRunIDPage`
  - `backend/internal/postgres/pipeline_repo.go` — `ListByParentRunIDPage` 实现（`COUNT` + `LIMIT/OFFSET`）
  - `backend/internal/usecase/pipeline/usecase.go` — `listDurableRunChildren` 改调用分页方法
  - 所有实现 `RunRelationRepository` 的 mock（测试文件）— 同 commit 更新
  - `Frontend/src/pages/BatchJobDetailPage.tsx` — P1b callback ref 重写（见 `design.md`）
  - `Frontend/src/pages/WorkflowExecutionList.tsx` — 视口守卫（确认与 callback ref 触发时机兼容）

- **New APIs**: 无
- **New tables / migrations**: 无
- **Dependencies**: 无新增 npm / go.mod 依赖
- **API contract**: 无 OpenAPI 字段变更；`/backfill/{id}/node-summary` 与 `/runs/{id}/children` 既有响应 shape 保持不变，仅性能改善

## Scope

- **In scope**：
  - P0：node-summary in-memory cache + sync 节流
  - P1a：AggregateNodeStatusByBatchJobID SQL 改写（基于既有 schema）
  - P1b：WorkflowExecutionList 视口懒加载
  - P1c：listDurableRunChildren 内存分页
- **Out of scope**（后续 issue 跟踪）：
  - `stableSerialize` 改轻量字段对比
  - `listPipelineVersions` 触发条件收紧
  - `pipeline_run_asset_nodes` 索引优化
  - 前端虚拟列表（`WorkflowExecutionList` 大数据场景）

## Goals (SLO)

**Baseline 实测**（2026-07-01，dev batch `0878ceb8-bc84-40b1-b19e-b19f0bac8e17`，1000 子任务，`completed`）：

| 指标 | 现状 (dev 实测) | 目标 |
|---|---|---|
| `/backfill/{id}/node-summary` 响应时间（5 次采样，均无缓存） | 600–750ms，每次相近（**确认无缓存**）| < 50ms (缓存命中) |
| `/runs/{id}/children` 响应时间（5 次采样，1000 子）| 620–730ms，每次相近（**确认无分页收益**）| < 300ms (内存分页) |
| BatchJobDetailPage 页面 mount 时的 API 调用 | **5 个全部 eager**（`backfill/{id}`、`node-summary`、`children`、`pipelines/{id}/versions`、`runs?batchJobId=`）——Chrome MCP Network 面板实测确认，无一受视口守卫 | node-summary/children/listRuns 三个改为视口触发 |
| 子任务执行记录 `listRuns` 调用次数（mount 后不滚动） | **1**（eager，无视口守卫）| 0（视口外不触发）|
| BatchJobDetailPage LCP（本地 vite dev:remote，非生产 bundle）| 2938ms（其中约 1.9s 是 Vite 首次 JIT 编译 `BatchJobDetailPage.tsx` 的 dev-only 开销，非生产环境代表值）| 待生产 bundle 部署后重新测量 |

**测量方法**：`source scripts/dev-backend-env.sh` + curl 直连 backend 5 次取样（排除 dev-server 噪音）；
Chrome DevTools MCP 本地 `dev:remote`（`VITE_API_BASE_URL` 指向 dev 后端）+ `performance_start_trace`
+ `list_network_requests` 实测页面真实行为。

## Success Criteria

- [ ] Dev 上同 batch URL（`6838eec8-ab2c-496d-8681-8f44fab7b18b`）首屏 LCP 改善 ≥ 50%
- [ ] `/backfill/{id}/node-summary` 第二次请求（30s 内）响应 < 50ms
- [ ] `/runs/{id}/children` 响应时间随 relations 总数线性可控（不再随子任务数线性增长）
- [ ] `WorkflowExecutionList` 在视口外不发起 `listRuns` 调用（Network 面板验证）
- [ ] 既有后端测试套件全绿：`go test ./internal/handlers/backfill/... ./internal/usecase/backfill/... ./internal/usecase/pipeline/...`
- [ ] 既有前端测试套件全绿：`npm run test -- --run` in `Frontend/`
- [ ] `make fmt && make vet` (Tier S)
- [ ] Tier L：`go test ./...` + `npm run build`
- [ ] Deploy verification：`bash deploy/cloudrun/backend-dev.sh` + smoke `curl -b cookie /backfill/{id}/node-summary`；Chrome DevTools MCP 跑 perf trace

## Risks

- **缓存一致性**：5s 缓存窗口内状态字段（completed / failed）可能滞后；与既有 `GetBatchStatus` 5s 缓存策略一致，可接受。`refreshBatchReadModel` 30s 节流与 5s 缓存 TTL 错峰运行，sync 仍然每 30s 跑一次为后续请求预热
- **P1a SQL 改写**：若 schema 不存在 `backfill_items_summary` 物化行则跳过；保持原 JOIN 不变；需 `EXPLAIN ANALYZE` 验证
- **P1c 内存分页**：一次性 ListByParentRunID 在极大量 relation（>10k）时仍是内存开销；当前批次典型量级 1k~3k，可接受；后续需 push-down 到 SQL

## Decisions

详见 `decisions.md`。

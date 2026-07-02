# Proposal — CYB-1609

## Why
Workflow 节点日志已经支持 bounded tail，但用户排查失败节点时仍缺少稳定的分页、实时状态、下载和大日志保护；100w 行级别日志不能依赖一次性读取或前端全量渲染。

## What Changes

### New Capabilities
- Pipeline execution detail exposes log windows with explicit cursor/tail semantics so users can load bounded chunks instead of full logs.
- Workflow log viewer shows clear realtime follow state, load-more state, truncation hints, and download action.
- Failed DAG nodes can open the log viewer directly with the failed node selected.

### Modified Capabilities
- Existing bounded workflow log API keeps `tailLines`/`limitBytes` compatibility while returning pagination metadata that tells the client whether another request can fetch more.
- Existing SSE follow endpoint keeps structured events but the frontend treats connection lifecycle as a first-class UI state.

## Impact
- **Affected code**: `backend/internal/argo`, `backend/internal/handlers/workflow`, `backend/routes`, `Frontend/src/api/workflowApi.ts`, `Frontend/src/pages/useWorkflowDetail.ts`, `Frontend/src/pages/WorkflowDetailPage.tsx`, `Frontend/src/pages/WorkflowDagNode.tsx`
- **New APIs**: none expected; existing `GET /api/v1/workflows/{name}/logs` and `GET /api/v1/workflows/{name}/logs/stream` will be extended with cursor/window metadata if feasible.
- **Dependencies**: no new runtime dependency expected.

## Scope
- **In scope**: bounded log window metadata, cursor or documented fallback semantics, frontend load-more/follow/download UX, failed-node log entry, API docs and smoke coverage.
- **Out of scope**: persisted log warehouse, Cloud Logging export, object-storage log archive, terminal/exec commands, multi-cluster log aggregation.

## Success Criteria
- [ ] Default log view never requests or renders an unbounded full log payload.
- [ ] User can see whether logs are live, disconnected, loading, truncated, or unavailable.
- [ ] User can load additional bounded log windows when the backend exposes a cursor or receives a clear explanation when historical paging is unavailable.
- [ ] User can download the currently loaded log content without blocking the detail page.
- [ ] Failed node card opens the log drawer/panel with that node selected.
- [ ] API documentation describes cursor/tail behavior honestly, including Argo/Kubernetes limitations.

## Goals (SLO)
- **Latency**: default log request returns within 3s for normal dev workflows when Argo is responsive.
- **Quality**: targeted backend handler tests and frontend hook/component tests cover bounded defaults, cursor metadata, follow lifecycle, and empty/error states.

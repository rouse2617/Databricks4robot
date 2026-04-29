# Design — P1-7/P1-8: 事件 Schema CI 守门 + Worker 独立进程

## Source of Truth

- 事件契约与版本规则：#[[file:docs/review/data-platform-design.md]] §5.9
- Worker 部署形态演进：#[[file:docs/review/data-platform-design.md]] §6.2、风险 R2
- Worker 上线节奏：#[[file:docs/review/outbox-worker-design.md]] §17

## P1-7: Schema CI 守门

### 方案
- 事件 schema 文件以 JSON Schema 形式维护在 `schemas/events/<event_type>.v<n>.json`
- CI workflow：检测 `internal/` 下 `Append(` 调用的 `EventType` 是否有对应 schema 文件
- PR 改 producer 加字段不 bump version → CI 失败

### 变更
1. `schemas/events/` 目录：每种 event_type 一个 JSON Schema 文件
2. `.github/workflows/event-schema-check.yml`：CI job
3. `CLAUDE.md`：提交前 checklist 加"事件 schema 版本"

## P1-8: Worker 独立进程

### 前置条件
- P0-4 进程内 worker 稳定运行 90 天

### 方案
- 新增 `cmd/outbox-worker/main.go`：独立入口
- Dockerfile：独立镜像
- K8s Deployment：独立 pod
- Backend `OUTBOX_WORKER_ENABLED=false`
- 切换期间两边同时跑（SKIP LOCKED 保证不冲突），确认无丢事件后关 backend 侧

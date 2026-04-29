# Tasks — P1-7/P1-8: 事件 Schema CI 守门 + Worker 独立进程

> Source of truth: #[[file:docs/review/data-platform-design.md]] §5.9, #[[file:docs/review/outbox-worker-design.md]] §17

## Milestone 1: 事件 Schema CI 守门（P1-7）

- [ ] 1.1 新增 `schemas/events/` 目录，为每种 event_type 创建 JSON Schema 文件（v1）
- [ ] 1.2 CI workflow `.github/workflows/event-schema-check.yml`
- [ ] 1.3 模拟 1 次 minor bump 演练
- [ ] 1.4 模拟 1 次 major bump 演练
- [ ] 1.5 `CLAUDE.md` 提交前 checklist 加"事件 schema 版本"

## Milestone 2: Worker 独立进程（P1-8，blocked-by P0-4 稳定 90 天）

- [ ] 2.1 新增 `cmd/outbox-worker/main.go`
- [ ] 2.2 独立 Dockerfile
- [ ] 2.3 K8s Deployment manifest
- [ ] 2.4 切换验证：两边同时跑 → 确认无丢事件 → 关 backend 侧

# Requirements — P1-7/P1-8: 事件 Schema CI 守门 + Worker 独立进程

## Introduction

事件 payload schema 演进的 CI 守门机制，以及 Outbox Worker 从进程内 goroutine 抽离为独立 K8s Deployment。

Source of truth:
- #[[file:docs/review/data-platform-design.md]] §5.9（事件契约与 Schema 演进）
- #[[file:docs/review/outbox-worker-design.md]] §17（上线节奏）
- #[[file:docs/review/next-steps-tasks.md]] P1-7 / P1-8

## Requirements

### R1: 事件 Schema CI 守门（P1-7）
- PR 改 producer 必须改 schema 版本（`payload_schema_version`）
- CI workflow 上线
- 至少 1 次 major bump 演练通过
- 规则写进 `CLAUDE.md`

### R2: Worker 独立进程（P1-8，blocked-by P0-4 稳定 90 天）
- 独立 K8s Deployment
- Backend 关 `OUTBOX_WORKER_ENABLED`
- 切换无丢事件

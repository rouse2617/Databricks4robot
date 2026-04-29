# Requirements — P1-1/P1-2: ES 索引正式启用 + Outbox 监控

## Introduction

ES `assets` 索引正式启用并通过 PG↔ES 一致率闸口；Outbox 监控指标上线。

Source of truth:
- #[[file:docs/review/data-platform-design.md]] §5.6.1（ES 索引设计）、§8（监控告警）
- #[[file:docs/review/outbox-worker-design.md]] §7（ES Sink 与 Mapping）、§11（可观测性）
- #[[file:docs/review/next-steps-tasks.md]] P1-1 / P1-2

## Requirements

### R1: ES 索引正式启用（P1-1）
- mapping = outbox-worker-design.md §6 字段映射
- 接入 outbox 后 ES 文档数与 PG `assets` 一致率 > 99.9%
- 闸口对账脚本验证

### R2: Outbox 监控指标（P1-2）
- Prometheus metrics 按 outbox-worker-design.md §11.1 暴露
- `GET /metrics` 端点包含 `outbox_*` 指标
- 告警阈值文档化到 `backend/README.md`

# Design — P1-1/P1-2: ES 索引正式启用 + Outbox 监控

## Source of Truth

- ES mapping 与字段投影：#[[file:docs/review/outbox-worker-design.md]] §6（Doc 投影）、§7（ES Sink 与 Mapping）
- 监控指标定义：#[[file:docs/review/outbox-worker-design.md]] §11（可观测性）
- 告警分级：#[[file:docs/review/data-platform-design.md]] §8（监控告警）
- 闸口验收：#[[file:docs/review/data-platform-design.md]] §9.1.1（1.0 → 2.0 闸口）

## 变更范围

### P1-1: ES 索引
1. `deploy/local/elasticsearch/init-index.sh`：确认 mapping 与 outbox-worker-design.md §6 一致
2. 闸口对账脚本 `scripts/es-pg-audit.sh`：比对 PG `assets` 与 ES `assets` 的 (asset_id, version) 集合
3. diff 报告输出 missing / stale / extra

### P1-2: 监控
1. `internal/metrics/outbox.go`：按 §11.1 定义所有 Prometheus metrics
2. `GET /metrics` 端点已注册（Prometheus handler）
3. `backend/README.md`：告警阈值建议表

# Tasks — P1-1/P1-2: ES 索引正式启用 + Outbox 监控

> Source of truth: #[[file:docs/review/outbox-worker-design.md]] §6/§7/§11, #[[file:docs/review/data-platform-design.md]] §8/§9.1.1

## Milestone 1: ES 索引闸口（P1-1）

- [ ] 1.1 确认 `deploy/local/elasticsearch/init-index.sh` mapping 与 outbox-worker-design.md §6 字段映射一致
- [ ] 1.2 新增 `scripts/es-pg-audit.sh`：比对 PG `assets` 与 ES `assets` 的 (asset_id, version) 集合
- [ ] 1.3 diff 报告输出 missing / stale / extra，一致率 > 99.9%
- [ ] 1.4 CI nightly job 跑对账脚本

## Milestone 2: Outbox 监控指标（P1-2）

- [ ] 2.1 `internal/metrics/outbox.go`：定义 `outbox_worker_batch_size` histogram
- [ ] 2.2 `outbox_worker_batch_duration_ms` histogram
- [ ] 2.3 `outbox_worker_pending_total` gauge
- [ ] 2.4 `outbox_oldest_pending_age_seconds` gauge
- [ ] 2.5 `outbox_worker_retry_max` gauge
- [ ] 2.6 `outbox_sink_lag_seq` gauge
- [ ] 2.7 `outbox_es_bulk_failures_total` counter
- [ ] 2.8 `outbox_bulk_partial_failure_total` counter
- [ ] 2.9 `backend/README.md` 更新告警阈值建议表（§11.1 阈值列）

## Milestone 3: 测试（P1-T-1/P1-T-2）

- [ ] 3.1 PG↔ES 一致性对账脚本跑闸口验收
- [ ] 3.2 Outbox 性能压测：50 events/s × 60s → P99 ≤ 60s

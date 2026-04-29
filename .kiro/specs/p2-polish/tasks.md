# Tasks — P2: 2.0 打磨

> Source of truth: #[[file:docs/review/next-steps-tasks.md]] §3

## 后端

- [ ] P2-1: `POST /api/v1/assets:batch_get` 批量查询 API
- [ ] P2-2: ES nested query 同 tag 内多条件捆绑
- [ ] P2-3: ES `tagged_at` 字段，支持「最近一天打的 tag」过滤
- [ ] P2-4: `lifecycle_state` 加 PG CHECK 约束（状态机 6 个月无变更后）
- [ ] P2-5: 资产详情 fan-out 单 SQL 化（JSON_AGG）
- [ ] P2-6: DLQ 表替代 `retry_count` 阈值
- [ ] P2-7: 事件 retention 策略（`asset_events` 滚动归档）
- [ ] P2-9: 彻底移除 `cf_*` 遗留层（4 步）

## 前端

- [ ] P2-FE-1: Tag 高级筛选 UI（nested + source_type / confidence）
- [ ] P2-FE-2: 通用事件流总览页（跨 asset 事件流面板）

## 测试

- [ ] P2-T-1: Frontend coverage 提升到 ≥ 85%
- [ ] P2-T-2: Outbox chaos 测试（注入 PG / ES 故障）

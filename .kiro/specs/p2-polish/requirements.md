# Requirements — P2: 2.0 打磨 / 已识别但非阻塞

## Introduction

2.0 打磨项，非阻塞上线但已识别的改进。按触发条件启动。

Source of truth:
- #[[file:docs/review/data-platform-design.md]]（各章节散落的 TODO）
- #[[file:docs/review/next-steps-tasks.md]] §3（P2 表）

## Requirements

### R1: 批量查询 API（P2-1）
`POST /api/v1/assets:batch_get`（任意 ID 集合批量详情），触发条件：出现循环调 `GET /assets/{id}` ≥ 10 次的页面。

### R2: ES nested query 同 tag 内多条件（P2-2）
`tags.scene:source_type:algo` 语法，触发条件：算法治理需要"同条 tag 同时满足 key + source"。

### R3: ES tagged_at 字段（P2-3）
支持「最近一天打的 tag」过滤，mapping + outbox sink projector 都要改。

### R4: lifecycle_state CHECK 约束（P2-4）
状态机 6 个月无变更后启用 PG CHECK 约束。

### R5: 资产详情 fan-out 单 SQL 化（P2-5）
用 `JSON_AGG` 合并三表查询，仅当 PG 连接池压力上升时启用。

### R6: DLQ 表（P2-6）
替代 `retry_count` 阈值，当 retry 失败事件出现非偶发漏投时启用。

### R7: 事件 retention 策略（P2-7）
`asset_events` 滚动归档到 cold tier，表行数到达千万级 / 90 天后评估。

### R8: Tag 高级筛选 UI（P2-FE-1）
基于 `tags.<key>` nested + `source_type / confidence` 的复合 chip 编辑器。

### R9: 通用事件流总览页（P2-FE-2）
跨 asset 的事件流，按 `event_type` 聚合的实时面板。

### R10: 彻底移除 cf_* 遗留层（P2-9）
分 4 步：filter/API 不再接受别名 → seed/mock 改用 typed columns → 确认无读 → drop 列。

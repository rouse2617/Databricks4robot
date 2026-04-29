# Requirements — P1-FE: 2.0 前端功能

## Introduction

2.0 阶段前端功能：Lakehouse Dashboard、同步状态可视化、Admin Reindex 入口、保留期/过期视图。

Source of truth:
- #[[file:docs/review/data-platform-design.md]] §5.3.5（列表筛选）、§5.3.8（训练数据快照查询）
- #[[file:docs/review/next-steps-tasks.md]] P1-FE-1 ~ P1-FE-4
- #[[file:.kiro/steering/frontend-design-rules.md]]

## Requirements

### R1: Lakehouse Dashboard（P1-FE-1，blocked-by P1-6）
- `/api/v1/lakehouse/*` 端点中 ≥ 2 个有可视化卡片
- ES 故障时降级 PG fallback

### R2: 同步状态可视化（P1-FE-2）
- `/api/v1/lakehouse/sync-status` 接到 SettingsPage 或独立页
- 显示 `last_sync_at / postgres_count / iceberg_count / diff_pct`
- `status != ok` 时红色徽标

### R3: Admin Reindex 入口（P1-FE-3，blocked-by P0-5）
- SettingsPage 给 admin 角色一个"重建 ES 索引"按钮
- 带二次确认 + dry_run 选项 + 进度回显

### R4: 保留期/过期视图（P1-FE-4，blocked-by P0-FE-3）
- 列表加"30 天内将过期"快捷过滤
- 详情显示 retention badge（hot/warm/cold/archive 颜色区分）

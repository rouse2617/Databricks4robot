# Tasks — P1-FE: 2.0 前端功能

> Source of truth: #[[file:docs/review/data-platform-design.md]] §5.3, #[[file:docs/review/api-guide.md]]

## Milestone 1: Lakehouse Dashboard（P1-FE-1，blocked-by P1-6）

- [ ] 1.1 `AnalyticsPage` 接入 `/api/v1/lakehouse/training-assets` 真实数据
- [ ] 1.2 接入 `/api/v1/lakehouse/quality-distribution` 真实数据
- [ ] 1.3 至少 2 个可视化卡片
- [ ] 1.4 ES 故障时降级 PG fallback

## Milestone 2: 同步状态可视化（P1-FE-2）

- [ ] 2.1 接入 `/api/v1/lakehouse/sync-status`
- [ ] 2.2 状态卡片：last_sync_at、postgres_count、iceberg_count、diff_pct
- [ ] 2.3 `status != ok` 时红色徽标

## Milestone 3: Admin Reindex 入口（P1-FE-3）

- [ ] 3.1 SettingsPage admin 角色可见"重建 ES 索引"按钮
- [ ] 3.2 Modal：dry_run + rebuild_index + 二次确认
- [ ] 3.3 调用 `POST /api/v1/admin/search/reindex`，显示结果
- [ ] 3.4 失败列表可下载

## Milestone 4: 保留期/过期视图（P1-FE-4）

- [ ] 4.1 列表快捷 chip："30 天内将过期"
- [ ] 4.2 详情 retention badge（hot=绿/warm=黄/cold=蓝/archive=灰）
- [ ] 4.3 vitest 覆盖

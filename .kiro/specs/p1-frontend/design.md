# Design — P1-FE: 2.0 前端功能

## Source of Truth

- 读路径与场景：#[[file:docs/review/data-platform-design.md]] §5.3.4–§5.3.9
- 前端设计规范：#[[file:.kiro/steering/frontend-design-rules.md]]
- API 端点：#[[file:docs/review/api-guide.md]]

## 变更范围

### P1-FE-1: Lakehouse Dashboard
- `src/pages/AnalyticsPage.tsx`：接入 `/api/v1/lakehouse/training-assets`、`/api/v1/lakehouse/quality-distribution` 等
- 至少 2 个可视化卡片（表格 / 图表）

### P1-FE-2: 同步状态
- `src/pages/SettingsPage.tsx` 或独立页：接入 `/api/v1/lakehouse/sync-status`
- 状态卡片：last_sync_at、行数对比、diff_pct、红绿灯

### P1-FE-3: Admin Reindex
- `src/pages/SettingsPage.tsx`：admin 角色可见的"重建 ES 索引"按钮
- Modal：dry_run checkbox + rebuild_index checkbox + 二次确认
- 调用 `POST /api/v1/admin/search/reindex`，显示结果

### P1-FE-4: 保留期视图
- 列表快捷 chip："30 天内将过期"（`expire_at:between:now,now+30d`）
- 详情 retention badge：颜色区分 hot/warm/cold/archive

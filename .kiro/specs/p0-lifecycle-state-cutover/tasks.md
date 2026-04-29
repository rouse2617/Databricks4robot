# Tasks — P0-3: lifecycle_state 主消费切换

## Milestone 1: 后端 API + OpenAPI deprecated 标注

- [ ] 1.1 `api/openapi.yaml`：给 `status`、`type`、`duration_sec` 字段加 `deprecated: true`
- [ ] 1.2 `internal/searchindex/builder.go`：ES 文档投影 aggregation 字段从 `status` 切到 `lifecycle_state`
- [ ] 1.3 `internal/handlers/search/handler.go`：搜索 aggregation 从 `status_agg` 改为 `lifecycle_state_agg`
- [ ] 1.4 `docs/review/api-guide.md`：更新示例响应，标注 deprecated 字段

## Milestone 2: 前端 duration_sec → duration_ms（P0-FE-1）

- [ ] 2.1 `AssetsResultsPane` 排序选项 / 时长列改成读 `duration_ms`，UI 仍按秒展示（÷1000）
- [ ] 2.2 `OverviewTab` / `AssetPreviewHero` 详情卡改成读 `duration_ms`
- [ ] 2.3 全仓 `rg "duration_sec"` 仅剩 SDK / API 兼容层
- [ ] 2.4 vitest 通过

## Milestone 3: 前端 lifecycle_state 主路径（P0-FE-2）

- [ ] 3.1 列表过滤默认走 `lifecycle_state`
- [ ] 3.2 详情 badge 用 `lifecycle_state`，`status` 灰显 `(legacy)`
- [ ] 3.3 facet sidebar 切到 `lifecycle_state`
- [ ] 3.4 vitest + 手测无回归

## Milestone 4: 暴露新字段到 UI（P0-FE-3）

- [ ] 4.1 `asset_type` 进列表可选列 + 详情概览 + 过滤入口
- [ ] 4.2 `retention_tier` 进列表可选列 + 详情概览 + 过滤入口
- [ ] 4.3 `expire_at` 进列表可选列 + 详情概览（相对时间渲染）+ 过滤入口
- [ ] 4.4 `owner` 进列表可选列 + 过滤入口（已有详情展示）
- [ ] 4.5 vitest 覆盖 column toggle

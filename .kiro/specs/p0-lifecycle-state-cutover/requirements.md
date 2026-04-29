# Requirements — P0-3: lifecycle_state 主消费切换

## Introduction

将 `assets.lifecycle_state` 确立为资产状态的唯一权威字段，完成从旧 `status` 字段的切换。涉及后端 API 响应、OpenAPI deprecated 标注、ES facet 切换、前端列表/详情/facets 默认使用 `lifecycle_state`。

Source of truth:
- #[[file:docs/review/data-platform-design.md]] §5.8.1（API 版本治理与字段退役流程）
- #[[file:docs/review/next-steps-tasks.md]] P0-3 / P0-FE-1 / P0-FE-2 / P0-FE-3
- #[[file:docs/review/schema-reference.md]]（assets 表字段定义）

## Requirements

### R1: API deprecated 标注
- `api/openapi.yaml` 中 `status`、`type`、`duration_sec` 字段加 `deprecated: true`
- 设计文档 §5.8.1 退役对照表：`status` → `lifecycle_state`、`type` → `asset_type`、`duration_sec` → `duration_ms`

### R2: ES facet 切换
- `/api/v1/search/assets` aggregation 从 `status` 切到 `lifecycle_state`
- 移除 `status_agg`，新增 `lifecycle_state_agg`

### R3: 前端 duration_sec → duration_ms 全面切换（P0-FE-1）
- 全仓 `rg "duration_sec"` 仅剩 SDK / API 兼容层
- UI 仍按秒展示（除以 1000）

### R4: 前端 lifecycle_state 主路径切换（P0-FE-2）
- 列表过滤、详情 badge、facet 从 `status` 切到 `lifecycle_state`
- 老 `status` 仅在 `OverviewTab` 标灰显示并标注 `(legacy)`

### R5: 前端暴露 P0-1 新增字段到 UI（P0-FE-3）
- `asset_type / retention_tier / expire_at / owner` 进列表列 + 详情概览 + 过滤入口
- `expire_at` 用相对时间渲染（"7 天后到期"）

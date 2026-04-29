# Design — P0-3: lifecycle_state 主消费切换

## Overview

本 spec 完成 `lifecycle_state` 从双写到主消费的切换。后端在 P0-1 已完成字段提升，本阶段聚焦于：API 层 deprecated 标注、ES facet 切换、前端全面切到新字段。

## Source of Truth

- 字段退役流程：#[[file:docs/review/data-platform-design.md]] §5.8.1
- 字段定义：#[[file:docs/review/schema-reference.md]]
- ES 索引设计：#[[file:docs/review/outbox-worker-design.md]] §6（Doc 投影）
- 前端设计规范：#[[file:.kiro/steering/frontend-design-rules.md]]

## 变更范围

### 后端
1. `api/openapi.yaml`：`status` / `type` / `duration_sec` 字段加 `deprecated: true`
2. `internal/searchindex/builder.go`：ES 文档投影中 aggregation 字段从 `status` 切到 `lifecycle_state`
3. `internal/handlers/search/handler.go`：搜索 aggregation 从 `status_agg` 改为 `lifecycle_state_agg`
4. `docs/review/api-guide.md`：更新示例响应

### 前端
1. `src/lib/assets/assetsDiscoveryTypes.ts`：排序/过滤字段从 `duration_sec` → `duration_ms`，`status` → `lifecycle_state`
2. `src/components/assets/`：时长列、排序选项读 `duration_ms`
3. `src/pages/AssetDetailPage.tsx`：详情 badge 用 `lifecycle_state`，`status` 灰显 `(legacy)`
4. `src/pages/AssetsPage.tsx`：facet sidebar 切到 `lifecycle_state`
5. 列表可选列新增 `asset_type / retention_tier / expire_at / owner`
6. 过滤入口新增上述字段

## 验收标准
- 前端列表 / 详情 / facets 默认都用 `lifecycle_state`
- API 仍同时返回两个字段
- OpenAPI deprecated 标注到位
- vitest 通过

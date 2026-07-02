# CYB-2835 — 批量任务列表补充 owner / finished_at / duration 列

## 问题

批量任务列表（`/pipeline?tab=executions&executionView=batch`）目前只显示：
批次名称、模板、进度、状态、创建时间、操作。

单次执行列表有所属用户、耗时、完成时间等列，批量任务缺失这些，运营难以定位问题。

## 目标列（最终态）

| 列 | 来源 | 变更 |
|---|---|---|
| 批次名称 | `name` | 无变更 |
| 模板 | `templateId` | 无变更 |
| 进度 | `completedCount/totalCount` | 无变更 |
| 状态 | `status` | 无变更 |
| **所属用户** | `backfill_jobs.created_by` | 新增（需 DB 字段 + API 字段） |
| **完成时间** | `backfill_jobs.finished_at` | 新增（DB 已有，API 未暴露） |
| **耗时** | `finished_at - created_at`（前端计算） | 新增（依赖 finished_at） |
| 创建时间 | `createdAt` | 无变更 |

**不包含**：命名空间（批量任务横跨多个 namespace）、总成本（聚合 N 条 pipeline_runs 性能差，不纳入）、标签（backfill_jobs 表无 label 概念）。

## 方案

### 1. DB migration（`created_by`）
- `backfill_jobs` 表缺 `created_by TEXT` 字段
- 新增 migration：`ALTER TABLE backfill_jobs ADD COLUMN IF NOT EXISTS created_by TEXT`
- 写入时机：`CreateBackfillJob` handler 从 JWT claims 取 email 写入

### 2. Backend model & repo
- `BackfillJob` struct 新增 `CreatedBy string` + `FinishedAt *time.Time`（已在 DB，未暴露）
- `backfillJobSelectCols` 追加 `created_by, finished_at`
- `scanBackfillJob` 对应 scan 字段
- `CreateBackfillJob` 写入 `created_by`

### 3. API（无新路由，扩展现有 `GET /api/v1/batch-jobs` 响应字段）
- `ListBatchJobs` 响应体新增 `createdBy string` + `finishedAt *time.Time`
- 需同步更新 `api/openapi.yaml`

### 4. Frontend
- `BatchJob` interface 新增 `createdBy?: string` + `finishedAt?: string`
- `BatchJobList.tsx` 列定义新增：所属用户（`createdBy`）、完成时间（`finishedAt`）、耗时（前端 `dayjs(finishedAt).diff(createdAt, 'second')` 或 `created_at` 到 `updated_at`）

## 风险

- 历史 `backfill_jobs` 记录 `created_by` 为 NULL，列表里显示 `—` 即可
- `finished_at` 只在 status=completed/failed 时有值，其他状态显示 `—`

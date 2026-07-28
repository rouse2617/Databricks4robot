# Proposal — CYB-3709

## Why

单次执行列表(`WorkflowExecutionList`,非批量 scope)被批量子任务淹没。dev 实测:总数 **66,374**,其中绝大多数是批量子任务(首页 50 条里 44 条带 `batchJobId`),把真正的 **4,420** 条单次 run 埋没,列表基本没法用。

根源:[`WorkflowExecutionList.tsx`](../../../Frontend/src/pages/WorkflowExecutionList.tsx) 非批量分支的 `listRuns` 传 `excludeBatchParents: true`(cyb-3392b / PR #385 引入),只藏批量父聚合行、**保留所有批量子任务**。当初批量小时是刻意设计(子任务带徽章作为个体执行显示),但在 6 万+ 子任务规模下,单次 tab 失去意义。

## What Changes

### Modified Capabilities

- 单次执行列表改用 `excludeBatch: true`:只显真正的单次 run(`pr.batch_job_id IS NULL`,~4,420)。批量父+子都从单次 tab 隐藏;批量子任务仍在**批量 tab**(`batchJobId` 过滤分支)完整可见 —— 该分支不动。
- 更新过期的 `cyb-3392b` 注释。

后端无改动(`excludeBatch` / `excludeBatchParents` 两个过滤在 `pipeline_repo.go` 均已实现)。

## Impact

- **Affected code**: `Frontend/src/pages/WorkflowExecutionList.tsx`(非批量 `listRuns` 一处参数 + 注释)
- **Backend / API / migration**: 无
- **批量子任务访问**: 不受影响(批量 tab 依旧列出)

## Scope

- **In scope**: 单次执行列表参数 `excludeBatchParents` → `excludeBatch`
- **Out of scope**: 批量 tab(isBatchScope 分支);后端过滤逻辑;批量徽章组件(单次行不再有批量归属,徽章自然不渲染)

## Success Criteria

- [ ] 单次执行 tab 总数从 ~66k 降到 ~4.4k,无行带 batchJobId
- [ ] 批量 tab 仍列出该批量的子任务
- [ ] 前端 lint/build 通过;dev 部署后 Chrome DevTools MCP 验收

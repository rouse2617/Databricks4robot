# asset-management spec delta — CYB-3309

## MODIFIED — algo_status 结构化筛选走服务端精确过滤

### Given/When/Then(精确性与分页)

- **Given** 用户在 `/assets` 用结构化筛选 `algo_status:eq:<status>`(status ∈ ok/failed/running/pending/blocked)
- **When** 前端发起 `/api/v1/queries/run`,`where` 携带该谓词
- **Then** 列表、总数(`total`)、分页三者必须一致且来自后端精确结果;前端**不得**再对返回结果做 client 端"当前页过滤",也**不得**展示"仅当前页生效/总数仍为全量"类降级警告

### Given/When/Then(语义:EXISTS-any)

- **Given** 一个资产关联多个算法(各有 status)
- **When** 按 `algo_status:eq:failed` 筛选
- **Then** 命中条件为"该资产**至少有一个**算法处于 failed"(EXISTS-any 语义);此为既有服务端语义,前端展示需与之一致

### Given/When/Then(无数据空态)

- **Given** 当前数据中无资产满足 `algo_status:eq:<status>`
- **When** 执行筛选
- **Then** 显示「没有匹配的资产」空态,`total = 0`,分页归零;**不得**出现"total < 实际行数"或"total 停留在全量"的错位

## ADDED Requirements

### Requirement: WorkflowDetailPage 加载 run ledger 数据不重复请求

The system SHALL fetch a run's ledger sub-resources (events/asset-nodes/cost-summary/inputs/outputs/runtime) at most once per mount, manual refresh, or poll tick, rather than once per independent trigger path.

**Priority**: P2 (Nice-to-have)
**Rationale**: 挂载时两个独立的 effect 曾无条件各自触发同一份 ledger 数据加载,纯粹浪费后端/DB 负载,不影响最终渲染的数据正确性。

#### Scenario: workflowName 模式下挂载一次
- **Given** 用户打开 WorkflowDetailPage,`useWorkflowDetail` 以默认的 `workflowName` 模式查找 run
- **When** 组件挂载,`getWorkflow` 成功返回
- **Then** run ledger 的 6 个子资源请求(events/asset-nodes/cost-summary/inputs/outputs/runtime)各只发起 1 次

#### Scenario: getWorkflow 失败时 ledger 数据仍尝试加载且错误可见
- **Given** `getWorkflow` 请求失败(无论是 404 还是其他错误)
- **When** 系统仍尝试加载该 run 的 ledger 数据
- **Then** ledger 数据的加载仅发起 1 次,失败时的错误信息正确写入 `runEventState`/`assetNodeState`/`costSummaryState`/`runMetadataState`,而不是被静默丢弃

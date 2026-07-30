# Spec delta — CYB-4477

## ADDED Requirements

### Requirement: 选中行时 Batch Action Bar 浮现
`Frontend/src/pages/BatchJobList.tsx` 表格上方 SHALL 在 `selectedRowKeys.length >= 1` 时浮现批量操作浮栏；`< 1` 时不渲染。

**Priority**: P2 (Nice-to-have)
**Rationale**: 当前只有"批量导出资产 ID"按钮，未覆盖重试/暂停/取消批量操作；浮栏让批量操作可视化。

#### Scenario: 0 行选中时浮栏不渲染
- **Given** `selectedRowKeys.length === 0`
- **When** 表格渲染
- **Then** `<BatchActionBar>` 不在 DOM 中（不渲染）

#### Scenario: ≥ 1 行选中时浮栏浮现
- **Given** `selectedRowKeys.length === 3`
- **When** 表格渲染
- **Then** `<BatchActionBar>` 显示 "已选 3 个批次" 文字 + 至少 3 个批量操作按钮（重试 / 导出 / 暂停 / 取消）

#### Scenario: 浮栏按钮禁用条件
- **Given** 选中的批次全部 status='completed'
- **When** 浮栏渲染
- **Then** "批量重试"按钮 disabled（无失败项可重试）

---

### Requirement: 行 Hover 快捷工具栏
`Frontend/src/pages/BatchJobList.tsx` 和 `WorkflowExecutionList.tsx` 的表格行 SHALL 在 hover 时显示快捷操作按钮。

**Priority**: P2 (Nice-to-have)
**Rationale**: Fitts's Law 改善——用户不需要把眼睛/鼠标移到最右侧操作列才能操作。

#### Scenario: 默认状态无 hover 工具栏
- **Given** 表格正常渲染（鼠标未 hover）
- **When** 用户扫视列表
- **Then** hover 工具栏不可见（opacity: 0）

#### Scenario: Hover 行时工具栏浮现
- **Given** 鼠标 hover 到某行
- **When** 停留 ≥ 100ms
- **Then** 该行操作按钮 opacity 0 → 1，pointer-events 启用

#### Scenario: hover 离开时工具栏消失
- **Given** 鼠标在某行 hover
- **When** 鼠标移到另一行
- **Then** 原行工具栏消失，新行工具栏浮现

---

### Requirement: 合并"耗时"列
`Frontend/src/pages/BatchJobList.tsx` SHALL 把独立的"耗时"列合并为"耗时（含等待）"单列。

**Priority**: P2 (Nice-to-have)
**Rationale**: 当前"耗时"+"完成时间"两列语义重叠；合并减少列数，释放横向空间。

#### Scenario: 耗时列默认显示
- **Given** 一条 batch 的 runDuration = 600s, totalWait = 120s
- **When** 渲染"耗时"列
- **Then** 显示 `10m 0s`（实际执行时间）

#### Scenario: hover 显示含等待
- **Given** 同上
- **When** hover 该单元格
- **Then** Tooltip 显示 `10m 0s (等待 2m 0s)`

---

### Requirement: 失败数 / 失败率排序
`Frontend/src/pages/BatchJobList.tsx` 和 `WorkflowExecutionList.tsx` 的"进度"列 SHALL 支持按失败数排序。

**Priority**: P2 (Nice-to-have)
**Rationale**: 运维需要快速找到失败最多的批次；目前只能按时间排。

#### Scenario: 进度列排序
- **Given** 列表有多个 batch，failedCount 各异
- **When** 用户点击"进度"列表头排序箭头（descend）
- **Then** 行按 failedCount 降序排列（失败的在前）

#### Scenario: 所属用户列排序
- **Given** 列表有多个 createdBy 各异
- **When** 用户点击"所属用户"列表头排序箭头
- **Then** 行按 createdBy locale-aware 字典序排列
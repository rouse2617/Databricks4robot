# Proposal — CYB-4477

## Why

CYB-4470 完成 BatchJobList + WorkflowExecutionList 的 P0/P1 改进，但仍有 4 项 P2 优化（批量操作浮栏 / 行 Hover 工具栏 / 列合并 / 失败数排序）因 scope creep 拆分。本 issue 跟完。

## What Changes

### New Capabilities

- `Frontend/src/pages/BatchJobList.tsx`：Batch Action Bar（顶部批量操作浮栏）+ 行 Hover 工具栏 + 列合并 + 失败数排序
- `Frontend/src/pages/WorkflowExecutionList.tsx`：行 Hover 工具栏 + 失败数排序

### Modified Capabilities

- 无（columns 数据未变，只调整展示 + 排序）

## Impact

- **Affected code**:
  - `Frontend/src/pages/BatchJobList.tsx`
  - `Frontend/src/pages/WorkflowExecutionList.tsx`
  - 可能新增 `Frontend/src/components/pipeline/BatchActionBar.tsx`
  - 可能新增 `Frontend/src/components/common/HoverActionBar.tsx`
- **New APIs**: 无
- **Dependencies**: 无（antd 已支持 sorter / Table components）

## Scope

### In scope（P2 4 项）

1. **Batch Action Bar**：选中行时浮现浮栏，含 批量重试 / 批量导出 / 批量取消 / 批量暂停
2. **行 Hover 工具栏**：两页面通用，hover 行时操作按钮就地浮现
3. **合并"耗时"/"总时长"列**：BatchJobList 单列，主耗时 + Tooltip 显示含等待
4. **失败数 / 失败率排序**：BatchJobList 进度列 + 两页所属用户列加 sorter

### Out of scope

- 详情页内部次级列表
- 移动端适配
- 全局搜索 / 全局筛选

## Success Criteria

- [ ] BatchJobList 选中行时顶部浮栏出现，< 1 行隐藏
- [ ] 浮栏含 ≥ 3 个批量操作按钮（重试/导出/取消/暂停中的至少 3 个）
- [ ] 两页面 hover 行时操作按钮就地浮现（不用移到操作列）
- [ ] BatchJobList "耗时"列合并为单列，主耗时默认 + Tooltip 含等待
- [ ] BatchJobList 进度列可按失败数排序
- [ ] 两页面所属用户列可按字母排序
- [ ] vitest 全部通过（不动业务逻辑，断言小调整）
- [ ] tsc -b exit 0
- [ ] Chrome DevTools MCP 截图 4 项验收点

## Goals (SLO)

- **Latency**: 浮栏 show/hide ≤ 50ms（无网络往返）
- **Quality**: 现有测试通过，新增 component 覆盖率 ≥ 70%

> **变更日志**：续作 CYB-4470 P0/P1；本 change 仅 P2。
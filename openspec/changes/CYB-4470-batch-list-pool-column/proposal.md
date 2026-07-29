# Proposal — CYB-4470

## Why

`/pipeline/batch`（批量任务列表，BatchJobList）与 `/pipeline/executions` + `/runs`（单次执行列表，WorkflowExecutionList）经 UI/UX/Design/Psychology 四维度审视存在多处痛点：**图标噪音 / Tag 规范 / 视觉焦虑 / 数据对齐 / 信息冗余**。两条界面共享同一套设计语言，统一打包进本 change，避免重复走 OpenSpec + review 流程。

## What Changes

### New Capabilities

- `Frontend/src/pages/BatchJobList.tsx`：新增"资源池"列（filterable + status-colored）
- `Frontend/src/components/pipeline/BatchProgressCell.tsx`：失败进度条降饱和
- `Frontend/src/pages/WorkflowExecutionList.tsx`：复制图标 Hover 显现、资产 ID 去前缀、时间列降行高、Tag 统一、数值右对齐、成本格式化、操作列清理、对比按钮反馈

### Modified Capabilities

- `Frontend/src/lib/batchJobs.ts`：新增 `STATUS_TAG_PALETTE` 与 `resolveStatusTagColor`
- `Frontend/src/components/pipeline/BatchProgressCell.tsx`：失败色 token 切换

## Impact

- **Affected code**:
  - `Frontend/src/pages/BatchJobList.tsx`
  - `Frontend/src/components/pipeline/BatchProgressCell.tsx`
  - `Frontend/src/pages/WorkflowExecutionList.tsx`
  - `Frontend/src/lib/batchJobs.ts`
  - 对应两个 `.test.tsx`（columns 改动会触发断言更新）
- **New APIs**: 无
- **Dependencies**: 无（antd 已有 Tag、Tooltip、Dropdown；dayjs 已有 relativeTime）

## Scope

### In scope（按文件分）

**A. BatchJobList**（5 项）
1. A1. 资源池列（filter + status-color）
2. A2. P0 截断文本 Tooltip
3. A3. P0 列宽重平衡 + 空态弱化
4. A4. P1 Status Tag 配色规范化
5. A5. P1 Progress Bar 失败降饱和

**B. WorkflowExecutionList**（8 项）
6. B1. P0 复制图标 Hover 显现
7. B2. P0 资产 ID 去 `asset_id=` 前缀
8. B3. P1 Tag 样式统一（与 A4 联动，共享 token）
9. B4. P1 时间列降行高
10. B5. P2 数值字段右对齐
11. B6. P2 微小成本格式化
12. B7. P2 操作列层级清理
13. B8. P2 "对比选中"按钮反馈

### Out of scope（留后续 change）

- 行 Hover 快捷工具栏
- Batch Action Bar
- 合并耗时/总时长列
- 失败数排序

## Success Criteria

- [ ] BatchJobList 表格新增"资源池"列，下拉过滤可用
- [ ] 资源池 Tag 按 status 着色（available 绿 / unavailable 红 / disabled 灰）
- [ ] WorkflowExecutionList 复制图标 hover 才出现
- [ ] 资产 ID 单元格无 `asset_id=` 前缀
- [ ] 时间列单行 + hover 弹完整时间
- [ ] 数值字段右对齐
- [ ] 失败批次进度条不再饱和红
- [ ] Status Tag 配色统一（两个页面共享 token）
- [ ] `cd Frontend && npm test -- BatchJobList WorkflowExecutionList` 全通过
- [ ] Chrome DevTools MCP 实测 8 项验收点

## Goals (SLO)

- **Latency**: 资源池过滤 onChange 渲染 ≤ 100ms；批量对比按钮反馈实时（无网络往返）
- **Quality**: 现有测试通过，新增 utility 覆盖率 ≥ 80%

> **变更日志**：原独立 issue `CYB-4471`（WorkflowExecutionList 单独提案）已 Cancel 并入本 issue。

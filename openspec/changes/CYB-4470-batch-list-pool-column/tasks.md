# Tasks — CYB-4470

## OpenSpec & 流程

- [x] Linear issue `CYB-4470` 创建
- [x] Linear issue `CYB-4471`（WorkflowExecutionList）Cancel 并入
- [x] OpenSpec proposal.md / tasks.md / design-ui.md 写完（含两个 spec delta）
- [ ] 用户确认「OpenSpec OK，继续」
- [ ] 分支：`git fetch origin dev && git checkout -b feat/CYB-4470-batch-list-pool-column origin/dev`

---

## A. BatchJobList

- [ ] `Frontend/src/lib/batchJobs.ts` 加 `STATUS_TAG_PALETTE`（success/error/warning/paused/neutral 五档统一 token）
- [ ] `Frontend/src/pages/BatchJobList.tsx` columns 新增"资源池"列：
  - [ ] 渲染 `targetMap.get(batchTargetId(record.filterJson))?.name`
  - [ ] 过滤：复用 `templateFilter` 模式，options 从数据中出现的 target 提取
  - [ ] 着色：按 `target.status`（available → success, unavailable → error, enabled=false → default）
  - [ ] Tooltip 显示 `cluster={target.cluster} · ns={target.namespace}`
  - [ ] 空态 `N/A`
- [ ] `Frontend/src/pages/BatchJobList.tsx` 列宽重平衡：
  - [ ] "耗时" 100 → 80
  - [ ] "所属用户" 160 → 140
  - [ ] "批次名称" 320 → 340（让位）
- [ ] `Frontend/src/pages/BatchJobList.tsx` 空态统一 `<span style="color:#bfbfbf;font-style:italic">—</span>`
- [ ] `Frontend/src/pages/BatchJobList.tsx` 补"所属用户"列 Tooltip
- [ ] `Frontend/src/components/pipeline/BatchProgressCell.tsx` 失败段降饱和：
  - [ ] `derived='failure'` → 暗红 `#cf1322`
  - [ ] `derived='partial_failure'` → 保持橙色 `#fa8c16`
  - [ ] `derived='completed'` → 保持绿色
  - [ ] `derived='running'` → 保持蓝色（动态）

---

## B. WorkflowExecutionList

- [ ] `Frontend/src/pages/WorkflowExecutionList.tsx` 复制图标 Hover 显现：
  - [ ] `<CopyOutlined />` 默认 `opacity: 0`
  - [ ] hover 父单元格时 `opacity: 1`
  - [ ] 复用现有 `navigator.clipboard.writeText(value)` + `message.success("已复制")` 流程
- [ ] `Frontend/src/pages/WorkflowExecutionList.tsx` 资产 ID 去前缀：
  - [ ] line 396 `labels.asset_id = assetId` 改为 `labels.asset_id = assetId.replace(/^asset_id=/, "")`（或在渲染时 strip）
  - [ ] 列宽可瘦身约 60px
- [ ] `Frontend/src/pages/WorkflowExecutionList.tsx` 时间列降行高（line 188-197）：
  - [ ] `renderTimestamp` 简化为单行 `parsed.fromNow()`
  - [ ] Tooltip 内显示完整 `YYYY-MM-DD HH:mm:ss`
  - [ ] 删除 line 192-194 的绝对时间 `<div>`
- [ ] `Frontend/src/pages/WorkflowExecutionList.tsx` Tag 样式统一（line 355/1090/1316/1331/1373/1389 等）：
  - [ ] 共用 BatchJobList 的 `STATUS_TAG_PALETTE`（提到 `lib/designTokens.ts`）
  - [ ] 版本/系统 → 蓝色浅底
  - [ ] 资源/失败 → 橙/红浅底
  - [ ] 辅助标记 → 灰色浅底
- [ ] `Frontend/src/pages/WorkflowExecutionList.tsx` 数值右对齐：
  - [ ] "耗时" 列 `align: 'right'`
  - [ ] "视频时长" 列 `align: 'right'`
  - [ ] "总成本" 列 `align: 'right'`
- [ ] `Frontend/src/pages/WorkflowExecutionList.tsx` 微小成本格式化（line 291）：
  - [ ] `cost === 0` → `$0.00`
  - [ ] `cost > 0 && cost < 0.001` → `<$0.001`
  - [ ] `cost >= 0.001 && cost < 0.01` → 4 位精度
  - [ ] `cost >= 0.01` → 2 位精度（保持现状）
- [ ] `Frontend/src/pages/WorkflowExecutionList.tsx` 操作列清理：
  - [ ] "批次名称"加 click handler 跳详情
  - [ ] "模板" / "查看" 按钮从操作列移除
  - [ ] 仅保留 `<Dropdown>` "操作 ⁝" 收纳次要动作
- [ ] `Frontend/src/pages/WorkflowExecutionList.tsx` "对比选中"按钮反馈：
  - [ ] `selectedRowKeys.length < 2` → 按钮 disabled，hover 弹 Tooltip "请至少勾选 2 条"
  - [ ] `selectedRowKeys.length >= 2` → 按钮高亮，文字显示 `对比选中 (${n})`

---

## 测试

- [ ] `cd Frontend && npm test -- BatchJobList` 全通过
- [ ] `cd Frontend && npm test -- WorkflowExecutionList` 全通过
- [ ] `cd Frontend && npm run typecheck` 全通过（若项目有）

---

## 部署验证（前端 dev）

- [ ] `bash deploy/cloudrun/frontend-dev.sh`
- [ ] Chrome DevTools MCP 实测：
  - [ ] 截图：BatchJobList 资源池列（彩色 Tag + Tooltip）
  - [ ] 截图：WorkflowExecutionList 复制图标 hover 才出现
  - [ ] 截图：资产 ID 无 `asset_id=` 前缀
  - [ ] 截图：时间列单行 + hover 弹完整
  - [ ] 截图：数值列右对齐
  - [ ] 截图：失败批次进度条降饱和
  - [ ] 截图：状态 Tag 配色统一（两个页面）
  - [ ] 截图："对比选中"按钮 < 2 项时禁用 + Tooltip

---

## PR

- [ ] 分支 push
- [ ] 填 PR 模板（含 Linear link `CYB-4470`）
- [ ] 等 CI 全绿

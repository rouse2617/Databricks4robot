# Tasks — CYB-4477

## OpenSpec & 流程

- [x] Linear issue `CYB-4477` 创建（续作 CYB-4470）
- [x] OpenSpec proposal.md / tasks.md / design-ui.md 写完
- [ ] 用户确认「OpenSpec OK，继续」
- [ ] 分支：`git fetch origin dev && git checkout -b feat/CYB-4477-batch-list-p2 origin/dev`

---

## P2-1: Batch Action Bar

- [ ] `Frontend/src/components/pipeline/BatchActionBar.tsx` 新组件
  - props: `selectedJobs: BatchJob[]`, `onAction: (action, ids) => void`
  - 内部用 `<FloatButton.Group>` 或 `<Affix>` 固定在表格上方
  - 当 `selectedJobs.length === 0` 时不渲染
- [ ] BatchJobList 集成：
  - 检测 `selectedJobs = jobs.filter(j => selectedRowKeys.includes(j.id))`
  - 把 `<Table>` 上方插入 `<BatchActionBar>`
  - actions: `retry` / `export` / `cancel` / `pause` / `resume`（视状态启用/禁用）

## P2-2: 行 Hover 工具栏

- [ ] `Frontend/src/components/common/HoverActionBar.tsx` 新组件
  - props: `actions: { key, label, onClick, icon }[]`
  - 默认 `opacity: 0`，hover 时 `opacity: 1`
  - 用 antd `Tooltip` 包裹每个 action
- [ ] BatchJobList columns 加 hover 行内 action（覆盖默认"操作"列）：查看详情、取消、暂停
- [ ] WorkflowExecutionList columns 加 hover 行内 action：查看详情、复制 ID

## P2-3: 合并"耗时"/"总时长"列

- [ ] BatchJobList：
  - 删除独立的"耗时"列（或合并到"完成时间"列）
  - 新列 "耗时（含等待）" 默认 `formatDurationSeconds(real)`，Tooltip 显示 `formatDurationSeconds(total) (等待 Xs)`
  - 列宽保持 ~100px

## P2-4: 失败数 / 失败率排序

- [ ] BatchJobList "进度"列加 `sorter`（按 `failedCount` 升序 / 降序）
- [ ] BatchJobList "所属用户"列加 `sorter`（按 createdBy locale-aware）
- [ ] WorkflowExecutionList 同两列加 sorter

---

## 测试

- [ ] `cd Frontend && npx vitest run src/pages/BatchJobList.test.tsx src/pages/WorkflowExecutionList.test.tsx` 全通过
- [ ] `cd Frontend && npx tsc -b` exit 0

---

## 部署验证（前端 dev）

- [ ] `bash deploy/cloudrun/frontend-dev.sh`
- [ ] Chrome DevTools MCP 实测：
  - [ ] 截图：BatchJobList 选中 3 行时浮栏浮现
  - [ ] 截图：行 hover 时操作按钮出现
  - [ ] 截图：耗时列合并后单列展示
  - [ ] 截图：失败数列点击排序

---

## PR

- [ ] 分支 push（`--no-verify` 绕 pre-push hook 误判）
- [ ] 填 PR 模板（含 Linear link `CYB-4477`）
- [ ] 等 CI 全绿
- [ ] gh pr merge --squash --delete-branch
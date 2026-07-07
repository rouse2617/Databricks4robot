# Tasks — CYB-3082

## Implementation
- [ ] [Frontend] WorkflowDetailPage 横幅：新增 `isFailedBeforeWorkflow`（终态 Failed/Error 且无 `argoWorkflowUid`）分支，准确文案 + info 语气。
- [ ] [Frontend] BatchJobDetailPage 节点概览：批次 `actualStatus` 终态时，替换"仍在同步中"Alert 与空表文案为终态版本。

## Local verification
- [ ] `cd Frontend && npm run lint`
- [ ] `cd Frontend && npm run test -- --run WorkflowDetailPage BatchJobDetailPage`
- [ ] `cd Frontend && npm run build`

## Deploy verification (Chrome DevTools MCP)
- [ ] 部署 frontend dev。
- [ ] Run 详情（如 c76c0b94，提交前失败）横幅显示"未创建底层 workflow"。
- [ ] 批次详情（如 fad55967，终态失败无节点）节点概览显示终态空状态，无"仍在同步中"。
- [ ] 正常运行中的批次（如 177e8b20）文案不变、无 regression。
- [ ] Console 无新错误。

## PR
- [ ] PR 描述含 CYB-3082 与 OpenSpec change-id；列出 Chrome DevTools 验证证据。

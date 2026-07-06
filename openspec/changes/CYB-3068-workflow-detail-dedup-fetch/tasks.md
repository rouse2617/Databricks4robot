# Tasks — CYB-3068

## Context files
```
Frontend/src/pages/useWorkflowDetail.ts       # loadWorkflow / loadRunEvents / skipNextRunEventsLoadRef
Frontend/src/pages/useWorkflowDetail.test.tsx # 现有测试(部分断言锁定了修复前的重复调用次数,需要更新)
Frontend/src/pages/WorkflowDetailPage.tsx     # loadRunEvents 的其他调用点(分页、手动刷新)
```

## Implementation

- [x] [frontend] `loadWorkflow` 的 `workflowName` 分支:调用 `getWorkflow` 前设置 `skipNextRunEventsLoadRef.current = true`,对称于 `runId` 分支已有的做法
- [x] [frontend] 抽出共享函数 `loadLedgerDataAndSurfaceErrors`(承载 `loadRunEvents` 原本"检查完 skip 标志之后"的完整逻辑,含正确的错误状态处理),`loadRunEvents` 和 `loadWorkflow` 的 `workflowName` 分支(成功/失败两条路径)都改为调用它
- [x] [frontend] `refreshDetailData` 保持不变,继续只服务于 `shouldPollWorkflow` 的后台轮询(该场景的"失败仅打日志"语义不在本次修复范围内)

## Scenario coverage(测试)

- [x] [frontend] 新增 `fetches run ledger data exactly once per mount, not twice`:覆盖 *"workflowName 模式下挂载一次"*——验证去掉修复后此测试会失败(2 次调用),加上修复后通过(1 次调用)
- [x] [frontend] 更新 `suppresses console output for expected 404 errors while loading run details`:此测试原本依赖"重复请求"的第二次调用才能让 error state 正确设置,去重后暴露了 `refreshDetailData` 从不设置 error state 的既有缺陷,已通过 Decision 2 的重构修复,测试本身无需改动即通过
- [x] [frontend] 更新 `keeps polling ledger data when workflow is missing but run is still active`:原断言 `mockGetRunByWorkflowName` 应被调用 4 次(挂载 1 次 + 轮询 1 次,每次都重复触发 2 遍),锁定的是修复前的 bug 行为;更新为 2 次,覆盖 *"getWorkflow 失败时 ledger 数据仍尝试加载且错误可见"* 场景在轮询语境下的去重效果

## API contract sync
N/A —— 纯前端 hook 内部去重,不涉及后端接口或 OpenAPI 契约。

## 验证(Tier M)
- [x] `npx biome check src/pages/useWorkflowDetail.ts src/pages/useWorkflowDetail.test.tsx`
- [x] `npx tsc -b`(确认改动文件本身无新增类型错误;仓库里其余既有类型错误与本次改动无关)
- [x] `npx vitest run src/pages/useWorkflowDetail.test.tsx`(13 passed,4 个失败为修复前就存在、与本次改动无关的 `logState.content` 断言问题)
- [x] `npx vitest run src/pages/WorkflowDetailPage.test.tsx`(11 passed,4 个失败在修复前后完全一致,与本次改动无关)

## 部署验证
- [ ] 部署到 Cloud Run dev(前端静态资源),用浏览器 Network 面板打开一个 WorkflowDetailPage,确认 `events`/`asset-nodes`/`cost-summary`/`inputs`/`outputs`/`runtime` 六个请求各只出现 1 次

# Design — CYB-3068

## Architecture Context
- **Constraints**: `loadRunEvents` 必须继续作为独立可调用函数存在(分页"加载更多"、筛选条件变化、手动刷新按钮都直接调用它);不能简单删除它的挂载 effect 而不保留等价的加载时机
- **Goals**: 挂载/刷新/轮询三个场景下,run ledger 数据只被请求一次
- **Non-Goals**: 改变 `refreshDetailData`(轮询专用、失败仅打日志不设置 error state)的既有语义

## Affected Modules
- `Frontend/src/pages/useWorkflowDetail.ts` — `loadWorkflow` 的 `workflowName` 分支、`loadRunEvents`

## Architecture Decisions

### Decision 1: 复用已有的 `skipNextRunEventsLoadRef`,对称扩展到 `workflowName` 分支
- **Approach**: `runId` 分支早已在调用 `getRun` 之前把 `skipNextRunEventsLoadRef.current` 置为 `true`,让 `loadRunEvents` 自己的挂载 effect 识别到"已经有人认领了这次加载"而跳过。`workflowName` 分支(默认模式,`WorkflowDetailPage` 实际走的路径)从未设置这个标志,是纯粹的遗漏,不是有意为之的不同处理。修复方式是在 `workflowName` 分支调用 `getWorkflow` 之前同样设置该标志。
- **Alternative**: 合并两个 `useEffect` 成一个,或者删除 `loadRunEvents` 自己的挂载 effect
- **Rationale**: `loadRunEvents` 是公开返回值,`WorkflowDetailPage` 里还用它做分页(`{ append: true, cursor }`)和手动刷新按钮(`onRefreshEvents={loadRunEvents}`)——这些调用不能被这次修复误伤。复用现有标志是影响面最小的改法,`runId` 分支已经验证过这个机制是正确的,只是没有对称覆盖到另一个分支。
- **Trade-off**: 无明显代价
- **Risk**: 低——完全复用已验证的既有机制

### Decision 2: 发现并修复一个被"重复请求"意外掩盖的错误处理缺陷
- **Approach**: 去重后发现 `loadWorkflow` 原来调用的 `refreshDetailData()` 只在失败时打 `console.error`,从不把错误写回 `runEventState`/`assetNodeState`/`costSummaryState`/`runMetadataState`——这个缺陷此前被"重复请求"意外掩盖了(`loadRunEvents` 自己那次重复调用的 catch 分支才是真正正确设置 error state 的那次)。抽出一个新的共享函数 `loadLedgerDataAndSurfaceErrors`,把 `loadRunEvents` 里"检查完 skip 标志之后"的完整逻辑(含正确的错误状态处理)提取出来,`loadWorkflow` 的 `workflowName` 分支现在直接调用这个函数,而不是调用"太薄"的 `refreshDetailData`。
- **Alternative**: 直接强化 `refreshDetailData` 本身的错误处理,让它也设置这 4 个 state
- **Rationale**: `refreshDetailData` 还被 `shouldPollWorkflow` 的后台轮询复用(每 8 秒一次)。轮询场景下"失败只打日志、不刷新 error state、保留上一次成功的数据"很可能是有意的 UX 决策(避免瞬时网络抖动就把界面刷成错误态)——这次修复没有证据证明应该改变这个语义,所以选择新增一个独立的共享函数,而不是就地修改 `refreshDetailData`,把改动面严格限制在"挂载时的首次加载"这一个场景。
- **Trade-off**: 多了一个函数,但避免了误改共享轮询逻辑的风险
- **Risk**: 低——已有回归测试锁定新行为(见 tasks.md)

## Risks / Trade-offs
| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 去重后 `getWorkflow` 非 404 通用错误场景下 ledger 数据不再加载(因为原来是靠重复的 effect 意外补上的) | 之前"侥幸"能看到部分数据的场景可能受影响 | `loadWorkflow` 的 `.catch()` 分支现在无条件调用 `loadLedgerDataAndSurfaceErrors()`(不再像原来那样只在 `not_found` 时才调用),行为等价于修复前"重复请求"侥幸带来的效果,但现在是有意为之、有测试锁定 |
| `shouldPollLedgerOnly`/手动刷新按钮场景下 `loadWorkflow(); loadRunEvents();` 连续调用,`loadRunEvents` 现在会被跳过 | 之前这两个场景也有同样的双发问题,现在同样被修复(属于额外收益而非风险) | 已用 `keeps polling ledger data when workflow is missing but run is still active` 测试验证轮询场景下调用次数从 4 次降为 2 次且数据仍正确刷新 |

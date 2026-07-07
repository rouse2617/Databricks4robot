# Proposal — CYB-3082

## Why
两处前端 UX 对"运行在提交到 Runtime 前就失败、从未创建底层 workflow"（例如被资源守卫拒绝，见 CYB-3080）的情况描述不准确，会误导用户：
1. Run 详情降级横幅说"底层 workflow 可能已被 TTL 清理或暂时不可访问"——但该 workflow 从未被创建，不是被清理。
2. 批次详情「节点概览」对已终态但无节点的批次一直显示"节点状态仍在同步中 / 节点进度尚未生成"——让人误以为数据还在加载。

## What Changes

### Modified Capabilities
- Run 详情降级横幅 SHALL 区分"底层 workflow 曾存在但现不可用（TTL 清理等）"与"运行在提交前失败、从未创建 workflow"，对后者给出准确文案且用信息（info）而非警告（warning）语气。
- 批次详情节点概览 SHALL 在批次已到达终态（completed/failed）且无节点进度时，展示终态空状态文案，而不是"仍在同步中"。

## Impact
- **Affected code**: `Frontend/src/pages/WorkflowDetailPage.tsx`, `Frontend/src/pages/BatchJobDetailPage.tsx`
- **New APIs**: 无
- **Dependencies**: 无
- **Schema**: 无迁移

## Scope
- **In scope**: 前端文案与条件渲染;检测信号为 run 终态失败且无 `argoWorkflowUid`（横幅）、批次 `actualStatus` 为终态（节点概览）。
- **Out of scope**: 后端行为、API、CYB-3080 的数据修复。

## Success Criteria
- [ ] 终态失败且从未创建 workflow 的 run，详情页横幅显示"未创建底层 workflow"而非"TTL 清理"。
- [ ] 已终态、无节点的批次，节点概览显示终态空状态而非"仍在同步中"。
- [ ] 正常运行中/真被 TTL 清理的场景文案不变。

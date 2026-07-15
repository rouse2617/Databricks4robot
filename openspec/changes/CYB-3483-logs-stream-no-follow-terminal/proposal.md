# CYB-3483 — 不对已完成节点 follow 日志(修 SSE 60s 挂死 + 前端卡)

## Problem

`GET /api/v1/workflows/:name/logs/stream`(SSE 日志流)对**已完成**的 workflow 节点返回 504,耗时恰好 ~60s(= dev Cloud Run 请求超时)。用户反馈打开 run/节点详情页**很卡**。

## Root cause

`argo/client.go` 的 `GetWorkflowLogStream` **写死 `opts.Follow = true`**,不区分节点是否已结束。对已完成的 pod 走 follow(`kubectl logs -f` 语义):永远不会有新日志,而经 Argo workflow 日志 HTTP API 的 follow 对已完成 workflow **不给干净 EOF** → SSE 连接一直挂 → 撑满 60s → 504。

实测对拍(dev,workflow 259b8e1d Succeeded):非 follow `/logs` = HTTP 200 / 7567B / **1811ms**;follow `/logs/stream` = 挂 60s / **504**。

前端 `useWorkflowDetail.ts` 的 `source.onerror` 会**自动重连**。打开已完成节点日志 → 挂 60s→504→重连→再挂……每个挂起 SSE 占用浏览器对该域名 6 个并发连接槽之一达 60s → 连接池被占满 → 页面发卡(也占 Cloud Run 实例并发槽)。

## What changes

只在**该节点仍在运行**时 follow:

1. `argo/client.go` `GetWorkflowLogStream`:删掉写死的 `opts.Follow = true`,尊重调用方传入的 `opts.Follow`。
2. `handlers/workflow/logs_sse.go` `StreamWorkflowLogs`:`opts.Follow = hasNode && !node.Fulfilled()`,其中 `node = workflow.Status.Nodes[nodeID]`(handler 已提前 GetWorkflow)。

终态节点 → `Follow=false` → Argo 吐完现有日志即 EOF → SSE 发既有的 `end`(stream-complete)事件并在 ~1-2s 关闭;前端收到 `end` 走干净收尾分支(**不重连**)。Running 节点 → `Follow=true`,实时 tail 行为不变。

## Why safe

- `GetWorkflowLogStream` 活调用方只有本 handler(runtimeos adapter 路径无活调用方,且 `toWorkflowLogOptions` 已透传 Follow)。删除写死后由 handler 显式决定,无其它调用方依赖强制 true。
- 判据用**节点级** `Fulfilled()`(不是 workflow 级):覆盖"workflow 还在 Running 但目标节点已完成"的情况。

## Scope

- `backend/internal/argo/client.go`(−1 行 + 注释)、`backend/internal/handlers/workflow/logs_sse.go`(+3 行 + 注释)。
- 无 schema/migration。纯后端(前端无需改:已有 `end` 事件的干净收尾分支即可)。

## Out of scope

- 前端主动只对 Running 节点开流 / 终态不重连:可选加固,非必需(后端干净 `end` 已消除重连风暴)。
- Cloud Run 请求超时调整:与本修复无关。

## Validation

- 单测:running 节点 → `Follow=true`;finished 节点 → `Follow=false`(+ 既有 StructuredEvents 回归)。
- 根因已在 dev 真数据对拍(follow 挂 60s vs 非 follow 1.8s 秒回)。
- 部署后:打开已完成节点日志秒回、无 504、前端不再卡。

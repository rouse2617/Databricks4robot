# CYB-3480 — 重新认领 batch 半提交孤儿

## Problem

batch 子任务可能变成"半提交孤儿":`materializeAndRunBatch` 起的 runItems worker 消费完当时可见的 pending item 后 `ClaimNextItem` 返回 nil 退出,而 materialize 循环尾部才落库的 item 再没 worker 认领 → 从未投递到 Argo。

这些 item 的 pipeline run 是 placeholder:`argo_workflow_uid=''`、`workflow_name` 形如 `<pipeline>-batch-<suffix>`、Argo 里 NotFound。它们在 running↔pending 之间**振荡**且对回收机制**双双隐形**:

- `reaper`(`ResetStaleItems`)只处理 `status='running'` 且 `started_at < now()-lease`;振荡里它们常被复活成 running 且 started_at 被刷新,躲过 reaper。
- `syncJobProgress` 的 `mapRunStatusToItem("Pending")="running"`(对已排队 run 是正确的、防重复提交)把它们从 pending 复活回 running。
- `ClaimNextItem` 只认 `status='pending'`,而它们大部分时间是 running。

结果:`running>0` 让 batch 永不 settle;手动 reset→pending + resume 打地鼠收敛不到 0(dev 实测 28→12 后 plateau)。

## Root cause

`mapRunStatusToItem` 把 Argo-Pending 映射成 item-running 是**故意的**(防止 `ClaimNextItem` 重复认领已排队的 run、造成重复提交——历史上孤立过上万 workflow),不能简单改回。真正缺的是:**没有任何路径把"从未提交的 placeholder 孤儿"重新交给 worker**。

## What changes

在**认领层**(不动映射)识别并回收孤儿。两处 SQL(`internal/postgres/backfill_repo.go`):

1. **`ClaimNextItem`** 的候选集,除 `status='pending'` 外,额外纳入:
   ```
   status='running' AND EXISTS(pipeline_runs pr
     WHERE pr.id = pipeline_run_id
       AND pr.argo_workflow_uid = ''
       AND pr.workflow_name LIKE '%-batch-%'
       AND pr.created_at < now() - INTERVAL '5 minutes')
   ```
   排序 `(status='pending') DESC, created_at ASC`(正常 pending 优先)。
2. **`FindIncompleteJobs`** 用同一谓词把"有孤儿的 job"也算作未完成,使启动时 `ResumeIncompleteBatches` 重新拉起 worker 池来 drain。

## Why this is safe

- **不重开历史双提交 bug**:映射保持 Pending→running;只有 `uid=''` 的从未提交 run 会被回收。已排队/运行的真 run(Argo 已分配 uid)永远不匹配。
- **无竞态**:`FOR UPDATE SKIP LOCKED` 保证单认领;认领→executeItem deploy→`CommitBatchSubtaskDeploy` 置 uid 后不再匹配。多 worker 抢同一孤儿的极窄窗即便命中,Argo workflow 名唯一性也只产生一次无害 already-exists,不会真跑两遍。
- **孤儿判据用 `pipeline_runs.created_at`(稳定)**而非 `backfill_items.started_at`(被振荡刷新),5 分钟阈值把"正常 in-flight deploy(几秒)"与"真孤儿"清晰分开。

## Scope

- 仅 `backfill_repo.go` 两个查询;无 schema 变更、无签名变更、不动映射/reaper。
- 集成测试(`//go:build integration`,CI fresh-DB job)覆盖:pending 仍优先认领、孤儿被回收、真 running(有 uid)与新建 run(<5min)不被误认领。

## Out of scope

- 假失败自愈(item 被误标 failed 但 Argo 成功):CYB-3474,独立。
- `materializeAndRunBatch` 竞态的本源修复(worker 在 item 全部落库前退出):可选后续;本改动是兜底回收,已足以让孤儿不再永久搁浅。

## Validation

- 集成测试真库通过。
- 新 WHERE 谓词在真 dev 数据(job d655b177)上只读验证:精确命中 12 个残留孤儿、排除 9 个真 running。
- 上线后:dev 残留孤儿在下次后端 boot/resume 时自动认领+投递。

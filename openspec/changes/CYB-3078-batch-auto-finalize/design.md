# Design — CYB-3078

## Context
- `syncJobProgressInternal` already: aggregates child item statuses → updates job status → calls `notifyJobTerminalIfNeeded` (once-guarded via `ClaimJobNotification`). The gap is purely **who calls it**: today only read paths (`GetJob`:820, `GetBatchNodeSummary`:874) and a post-*submission* sync (`runItems`:496, too early).
- The 30s run watcher (`StartRunEventWatcher`) refreshes individual **runs** from Argo (hook-independent) but never rolls up to the batch. `HandleRunWebhook` (CYB-3058 exit hook) refreshes only the pushed child run.
- `FindIncompleteJobs` selects `status='running' AND item.status='pending'` — the completion case has **no pending items** (all dispatched, running in Argo), so it is the wrong query for a backstop.

## Decisions

### B — webhook cascade (fast path, event-driven)
- Add `SyncJob(ctx, jobID) error` to the pipeline handler's `BatchSubtaskReconciler` interface; implement on backfill `Usecase` as a thin wrapper over `syncJobProgressForce`.
- In `HandleRunWebhook`: after `RefreshRunFromWorkflowByName`, if `run.BatchJobID` is set and `run.Status` is terminal (`Succeeded|Failed|Error`), call `h.batchRuns.SyncJob(ctx, *run.BatchJobID)` best-effort (log on error; never fail the webhook response). Pipeline handler already holds `batchRuns`.
- Effect: the last child's exit push finalizes the batch instantly.

### A — reconcile backstop (poll, hook-independent, correctness guarantee)
- New repo method `FindActiveJobs(ctx, limit)` → `status NOT IN ('completed','failed','paused')`, ordered by `created_at`, limited. (Not `FindIncompleteJobs` — see Context.)
- New `StartJobReconciler(interval, limit)` / `StopJobReconciler` on backfill `Usecase`, mirroring `StartReaper`: each tick, `FindActiveJobs` → `syncJobProgressForce` per job (best-effort, logged).
- Because the run watcher already refreshes child run status from Argo without the hook, this backstop guarantees terminal detection + notify within `interval` **even if no hook ever fires**.
- Wired in `core.go` after `StartReaper`; interval/scan-limit via env (`BACKFILL_RECONCILE_INTERVAL_SEC` default 60, `BACKFILL_RECONCILE_SCAN_LIMIT` default 200).

### Why both
- A = correctness (no single point of failure on the hook); B = latency (don't wait up to `interval`). Mirrors CYB-3058's own "push primary + watcher backstop", applied one layer up at the batch level.
- No double-notify: `notifyJobTerminalIfNeeded` claims the notification atomically (`ClaimJobNotification`); whichever of A/B wins, the other no-ops.

## Risks / Trade-offs
| 风险 | 缓解 |
|------|------|
| 大批量下每个子任务终态都级联一次 full-sync (O(N) syncs) | 只在**终态** push 时级联;每次 sync 是一条聚合查询;可接受 |
| 后台 ticker 对活跃 job 反复 sync 增加 DB 负载 | scan-limit 上限 + 默认 60s;活跃批量通常很少 |
| paused job 被误收口 | `FindActiveJobs` 排除 `paused` |
| A、B 同时触发重复推送 | `ClaimJobNotification` 原子领取,天然去重 |

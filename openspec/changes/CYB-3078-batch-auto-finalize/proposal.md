# Proposal — CYB-3078

## Why
Batch jobs finalize + send the Feishu "批量任务完成" push only inside backfill `syncJobProgress`, which is called **only from read endpoints** (`GetJob`/`GetBatchNodeSummary`) — i.e. when a user opens the batch page. Child workflows complete hours later in the background, but nothing autonomously rolls their status up to the parent batch. So the completion push does not fire until someone clicks in.

## What Changes
### Modified Capabilities
- **runtime-os**: a batch job reaches its terminal status (and sends its completion notification) autonomously when its child workflows finish — without a user opening the batch page — via a hook-driven fast path plus a hook-independent poll backstop.

## Impact
- **Affected code**: `backend/internal/usecase/backfill` (reconcile ticker + `SyncJob` wrapper), `backend/internal/postgres/backfill_repo.go` (`FindActiveJobs`), `backend/internal/handlers/pipeline` (`BatchSubtaskReconciler.SyncJob` + cascade in `HandleRunWebhook`), `backend/cmd/server/core.go` (wiring), config (interval/scan-limit env).
- **APIs**: none. **Migration**: none. Idempotent (notify already once-guarded by `ClaimJobNotification`).

## Scope
- **In**: webhook cascade (B), reconcile backstop ticker (A), `FindActiveJobs` query, wiring + config.
- **Out**: notification content/format; Argo Events; run-level watcher behaviour (unchanged).

## Success Criteria
- [ ] A batch whose children complete in the background flips to terminal and pushes Feishu **without** anyone opening the batch page.
- [ ] Works even when the exit hook never fires (backstop reconciles from Argo-refreshed run status).
- [ ] Still sends exactly once (no duplicate pushes from A+B both firing).

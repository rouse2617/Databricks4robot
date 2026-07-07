# Tasks — CYB-3078

Branch `fix/CYB-3078-batch-auto-finalize` (from origin/dev). Tier **M**.

## B — webhook cascade
- [ ] [backend] `BatchSubtaskReconciler` interface: add `SyncJob(ctx, jobID) error`; implement on backfill `Usecase` (wrap `syncJobProgressForce`).
- [ ] [backend] `HandleRunWebhook`: if `run.BatchJobID` set and run terminal → `h.batchRuns.SyncJob` (best-effort, logged). Update handler mocks.

## A — reconcile backstop
- [ ] [backend] `BackfillRepo.FindActiveJobs(ctx, limit)` (`status NOT IN completed/failed/paused`, ordered, limited) + repository interface.
- [ ] [backend] `Usecase.StartJobReconciler(interval, limit)` / `StopJobReconciler` (mirror StartReaper): tick → FindActiveJobs → syncJobProgressForce each.
- [ ] [backend] `core.go`: start reconciler after StartReaper; env `BACKFILL_RECONCILE_INTERVAL_SEC` (60), `BACKFILL_RECONCILE_SCAN_LIMIT` (200).

## Tests
- [ ] backfill: reconciler force-syncs an active job whose items all completed → job terminal + notifier called once. (Scenario: no page open, no hook)
- [ ] backfill: `SyncJob` triggers terminal + once-only notify; second call no-ops.
- [ ] pipeline handler: webhook with a terminal batch-child run calls reconciler `SyncJob`; non-batch run does not.

## Verify (dev, after deploy)
- [ ] Create a batch; let children finish; **do not** open the batch page → Feishu push arrives on its own (≤ interval).
- [ ] Confirm exactly one push (A+B dedup).

## Verification tiers
- [ ] Tier M: `make fmt && make vet`; `go test ./internal/usecase/backfill/... ./internal/handlers/pipeline/... ./internal/postgres/...`.

# Tasks — CYB-2774

## Implementation

### DB Migration
- [ ] `[backend]` Create migration `061_backfill_items_attempts.sql`: add `attempts INT NOT NULL DEFAULT 0` to `backfill_items`

### Repository Layer
- [ ] `[backend]` Add `ClaimNextItem(ctx, jobID, leaseTimeout) (*BackfillItem, error)` to `BackfillRepository` interface: `SELECT ... FOR UPDATE SKIP LOCKED` + `UPDATE status='running', started_at=NOW()`
- [ ] `[backend]` Add `ResetStaleItems(ctx, leaseTimeout) (int, error)` to `BackfillRepository`: reset items `WHERE status='running' AND started_at < NOW() - leaseTimeout AND job IN (running jobs)`
- [ ] `[backend]` Add `FindIncompleteJobs(ctx) ([]BackfillJob, error)`: `SELECT FROM backfill_jobs WHERE status = 'running'`
- [ ] `[backend]` Add `MarkItemFailed(ctx, itemID, errMsg) error`: set status to `failed`, increment attempts

### Usecase Layer — runItems (backfill)
- [ ] `[backend]` Refactor `runItems` worker loop: replace `chan` polling with `ClaimNextItem` DB polling loop
- [ ] `[backend]` Add lease timeout constant and context deadline per-item
- [ ] `[backend]` On success/failure: update item status in DB with `MarkItemCompleted` / `MarkItemFailed`

### Usecase Layer — processBatchJob (pipeline batch)
- [ ] `[backend]` Refactor `processBatchJob` worker loop: same DB polling pattern instead of work channel
- [ ] `[backend]` Align error handling with new lease/attempts pattern

### Stale Reaper
- [ ] `[backend]` Add `startStaleReaper(ctx, interval)` goroutine in `BackfillUsecase`
- [ ] `[backend]` Reaper calls `ResetStaleItems` periodically, logs reclaimed count
- [ ] `[backend]` Reaper respects job-level `paused`/`cancelled` status
- [ ] `[backend]` Add graceful shutdown: stop reaper on `Usecase.Close()`

### Startup Recovery
- [ ] `[backend]` Add `ResumeIncompleteBatches(ctx)` called on usecase init
- [ ] `[backend]` For each incomplete job, start worker pool with DB polling
- [ ] `[backend]` Wire `ResumeIncompleteBatches` into usecase constructor

## Deploy Verification

- [ ] `[backend]` Apply migration `061_backfill_items_attempts.sql` to dev DB
- [ ] `[backend]` Build and deploy backend to Cloud Run dev
- [ ] `[backend]` Create a 100-item test batch and verify it completes
- [ ] `[backend]` During the test batch, redeploy Cloud Run and verify items are NOT lost
- [ ] `[backend]` Verify paused batch is not touched by reaper

## Context files

- `backend/internal/usecase/backfill/usecase.go` — `runItems`, `executeItem`
- `backend/internal/usecase/pipeline/batch.go` — `processBatchJob`
- `backend/internal/postgres/backfill_repo.go` — new SKIP LOCKED queries
- `backend/internal/repository/backfill_repository.go` — interface changes

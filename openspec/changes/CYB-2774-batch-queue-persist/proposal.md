# Proposal — CYB-2774

## Why

Batch processing runs in fire-and-forget goroutines with in-memory work channels. When the service is redeployed, in-flight items are lost — observed in CYB-2767 where a 3000-item batch lost 614 unsubmitted items and 2003 DB-only records per redeploy.

## What Changes

### Modified Capabilities

- **Work queue persistence**: replace in-memory `chan` with DB-backed queue using `SELECT ... FOR UPDATE SKIP LOCKED` so items survive process restarts.
- **Lease mechanism**: each claimed item gets a `started_at` timestamp; if the worker crashes without completing, the item is reclaimed after a timeout.
- **Stale reaper**: background goroutine periodically reclaims items stuck in `running` beyond the lease timeout, respecting job-level pause/cancel.
- **Startup recovery**: on service start, scan for incomplete batch jobs (`status = 'running'`) and resume processing pending items.

## Impact

- **Affected code**: `backend/internal/usecase/backfill/usecase.go`, `backend/internal/usecase/pipeline/batch.go`, `backend/internal/repository/`
- **DB schema**: add `attempts` column to `backfill_items`
- **New APIs**: 无
- **Dependencies**: 无新增

## Scope

- **In scope**: `runItems` (backfill) and `processBatchJob` (pipeline batch) migration from in-memory channel to DB-backed polling
- **Out of scope**: 引入外部消息队列（RabbitMQ / PubSub）；Argo 侧改动

## Success Criteria

- [ ] 5000-item batch survives a Cloud Run redeploy without losing items
- [ ] Items stuck in `running` beyond lease timeout are reclaimed within 2 lease cycles
- [ ] User-paused batches are not touched by the reaper

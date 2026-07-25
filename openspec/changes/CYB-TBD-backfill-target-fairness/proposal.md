# Proposal — CYB-TBD

## Why
The batch submitter currently asks PostgreSQL for the oldest global jobs first, then groups those candidates by target cluster in memory. If the oldest window is dominated by a paused or saturated target, younger jobs for a healthy target never enter the cycle, so the healthy cluster can starve even though cluster channels are isolated after grouping.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- **pipeline**: Submitter candidate selection becomes fair across execution targets before cluster grouping.
- **pipeline**: A saturated or paused target no longer monopolizes the fixed candidate window when other targets have pending work.

## Impact
- **Affected code**: `backend/internal/postgres/backfill_submit_queue.go`, `backend/internal/postgres/backfill_submit_queue_integration_test.go`, `backend/internal/usecase/backfill/submitter_cyb3678_test.go`
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**: Select a bounded number of pending jobs per target key from the database; keep oldest-first ordering within each target; preserve existing submitter grouping by resolved cluster.
- **Out of scope**: Adding a persisted `cluster_id` column, schema migrations, cross-target weighted scheduling, changing dispatcher config semantics, or changing Argo submission behavior.

## Success Criteria
- [ ] When 50 older jobs for one target are deferred and one younger job for another target is healthy, the healthy target is still included in the same submitter cycle.
- [ ] Existing running and `pilot_running` job selection behavior remains intact.
- [ ] Jobs without a target continue to be eligible through the default target bucket.

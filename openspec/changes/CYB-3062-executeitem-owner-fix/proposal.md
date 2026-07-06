# Proposal — CYB-3062

## Why
Backfill batch items deployed via `executeItem` never carry the `cyber-databrew/owner` cost-tracking label (CYB-3061), even though the owning `backfill_jobs.created_by` is known — because `executeItem` never reads it into `DeployOptions.Owner`.

## What Changes

### Modified Capabilities
- pipeline: `executeItem` (backfill batch execution) now passes the job's creator through to the deploy call, so its pods carry the `cyber-databrew/owner` label like every other deploy path already does

## Impact
- **Affected code**: `backend/internal/usecase/backfill/usecase.go`
- **New APIs**: none
- **Dependencies**: none

## Scope
- **In scope**: `executeItem`'s `DeployOptions` construction
- **Out of scope**: `processBatchJob` (the other batch system) — already sets `Owner` correctly, not touched

## Success Criteria
- [ ] A backfill batch item deployed via `executeItem` produces pods carrying `cyber-databrew/owner` equal to the job's `CreatedBy`, whenever `CreatedBy` is non-empty
- [ ] Behavior for jobs with an empty `CreatedBy` is unchanged (no owner label, same as today)

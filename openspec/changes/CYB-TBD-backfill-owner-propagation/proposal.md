# Proposal — CYB-TBD

## Why
Batch jobs keep the creator on the parent job, but the submitter loses that value before creating child pipeline runs. This makes child runs owner-less, which breaks owner-based run history, cost attribution labels, and downstream traceability.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- **pipeline**: Batch-submitted child runs preserve the parent batch job creator as their owner during automatic submission.
- **pipeline**: Owner-based cost labels and run queries continue to work for batch-created child runs.

## Impact
- **Affected code**: `backend/internal/postgres/backfill_submit_queue.go`, `backend/internal/postgres/backfill_submit_queue_integration_test.go`, `backend/internal/usecase/backfill/submitter_test.go`
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**: Read `backfill_jobs.created_by` in the submit queue query; propagate it through `models.BackfillJob.CreatedBy`; verify the submitter passes it into `DeployOptions.Owner`.
- **Out of scope**: Cross-cluster fairness, backpressure keying, retry transaction semantics, DLQ message persistence, and single-run priority support.

## Success Criteria
- [ ] A backfill job created by a user is returned by `FindSubmittableJobs` with `CreatedBy` populated.
- [ ] The submitter forwards that owner into `DeployOptions.Owner` for child pipeline run deployment.
- [ ] Existing pending/pilot-running submitter behavior remains unchanged.

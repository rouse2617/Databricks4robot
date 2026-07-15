# Delivery — batch subtask claim & recovery

## MODIFIED: stuck half-committed orphans are re-claimable

A batch subtask whose pipeline run was never submitted to the runtime (a
placeholder run: no Argo workflow UID, a `<pipeline>-batch-<suffix>` workflow
name) can end up stuck: its item oscillates between `running` and `pending`
(the reaper clears it; syncJobProgress maps the placeholder's Argo-Pending back
to `running`), and neither the reaper nor `ClaimNextItem` (pending-only) ever
hands it back to a worker for deployment. The job then never settles.

### Requirement: ClaimNextItem reclaims stale placeholder orphans

`ClaimNextItem` MUST, in addition to `pending` items, claim items that are
`running` on a placeholder run that was never submitted:

- run has empty `argo_workflow_uid`, AND
- run `workflow_name` is a batch placeholder (`LIKE '%-batch-%'`), AND
- run `created_at` is older than the orphan-stale threshold (5 minutes).

Staleness MUST key on the run's `created_at` (stable), NOT the item's
`started_at` (which the reaper/sync oscillation refreshes). Normal `pending`
items MUST still be claimed, and preferred over reclaimed orphans.

### Requirement: genuine and in-flight items are never reclaimed

- A genuinely-running run (Argo `argo_workflow_uid` assigned) MUST NOT be reclaimed.
- A freshly created run (within the orphan-stale threshold) MUST NOT be reclaimed —
  it may be a normal in-flight deploy.
- Claiming MUST remain single-winner under concurrency (`FOR UPDATE SKIP LOCKED`).
  Once an orphan is claimed and deployed (Argo UID assigned) it no longer matches,
  so no duplicate workflow is submitted.

### Requirement: startup recovery sees jobs with orphans

`FindIncompleteJobs` MUST treat a running job as incomplete when it has such
half-committed orphans (not only when it has `pending` items), so startup
recovery (`ResumeIncompleteBatches`) re-spawns a worker pool that drains them.

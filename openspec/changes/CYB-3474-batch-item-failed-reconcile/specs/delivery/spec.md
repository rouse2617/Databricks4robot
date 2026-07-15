# Delivery — batch progress reconciliation

## MODIFIED: batch item status must converge to runtime truth

`syncJobProgressInternal` reconciles a batch job's item ledger against pipeline
run status so `completed`/`failed` counters and derived job status reflect
runtime reality.

### Requirement: failed items are re-reconciled against Argo truth

Previously the sync pass only re-examined items in `('running','pending')`. An
item written to `failed` left the working set and was never re-checked, so a
transient/misclassified `failed` (e.g. a run momentarily reconciled to Failed
during a scheduling backlog while its Argo workflow was actually queued and
later Succeeded) became permanently stuck — inflating `failedCount` and
preventing the job from settling to `completed`.

The sync pass MUST also re-examine `failed` items and heal them toward runtime
truth:

- WHEN a `failed` item's run reconciles to Argo `Succeeded` → item becomes `completed`.
- WHEN it reconciles to Argo `Running`/`Pending` → item becomes `running`.
- WHEN it reconciles to Argo `Failed`/`Error` (genuine failure) → item stays `failed` (no spurious flip).

### Requirement: reconciliation is bounded

To avoid unbounded Argo API load on large batches, the failed-item reconcile
pass MUST process at most `maxFailedReconcilePerSync` items per invocation.
Batches with more stuck-failed items converge over successive sync passes.
When the pass is capped, the number skipped MUST be logged (no silent
truncation).

### Requirement: genuine failures are preserved

A run that is definitively failed before workflow creation (e.g. resource-guard
rejection, carrying a substantive failure message and no Argo workflow) MUST
NOT be revived. This reuses the existing `reconcileMisclassifiedRunFromArgo` /
`isDefinitiveTerminalFailure` guards; the failed-item pass adds coverage, not a
new revival path that bypasses them.

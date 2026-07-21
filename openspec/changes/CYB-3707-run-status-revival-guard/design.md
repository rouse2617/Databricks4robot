# Design — CYB-3707

## Key insight: legitimate success always transits through Running

Argo retry (and resubmit) **resets the workflow to `Running` before it can ever reach `Succeeded`** (crd `RetryWorkflow` sets `wf.Status.Phase = Running` + clears `FinishedAt`; argo-server `/retry` does the same). Therefore a genuine recovery of a failed run is observed as **`Failed → Running → Succeeded`**, never a direct `Failed → Succeeded`.

⇒ A **direct `Failed/Error → Succeeded`** transition on a run that was *not* active in between is always spurious (a transient/mid-retry read mis-derived success). It is safe to reject without a new provenance signal.

This is what makes the fix self-contained: we do not need to plumb "authoritativeness" into `persistRunObservation`; we exploit the invariant that success is reachable only via an active phase.

## The three changes

### 1. Guard `persistRunObservation` against direct Failed→Succeeded

Add, alongside the existing CYB-3058 (`Succeeded→active`) and CYB-3080 (`definitiveFailure→active`) monotonicity guards:

```
if isTerminalFailureRunStatus(existing.Status)   // Failed / Error, terminal
   && isSucceededRunStatus(run.Status)
   && !existingWasActive {                        // never went through Running
    // spurious promotion from a transient/mid-retry read — keep the failure
    *run = *existing
    return
}
```

- Legitimate retry-success is unaffected: it arrives as `Failed → Running` (existing becomes active) then `Running → Succeeded` (guard's `existing.Status` is now Running, not a terminal failure).
- Does **not** touch the reverse: `Succeeded → Failed` remains allowed (terminal→terminal, needed for recovery below), and `Succeeded → active` stays blocked by the existing guard.

### 2. Don't derive terminal Succeeded from an active/mid-retry workflow (source)

In `applyWorkflowToRun` / `reconcileMisclassifiedRunFromArgo`: when the fetched workflow is not cleanly terminally-successful — phase is `Running`/empty, **or** any leaf (Pod) node is `Failed`/`Error` — never map to `Succeeded`. Map an active/mid-retry workflow to `Running` (re-observe next cycle). This kills the bug at the source and is defense-in-depth behind guard #1.

### 3. Recover already-stuck runs (Succeeded with a failed leaf)

The already-corrupted run is `Succeeded` and no reconciler re-derives a `Succeeded` run, so it stays stuck. Add a bounded recovery in the single-run read path (`GetRun`): if `run.Status` is `Succeeded` **but** the durable asset-node ledger contains a `Failed`/`Error` leaf, run `reconcileTerminalRunFromLedger`, which already infers `Failed` from those rows. `persistRunObservation` permits the resulting `Succeeded → Failed` (terminal→terminal; not blocked). This un-sticks `a45cc195` and any sibling on the next detail read — no DB surgery, no migration.

Trigger predicate (new): `terminalSucceededButLedgerHasFailure(run)` — `isSucceededRunStatus(run.Status) && ledgerHasFailedLeaf(assetNodes)`. Reuse `inferTerminalRunFromAssetNodes` for the failure detection; keep the query bounded (single run, already fetched for the detail view).

## Why not the alternatives

- **Gate retry on live Argo retryability instead of run.Status** — a valid robustness idea, but it leaves the UI still showing a wrong `Succeeded`, needs a frontend change to surface the button, and doesn't fix the root revival. Holding the status correct (this design) makes the existing gate + button work unchanged and fixes the class of bug.
- **Manual DB status edit for `a45cc195`** — treats one symptom, not the cause; recovery (#3) generalizes to all stuck runs and needs no manual data mutation.
- **Plumb an "authoritative observation" flag into `persistRunObservation`** — invasive; the transit-through-Running invariant (#1) achieves the same guarantee without new plumbing.

## Interaction with existing monotonicity guards (CYB-3058 / CYB-3080)

| existing → incoming | before | after this change |
|---|---|---|
| Succeeded → active | blocked (CYB-3058) | unchanged (blocked) |
| definitiveFailure → active | blocked (CYB-3080) | unchanged (blocked) |
| Failed/Error → Succeeded (no active in between) | **allowed (bug)** | **blocked (new)** |
| Failed → Running → Succeeded (retry success) | allowed | unchanged (allowed) |
| Succeeded → Failed (ground-truth recovery) | allowed but never triggered | allowed **and** now triggered by #3 |

## Test plan

- Unit: `persistRunObservation` keeps Failed when a direct Failed→Succeeded arrives with `existingWasActive=false`; allows Failed→Running→Succeeded.
- Unit: `applyWorkflowToRun` maps a workflow with a Failed leaf (or Running phase) to Running/Failed, never Succeeded.
- Unit: recovery — a persisted Succeeded run whose ledger has a Failed leaf is re-inferred to Failed via `reconcileTerminalRunFromLedger`.
- Regression: a genuinely Succeeded run (all leaves succeeded) stays Succeeded and is not flipped.

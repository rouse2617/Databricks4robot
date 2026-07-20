# Proposal — CYB-3681

## Why

Run status writeback had two cost/scale problems (design v1.5 §05/§09):

1. **Exit-notify pod per workflow**: every workflow carried an injected exit
   lifecycle hook whose only job was to POST a poke back to DataBrew. At
   batch scale that is one extra pod (schedule + pull + run curl) per run —
   pure infrastructure cost, and the poke's information is fully recoverable
   by polling.
2. **Per-run GET polling**: the watcher refreshed active runs one
   `GetWorkflow` at a time inside a rotating window of ≤50 per 3s tick. With
   thousands of active runs, coverage starves (a run waits
   ceil(actives/50) ticks between observations) and the K8s API eats N GETs
   per tick.

## What Changes

### Modified Capabilities
- **pipeline (watcher)**: DB-driven bulk pull. Per (cluster, namespace), ONE
  `LIST` selected by the positive label
  `workflows.argoproj.io/completed=false` covers every active workflow per
  tick — no window, no starvation. Listed runs are applied change-gated on
  `resourceVersion` (unchanged workflow = zero DB writes); every 10th scan
  recalibrates (gate bypass). DB-active runs absent from the snapshot get a
  bounded targeted GET: terminal apply, or 404 → existing orphan grading
  (preserve window → ledger reconcile → expired). `WATCHER_MODE=legacy`
  restores the rotating-window path (rollback hatch).
- **pipeline (exit hook)**: injection is now opt-in via
  `ARGO_EXIT_HOOK_ENABLED` (default **false**). The inbound webhook ENDPOINT
  stays registered — re-enabling is a config flip, not a deploy.
- **transpiler**: step-count guardrails — >200 steps warns (hookable),
  >500 rejects as a ValidationError (batch dispatch classifies it permanent →
  DLQ immediately instead of etcd rejecting an oversized object opaquely).

### Non-Goals
- Informer/watch machinery (rejected in review: Cloud Run multi-replica +
  apiserver memory).
- Argo Archive/offload startup hard-gate: workflows self-carry
  TTLStrategy + PodGC from the transpiler, so a controller-level default is
  not a correctness dependency.
- UNNEST bulk DB writes: the RV gate already reduces writes to
  changed-runs-only; row-at-a-time on the changed set is within budget.

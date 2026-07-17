# Proposal — CYB-3667

## Why

`backend/internal/transpiler/transpiler.go` builds the Argo `WorkflowSpec` with a
`TTLStrategy` (delete the Workflow object 30 days after completion) but **no
`PodGC`**. Consequently, a workflow's step pods are never reclaimed on
completion — they linger until the 30-day TTL cascades the Workflow deletion.

In high-throughput namespaces (`video-proc-prod`, e.g. the `cc2-mcap-slimmer`
step) this accumulates thousands of terminated pods in etcd. Field evidence at
time of writing: ~2063 Succeeded/Failed pods resident cluster-wide, well below
kube-controller-manager's `--terminated-pod-gc-threshold` default (12500), so
nothing reclaims them automatically. Terminated pod objects (plus their events)
inflate etcd and raise API-server memory pressure for no operational benefit —
DataBrew holds the durable run ledger, so finished-run pod objects are not the
source of truth.

## What Changes

### Modified Capabilities
- **runtime-os**: The transpiler SHALL set `PodGC.Strategy =
  OnWorkflowSuccess` on every generated Workflow, so a fully-successful
  workflow's step pods are deleted promptly, while a failed workflow keeps its
  pods for operator diagnostics (logs, exit codes). The Workflow object itself
  is still reclaimed by the existing `TTLStrategy`.

## Impact
- **Affected code**: `backend/internal/transpiler/transpiler.go` (WorkflowSpec
  construction), `backend/internal/transpiler/transpiler_test.go` (new
  `TestTranspilePodGC`).
- **New APIs**: none. **Migration**: none. **Schema**: none.
- **Dependencies**: none new (`wfv1.PodGC` / `wfv1.PodGCOnWorkflowSuccess`
  already available via the pinned `argo-workflows/v3` module).

## Scope
- **In scope**: default `PodGC: OnWorkflowSuccess` on all transpiled workflows.
- **Out of scope**:
  - Making the strategy configurable per-pool / per-run (can follow if needed).
  - Retroactive cleanup of the existing ~2063 terminated pods — those still age
    out via the 30-day TTL, or an operator can prune them manually
    (`kubectl delete pod --field-selector=status.phase=Succeeded -n <ns>`).
  - Any change to `TTLStrategy`.

## Success Criteria
- [ ] A new run whose workflow succeeds has its step pods removed shortly after
      completion (not resident 30 days).
- [ ] A new run whose workflow fails keeps its step pods for diagnostics.
- [ ] `spec.podGC.strategy == OnWorkflowSuccess` on a freshly transpiled/submitted
      workflow (`kubectl get workflow <uid> -o json`).
- [ ] No regression to workflow/run views (DataBrew run ledger unaffected by pod
      GC).

## Goals (SLO)
- **etcd hygiene**: terminated step pods from successful runs stop accumulating.
- **Diagnosability preserved**: failed runs retain their pods.

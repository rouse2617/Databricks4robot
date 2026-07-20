# Pipeline Spec Delta — CYB-3681

## MODIFIED Requirements

### Requirement: Active run status is refreshed by bulk LIST, not per-run GET
- **Before**: The watcher SHALL refresh active runs one GetWorkflow at a time
  inside a rotating window (≤50/tick), starving coverage at scale and issuing
  N GETs per tick.
- **After**: The watcher SHALL issue one LIST per (cluster, namespace)
  selected by `workflows.argoproj.io/completed=false`, SHALL apply listed
  workflows change-gated on resourceVersion (recalibrating every 10th scan),
  and SHALL probe snapshot-absent runs with a bounded targeted GET whose 404
  flows into orphan grading. `WATCHER_MODE=legacy` SHALL restore the window.
- **Reason**: v1.5 §05 (writeback), C10/C11 (starvation, API cost).

#### Scenario: Unchanged workflow costs nothing
- **Given** an active run whose workflow resourceVersion is unchanged
- **When** the next bulk tick runs
- **Then** no DB write occurs for that run

#### Scenario: LIST failure does not orphan a healthy cluster
- **Given** a cluster whose LIST returns 500
- **When** the tick runs
- **Then** the cluster degrades to bounded per-run refresh; no run is expired

### Requirement: Exit-notify hook is opt-in
- **Before**: Workflows SHALL carry an injected exit lifecycle hook pod that
  POSTs a status poke (one extra pod per run).
- **After**: Injection SHALL be gated by `ARGO_EXIT_HOOK_ENABLED` (default
  false); the inbound webhook endpoint SHALL remain registered.
- **Reason**: v1.5 §05 — pull replaces push; exit pods are pure cost.

#### Scenario: Default deployment injects no hook
- **Given** `ARGO_EXIT_HOOK_ENABLED` unset
- **When** a workflow is transpiled and submitted
- **Then** it contains no exit-notify template or lifecycle hook

## ADDED Requirements

### Requirement: Pipelines have step-count guardrails
- The transpiler SHALL warn above 200 steps and SHALL reject above 500 steps
  with a ValidationError (classified permanent by batch dispatch).

#### Scenario: Oversized pipeline dead-letters instead of etcd erroring
- **Given** a batch item whose pipeline expands to >500 steps
- **When** the submitter dispatches it
- **Then** validation rejects it and the item fails permanent (DLQ)

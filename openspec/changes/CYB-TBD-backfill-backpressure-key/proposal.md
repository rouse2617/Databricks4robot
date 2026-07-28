# Proposal — CYB-TBD

## Why

Batch dispatch admission backpressure (CYB-3681) reads the "active
(pending+running) workflow" count the bulk watcher observed, and defers
dispatch when a target's namespace is at/over its ceiling. That observation is
keyed by **namespace only** (`activeWFCount[namespace]`).

The watcher scans **per cluster** with a separate Argo client per cluster, but
records every namespace under the bare namespace key. When two clusters use the
same namespace name (e.g. both dispatch into `argo`), their counts overwrite
each other — last scan wins:

- cluster-A / `argo` → 200 active (saturated)
- cluster-B / `argo` → 10 active (idle)

`activeWFCount["argo"]` ends at whichever cluster's scan completed last. The
submitter then either keeps dispatching into a saturated cluster (read 10, real
200) or wrongly withholds dispatch from an idle one (read 200, real 10).

## What Changes

### Modified Capabilities

- **pipeline**: The active-workflow backpressure observation is keyed by
  `(cluster, namespace)` instead of `namespace` alone. The bulk watcher records
  each namespace's count under its cluster; the backfill submitter and the
  execution-target status read back the same `(cluster, namespace)` key.
- **pipeline**: The `backend_dispatcher_active_workflows` and
  `backend_dispatcher_backpressure_active` gauges gain a `cluster` label so
  same-named namespaces on different clusters no longer collapse in metrics.

## Impact

- **Affected code**:
  - `backend/internal/usecase/pipeline/watcher_bulk.go`
  - `backend/internal/usecase/pipeline/usecase.go`
  - `backend/internal/usecase/backfill/submitter.go`
  - `backend/internal/usecase/backfill/submitter_cluster.go`
  - `backend/internal/metrics/backend.go`
  - tests: `submitter_test.go`
- **New APIs**: None (internal signature change only).
- **Dependencies**: None. No migration, no HTTP contract change.

## Scope

- **In scope**: cluster-qualified backpressure key on both write (watcher) and
  read (submitter + target status) sides; canonicalize the default-cluster id so
  the watcher's `resolveRunClusterID` fallback (`cluster-default`) and the
  submitter's `ResolveTargetClusterID` fallback (`default`) map to one key;
  cluster label on the two backpressure gauges.
- **Out of scope**: distinguishing multiple Argo controllers (`instanceID`)
  inside a single (cluster, namespace). There is no per-target `instanceID` in
  the data model today, and the watcher LIST is already per cluster+namespace;
  multi-controller-per-namespace splitting is a separate change.

## Success Criteria

- [ ] Two clusters sharing a namespace name keep independent active-workflow
      counts; neither overwrites the other.
- [ ] The submitter's backpressure decision for a target reads the count for
      that target's own `(cluster, namespace)`.
- [ ] Default-cluster runs (empty `cluster_id`) resolve to the same key on the
      watcher write path and the submitter read path.
- [ ] Existing fail-open behavior (unknown observation → dispatch proceeds) is
      unchanged.

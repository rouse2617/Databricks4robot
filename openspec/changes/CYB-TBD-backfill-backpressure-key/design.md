# Design — CYB-TBD backfill backpressure key

## Problem

`activeWFCount map[string]int` is keyed by namespace. Writer and readers:

| Path | Location | Cluster context |
|------|----------|-----------------|
| Write | `recordActiveWorkflowCount(ns, n)` in `bulkSyncClusterRuns` | Scans per cluster, but the `cluster` var is discarded (`_ = cluster`) |
| Read (dispatch) | `deferForBackpressure` → `ActiveWorkflowCount(ns)` | Has the job's target → cluster via `ResolveTargetClusterID` |
| Read (status) | `ExecutionTargetsStatus` → `ActiveWorkflowCount(ns)` | Has the target → cluster via `ResolveTargetClusterID` |

Same namespace name on two clusters → one map slot → last writer wins.

## Decision

Key the observation by `(cluster, namespace)`. Keep the map `map[string]int`;
build the key with a small helper so the change stays minimal and the map type
is unchanged:

```go
func activeWFKey(cluster, namespace string) string {
    return backpressureClusterKey(cluster) + "\x00" + namespace
}
```

### Cluster-id canonicalization (correctness-critical)

The two cluster resolvers disagree only on the default/unresolvable case:

- `ResolveTargetClusterID` → `"default"` (used by submitter + target status)
- `resolveRunClusterID` → `"cluster-default"` (used by the watcher grouping)

For any **real** cluster both return the same `target.ClusterID`, so only the
default case can mismatch. If left as-is, default-cluster backpressure would
silently fail open (write key `cluster-default\0ns`, read key `default\0ns`).

`backpressureClusterKey` collapses `""`, `"default"`, and `"cluster-default"`
to a single `"default"` token. It is applied inside both
`recordActiveWorkflowCount` and `ActiveWorkflowCount`, so any caller — whichever
resolver produced the cluster id — lands on the same key. `ResolveTargetClusterID`
never emits `cluster-default`, so read-side values are already canonical; the
helper defends the write side and future callers.

### Metrics

`DispatcherActiveWorkflows` and `DispatcherBackpressureActive` gain a `cluster`
label. Watcher publishes the canonical cluster; the submitter publishes the
`ResolveTargetClusterID` value (already canonical), so labels agree per cluster.

## Call flow after the change

```
watcher: bulkSyncActiveRuns groups by resolveRunClusterID
      → bulkSyncClusterRuns(cluster, ...)
        → recordActiveWorkflowCount(cluster, ns, n)   # key = default|<clusterID> \0 ns

submitter: deferForBackpressure(job)
      → cluster = ResolveTargetClusterID(targetID)
      → ns, max = ResolveTargetBackpressure(targetID)
      → ActiveWorkflowCount(cluster, ns)              # same key
```

## Alternatives considered

- **Unify the two resolvers' fallbacks**: `resolveRunClusterID` feeds Argo
  client resolution across the watcher/webhook; changing its fallback risks
  unrelated regressions. Canonicalizing only at the backpressure boundary is
  narrower and safer.
- **instanceID in the key**: no per-target `instanceID` exists today, and the
  LIST is already per (cluster, namespace). Deferred.

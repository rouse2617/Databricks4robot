# Spec Delta — multi-cluster runtime routing

## ADDED

### `internal/k8s.ClientFactory`

An interface abstracting "which K8s client for this cluster." Implemented by `dbClientFactory` which reads the `clusters` table.

```go
type ClientFactory interface {
    ForCluster(ctx context.Context, clusterID string) (kubernetes.Interface, error)
    DynamicForCluster(ctx context.Context, clusterID string) (dynamic.Interface, error)
    ForTarget(ctx context.Context, t *models.ExecutionTarget) (kubernetes.Interface, error)
    Invalidate(clusterID string)
}
```

Contract:

- `ForCluster` returns `ErrClusterNotFound` if the cluster row is missing (or soft-deleted). Consumers surface this as a 404 to the admin API and as a run failure to the deploy usecase.
- `ForCluster` returns `ErrClusterMisconfigured` for non-default clusters with empty `k8s_audience` (metadata-token auth requires an audience).
- Default-cluster compat: if `cluster.K8sAPIEndpoint == ""`, the factory falls back to the existing env-derived `buildConfig("")` path — behavior is byte-identical to the pre-3486 singleton for the `cluster-default` row.
- The factory caches `(kubernetes.Interface, dynamic.Interface)` per `clusterID` for 60 seconds (`WithTTL` overrides). Cache misses fetch the row + build config + build both clients.
- `Invalidate(id)` drops the cached entry so the next call reads a fresh row (called by admin cluster CRUD handlers after Create/Update/SoftDelete).
- `ForTarget` treats `target.ClusterID == ""` as `cluster-default` (bridges pre-3425 rows that never carried the FK).

### `internal/argo.ClientFactory`

Parallel interface for Argo Workflow clients:

```go
type ClientFactory interface {
    ForCluster(ctx context.Context, clusterID string) (WorkflowClient, error)
    ForTarget(ctx context.Context, t *models.ExecutionTarget) (WorkflowClient, error)
    Invalidate(clusterID string)
}
```

- Non-empty `cluster.ArgoServerURL` overrides the env `ARGO_SERVER_URL`; token and TLS still come from env until per-cluster secret storage is justified.
- Cache lifetime + `Invalidate` semantics mirror `k8s.ClientFactory`.

## What stays UNCHANGED

- `k8s.NewClientset("")` / `k8s.NewDynamicClient("")` continue to exist and continue to return the env-derived singleton client for the default cluster. All existing consumers (elastic_quota, resource_quota, run_watcher, deploy usecase) are untouched in PR 4a — they still call the singletons.
- No migration. No new column. No behavior change for pipelines already running on `cluster-default`.

## What follows in later PRs (not part of this delta)

- PR 4b flips elastic_quota + resource_quota handlers to `factory.DynamicForCluster / ForCluster`.
- PR 4c flips Deploy usecase + argo submit to `argoFactory.ForTarget(target)`.
- PR 4d makes `run_watcher` spawn one goroutine per active cluster.

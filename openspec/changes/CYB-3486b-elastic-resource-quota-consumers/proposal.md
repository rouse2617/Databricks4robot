# CYB-3486 PR 4b — elastic_quota + resource_quota consumers switch to factory

## Why

PR 4a landed `k8s.ClientFactory` + `argo.ClientFactory` as no-op skeletons wired into `admin.ClusterHandler`. No runtime consumer read from them yet, so `GET /api/v1/elastic-quotas` and `GET /api/v1/resource-quotas` still hit the env-based singleton (`k8s.NewDynamicClient("")` / `k8s.NewClientset("")`). Both endpoints ignore `target.cluster_id` entirely.

Multi-cluster is a no-go until at least one endpoint actually routes.

## What Changes

- `workflow.Handler` gets a `k8sFactory k8s.ClientFactory` field + `SetK8sFactory` setter.
- `listElasticQuotas` closure signature grows `(ctx, factory, clusterID)`; production path uses `factory.DynamicForCluster(clusterID)`. Nil-factory branch falls back to the env singleton so tests without a wired factory keep working.
- `newK8sClientset` grows the same signature; used by `ListResourceQuotas`.
- Both handlers read `?clusterId=` (default `cluster-default`).
- `setupInfra` builds `clusterRepo` + `k8sFactory` + `argoFactory` once and stores on `infra`; `server.go` reuses `inf.clusterRepo` / `inf.k8sFactory` / `inf.argoFactory` (PR 4a redundant construction removed); `setupCore` calls `workflowHandler.SetK8sFactory(inf.k8sFactory)`.

## What NOT in this PR

- Deploy usecase + argo submit — that's PR 4c.
- `run_watcher` per-cluster goroutines — PR 4d.
- Frontend UI: the ElasticQuota panel doesn't ship a cluster picker yet; when it does, it'll pass `?clusterId=`.
- Any change to the `NewClientset("")` call in `core.go` line 146 (RuntimeConfigStore + NodeInstanceResolver). That's a cluster-default-only surface; leaving it explicit as env singleton until we have a real multi-cluster use case for it.

## Compat matrix

| Client state | Behavior |
|--------------|----------|
| `k8sFactory == nil` (tests / no PG) | Fall back to `k8s.NewDynamicClient("") / NewClientset("")` — byte-identical to pre-3486 |
| `k8sFactory != nil`, no `?clusterId=` | Route to `cluster-default` → env-based `buildConfig("")` via factory (same underlying config as singleton) |
| `k8sFactory != nil`, `?clusterId=delivery` | Route to delivery-clust config from clusters row |

## Verify plan

- [x] `go build ./... && go test ./...` clean including new `TestListElasticQuotas_ClusterIDParam`
- [ ] CI green on PR
- [ ] After merge to dev: `GET /api/v1/elastic-quotas` (no clusterId) → same 3 quotas as PR 4a smoke
- [ ] After merge to dev: `GET /api/v1/elastic-quotas?clusterId=cluster-default` → identical result
- [ ] After merge to dev: `GET /api/v1/resource-quotas` → non-empty result unchanged

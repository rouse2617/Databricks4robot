# Tasks — CYB-3486 PR 4b: elastic_quota + resource_quota consumers via factory

## Landing PR (this iteration)

- [x] Hoist cluster repo + k8s factory + argo factory to `infra` (once per process)
- [x] `server.go` reuses `inf.clusterRepo` / `inf.k8sFactory` / `inf.argoFactory`; PR 4a in-place construction removed
- [x] `workflow.Handler` gets `k8sFactory` field + `SetK8sFactory` setter
- [x] `setupCore` calls `workflowHandler.SetK8sFactory(inf.k8sFactory)`
- [x] `listElasticQuotas` signature `(ctx, factory, clusterID)`; nil-factory fallback to env singleton
- [x] `newK8sClientset` signature `(ctx, factory, clusterID)`; nil-factory fallback
- [x] `ListElasticQuotas` / `ListResourceQuotas` read `?clusterId=` (default `cluster-default`)
- [x] Existing 4 elastic_quota tests updated to new stub signature; still green
- [x] New `TestListElasticQuotas_ClusterIDParam` verifies query param → downstream clusterID
- [x] `go test ./...` full suite green

## Verify plan (dev)

- [ ] Merge to dev → GHA deploy-dev green
- [ ] `GET /api/v1/elastic-quotas` (no query) matches pre-4b response (3 delivery quotas)
- [ ] `GET /api/v1/elastic-quotas?clusterId=cluster-default` matches
- [ ] `GET /api/v1/resource-quotas` (no query) matches pre-4b response

## Follow-ups

- **PR 4c** — Deploy usecase + argo submit routed via `argoFactory.ForTarget(target)`
- **PR 4d** — `run_watcher` per-cluster goroutine
- (opt) frontend: cluster picker on ElasticQuota panel passes `?clusterId=`

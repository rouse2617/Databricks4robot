# Tasks — CYB-3486 runtime client factory (PR 4a 骨架)

## PR 4a (this iteration): interfaces + skeleton + tests, no consumers

- [x] `backend/internal/k8s/factory.go` — `ClientFactory` interface + `dbClientFactory` impl
  - [x] `ForCluster` / `DynamicForCluster` / `ForTarget` / `Invalidate` methods
  - [x] 60s TTL cache, `WithTTL` option
  - [x] Default-cluster compat: empty `K8sAPIEndpoint` → fall back to env
  - [x] Non-default cluster requires `K8sAudience` (returns `ErrClusterMisconfigured`)
- [x] `backend/internal/k8s/base64.go` — `base64DecodeIfBase64` helper for CA data
- [x] `backend/internal/argo/factory.go` — argo `ClientFactory` with `WithEnvFallback` option
- [x] `backend/internal/k8s/factory_test.go` — 10 tests (NotFound / DefaultUsesEnv / RequiresAudience / CacheHit / CacheExpiry / Invalidate / ForTarget / FailureIsolation)
- [x] `backend/internal/argo/factory_test.go` — 7 tests (parallel coverage on argo side)
- [x] `go build ./... && go test ./internal/k8s/... ./internal/argo/...` clean

## PR 4a wiring (this iteration, additive)

- [ ] `backend/cmd/server/server.go` — construct `k8sFactory` + `argoFactory` alongside existing singletons; hold on `Server` struct; not yet consumed
- [ ] Wire cluster admin CRUD handlers to call `factory.Invalidate(clusterID)` after Create/Update/SoftDelete (Cluster row edits pick up without waiting for TTL)

## Follow-ups (separate PRs, NOT in 4a)

- [ ] **PR 4b** — replace `elastic_quota.go` + `resource_quota.go` singletons with `factory.DynamicForCluster / ForCluster`
- [ ] **PR 4c** — replace Deploy usecase + argo submit with `argoFactory.ForTarget(target)`
- [ ] **PR 4d** — replace `run_watcher` single-goroutine with per-cluster goroutines
- [ ] **PR 4e (opt)** — per-cluster circuit breaker + Feishu cluster tag in alerts

## Verify plan (this iteration)

- [x] Unit tests green (17 tests across k8s + argo)
- [ ] Backend build clean on dev CI
- [ ] Backend deploys to dev without runtime regression (factory unused = zero behavior change)
- [ ] After deploy: existing pipelines on `cluster-default` still run (single-cluster smoke)

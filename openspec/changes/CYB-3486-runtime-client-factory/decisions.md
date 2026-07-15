# Decisions — CYB-3486 runtime client factory

## D1. Introduce a `ClientFactory` interface instead of adding a `clusterID string` parameter to every existing K8s helper

- **Chosen**: extract `ForCluster(ctx, id)` / `DynamicForCluster` / `ForTarget(target)` on an interface implemented by `dbClientFactory`.
- **Rejected**: thread `clusterID` through every existing `k8s.NewClientset(namespace)` call.
- **Why**: the multi-cluster switch is orthogonal to what consumers ask for (they want "the k8s client for this target"). An interface preserves the current call sites' shape and future-proofs the seam for a Karmada-backed impl (Stage 2) that will not read `clusters` at all.

## D2. TTL cache (60s) instead of per-request DB read or per-startup singleton

- **Chosen**: 60s TTL cache keyed by `clusterID`, invalidated eagerly on admin CRUD (`factory.Invalidate`).
- **Rejected A**: no cache — every deploy hits the `clusters` table for endpoint/CA/audience.
- **Rejected B**: startup-only singleton — cluster edits require a backend restart.
- **Why**: cluster row edits are rare (admin flow); 60s ceiling on staleness is acceptable and eager invalidation keeps admin UX snappy. Cache also amortizes rest.Config + clientset construction across bursts (e.g. 40 concurrent deploys hitting the same cluster).

## D3. Default cluster falls back to env vars instead of requiring backfill

- **Chosen**: if `clusters.k8s_api_endpoint == ""`, `buildConfigFromCluster` calls the existing env-derived `buildConfig("")` path (same code the pre-3486 singleton used).
- **Rejected**: run a migration to copy `K8S_API_ENDPOINT` / `K8S_AUDIENCE` / etc into the default-cluster row and require every cluster row to be self-contained.
- **Why**: keeps PR 4a a pure additive change. The env vars are the source of truth today (they drive the Cloud Run env of every backend replica); duplicating them into a DB row invites drift ("who updated it last, DB or env?"). When a second cluster shows up, that cluster row IS self-contained (endpoint + audience + CA populated); only the historical default sits on the env fallback.

## D4. WIF metadata token only — no per-cluster bearer / kubeconfig field

- **Chosen**: non-default cluster rows must carry `k8s_audience`. `buildConfigFromCluster` builds a metadata-server token source scoped to that audience. Kubeconfig-file / static-bearer paths are not exposed per cluster.
- **Rejected**: add `k8s_bearer_token` / `k8s_kubeconfig` columns to `clusters`.
- **Why**: defense against key sprawl. Every cluster we manage is a GKE cluster reachable from the backend's Cloud Run service account via WIF; extra auth modes add rotation + secret-storage surface with no current customer. Add columns when there's a concrete use case (e.g. an on-prem k3s that can't federate).

## D5. Argo factory caches the `WorkflowClient` too, even though it's cheap

- **Chosen**: argo `dbClientFactory` uses the same 60s TTL + `Invalidate` shape as k8s.
- **Rejected**: build a fresh argo client on every call (they're stateless HTTP clients).
- **Why**: uniform interface across k8s + argo is a small win for the callers, and the mechanism is cheap. Also it lets admin cluster CRUD invalidate both factories via one `.Invalidate(id)` on each.

## D6. Ship 4a with the factory constructed in `server.go` but NOT wired into any consumer

- **Chosen**: `Server` struct holds `k8sFactory` + `argoFactory`; nothing calls them yet. Zero runtime behavior change on dev.
- **Rejected**: bundle 4a + 4b (replace elastic_quota handler) into a single PR.
- **Why**: user rule "一功能一PR / 开发完成后充分测试验证再提PR" plus explicit compaction to "拒绝新基础设施/依赖,优先自包含小方案". Skeleton lands with tests, next PR flips one handler to the factory in isolation, so any regression is bisectable.

## D7. Pre-CYB-3486 targets with empty `cluster_id` resolve to `cluster-default`

- **Chosen**: `ForTarget` treats empty `t.ClusterID` as `cluster-default`.
- **Rejected**: hard-error on empty cluster ID.
- **Why**: the FK column was added by PR #397/398; PoolManager rows created before then have empty `cluster_id` in-memory until they're re-fetched (backend save layer defaults to `cluster-default`). A silent fallback keeps SDK-created targets working through the rollout.

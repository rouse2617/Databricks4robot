# Decisions — CYB-3073

## 2026-07-06 — Bug 1 fix location: ComputeRunCost + the list SUM subquery
- `ComputeRunCost` (cost.go) and the run-list `SUM(estimated_cost_usd)` subquery (pipeline_repo.go:741/769) both summed ALL nodes, including the DAG rollup → the batch list total ($0.39) over-counted. Both now sum leaf Pod nodes only (`n.type='Pod'`, tolerating blank-typed rows with a pod name).
- The asset-node summary already excluded DAG/Steps via `isWorkflowControlRunNode`, so it was correct and is left as-is (the fix aligns the run total with it).

## 2026-07-06 — Bug 2: get-nodes RBAC already satisfied (no new grant)
- The backend's K8s identity is the KSA `cyber-databrew-backend-argo`, which already has cluster-scoped `nodes get/list/watch` via the existing `cyber-databrew-resource-capacity-reader` ClusterRole (`kubectl auth can-i list nodes --as=system:serviceaccount:cyber-databrew-dev:cyber-databrew-backend-argo` → yes). No new RBAC needed.
- Caveat to confirm on dev: the backend's Cloud Run clientset authenticates via WI metadata token; if at runtime it presents as the GCP SA (which lacks node read) rather than the KSA, node resolution silently misses and cost falls back to the default GPU rate (best-effort, no error). Verify CPU costs actually drop on dev; if not, revisit the K8s auth identity.

## 2026-07-06 — Best-effort, backward compatible
- `NodeInstanceResolver` returns a zero profile on miss/error/nil; `enrichResourcesDurationWithNode` then leaves the node unchanged and pricing keeps the prior default. No crash, no migration, no HTTP change. Values decrease (more accurate); response shapes unchanged.

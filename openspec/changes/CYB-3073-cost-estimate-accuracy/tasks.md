# Tasks — CYB-3073

Branch: `fix/CYB-3073-cost-estimate-accuracy` (from `origin/dev`). Tier **M**.

## Part 1 — Bug 1: exclude DAG rollup (no RBAC; ship first)
- [ ] [backend] `cost.go` `ComputeRunCost`: sum only leaf nodes (`n.Type == "Pod"`), skip DAG/Steps/StepGroup aggregate nodes. (Scenario: total equals sum of steps)
- [ ] [backend] Audit sibling cost aggregations (batch node-summary, `/runs/{id}/cost-summary`, asset-node projection) for the same over-count; make them Pod-only/consistent.
- [ ] [backend] unit test: run with DAG node + 2 Pod nodes → total = sum of Pods only.

## Part 2 — Bug 2: price by real node type (needs `get nodes` RBAC)
- [ ] [k8s/RBAC] Grant backend SA `get/list nodes` (cluster-scoped ClusterRole+binding). **Needs approval.**
- [ ] [backend] `k8s`: `NodeInstanceProfileResolver` — `Resolve(ctx, nodeName) (instanceType, accelerator, provisioning)` from node labels (`node.kubernetes.io/instance-type`, `cloud.google.com/gke-accelerator`, `cloud.google.com/gke-provisioning`), with an in-memory cache (nodes are stable). Miss/err → empty (caller falls back).
- [ ] [backend] Node cost derivation (`usecase.go` ~1867): before `resourcesDurationToCost`, enrich `ResourcesDuration` with resolved instance_type/gpu_type/provisioning by `HostNodeName`. Empty resolution keeps the current default (Scenario: lookup failure degrades gracefully).
- [ ] [backend] `cmd/server`: construct resolver from the existing k8s clientset; inject into the pipeline usecase (nil-safe when clientset disabled).
- [ ] [backend] unit test: node with CPU instance labels → CPU rate; unresolved node → default.

## Verify (dev, after deploy)
- [ ] Re-open batch_2c1accbf run: total now equals sum of visible step costs (Bug 1).
- [ ] A CPU step's 估算成本 drops to its CPU-rate value (Bug 2).
- [ ] Grace/K8s lookup failure or no RBAC: cost still renders (fallback), no 500.

## API contract sync
- N/A — no HTTP API change; response field shapes unchanged (values differ). No migration.

## Verification tiers
- [ ] Tier M: `make fmt && make vet`; `go test ./internal/usecase/pipeline/... ./internal/k8s/...`.

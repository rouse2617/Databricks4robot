# Tasks — CYB-3486 workflow handler cluster-aware

## Implementation

- [x] Add `argoFactory`, `targetRepo`, `targetClusterCache` fields to
      `workflow.Handler`; add `SetArgoFactory` / `SetExecutionTargetRepo`.
- [x] `cluster_routing.go`: `resolveRunClusterID`, `argoClientForCluster`,
      `argoClientForRun`, `namespaceFromRun`, `namespaceForRequest`,
      `resolveWorkflowRouting`, `podClientForCluster`.
- [x] Route `GetWorkflow`, `GetWorkflowLogs`, retry/resubmit/suspend/stop/
      resume/terminate/delete, `ListWorkflows` (handler.go) by cluster.
- [x] Route `StreamWorkflowLogs` (logs_sse.go) by cluster.
- [x] Route `GetNodePodDiagnostics` (pod.go) argo + per-cluster pod client.
- [x] Route `CreateTerminalSession` (terminal.go) GetWorkflow by cluster;
      note exec attach limitation.
- [x] Wire `SetArgoFactory` + `SetExecutionTargetRepo` in `cmd/server/core.go`.

## Tests

- [x] `cluster_routing_test.go`: `resolveRunClusterID` object-first,
      target-lookup + cache, fallbacks.
- [x] `argoClientForRun` factory-routes / nil→singleton / error→fallback.
- [x] HTTP-level: non-default-cluster run routes GetWorkflow + logs to the
      matching client; factory-nil falls back to singleton.

## Verification

- [ ] `go build ./...` + `go test ./...` green.
- [ ] Deploy dev; delivery-clust run Log tab shows pod logs (no argo-server
      not found).

## API contract sync

- Not applicable — no route/handler/request/response/status-code change. This
  is an internal client-routing fix. (AI-RULES API contract sync: none of rows
  1–8 triggered.)

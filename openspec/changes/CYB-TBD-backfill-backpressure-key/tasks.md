# Tasks — CYB-TBD

## Context files

- `backend/internal/usecase/pipeline/watcher_bulk.go` — watcher write path and `ActiveWorkflowCount` reader.
- `backend/internal/usecase/pipeline/usecase.go` — `ResolveTargetClusterID`, `ResolveTargetBackpressure`, `ExecutionTargetsStatus`.
- `backend/internal/usecase/backfill/submitter.go` — `subtaskDeployer` interface.
- `backend/internal/usecase/backfill/submitter_cluster.go` — `deferForBackpressure`.
- `backend/internal/metrics/backend.go` — backpressure gauges.
- `backend/internal/usecase/backfill/submitter_test.go` — fake deployer + backpressure tests.

## Implementation

- [x] [backend] Add `normalizeBackpressureCluster` + `backpressureKey` `(cluster, namespace)` composite key helper in the pipeline package.
- [x] [backend] `recordActiveWorkflowCount(cluster, namespace, n)` — key by composite, publish `cluster`+`namespace` gauge labels.
- [x] [backend] `ActiveWorkflowCount(cluster, namespace)` — read composite key.
- [x] [backend] Thread `cluster` into `bulkSyncClusterRuns` and record per `(cluster, namespace)`.
- [x] [backend] `ExecutionTargetsStatus` — resolve each target's cluster and read `(cluster, namespace)`.
- [x] [backend] `subtaskDeployer.ActiveWorkflowCount(cluster, namespace)` interface update.
- [x] [backend] `deferForBackpressure` — resolve cluster via `ResolveTargetClusterID`, read `(cluster, namespace)`, label metric with cluster.
- [x] [backend] Update fake deployer + add a same-namespace-two-clusters regression test.

## API Contract Sync

- [x] Not applicable: no HTTP route/request/response/status/SDK change. Prometheus gauge gains a `cluster` label (internal observability only).

## Verification

- [x] [backend] `go build ./...` — passed (CGO disabled on this box).
- [x] [backend] `go vet` on touched packages — passed.
- [x] [backend] `go test ./internal/usecase/backfill/... ./internal/usecase/pipeline/...` — passed, incl. new same-namespace cross-cluster regression.

## Deploy Verification

- [x] User requested no manual dev deploy; rely on PR CI/CD. Recorded in decisions.md.
- [x] Provide local test evidence in the PR.

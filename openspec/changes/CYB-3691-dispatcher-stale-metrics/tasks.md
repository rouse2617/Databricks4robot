# Tasks — CYB-3691

## Context files

- `backend/internal/metrics/backend.go` — metrics registry
- `backend/internal/repository/backfill_repository.go` — interface to extend
- `backend/internal/postgres/backfill_repo.go` — SQL query + repo impl
- `backend/internal/usecase/backfill/usecase.go` — reconcileActiveJobs wiring
- `deploy/k8s/monitoring/dashboards/backend-observability.json` — Grafana panel
- `deploy/k8s/monitoring/prometheus-configmap.yaml` — GMP config

## Implementation

- [x] [backend] Add `DispatcherStaleItems` gauge to `backend/internal/metrics/backend.go`
- [x] [backend] Add `CountStaleBackfillItems(ctx) (int, error)` to `BackfillRepository` interface
- [x] [backend] Implement `CountStaleBackfillItems` in `BackfillRepo` (Postgres SQL)
- [x] [backend] Wire gauge into `reconcileActiveJobs` — set at cycle start (pre-sync gap)

## API contract sync

No new/changed HTTP API — metrics only.

## Local verification

- [x] [backend] `go build ./...` — compile check
- [x] [backend] `go test ./...` — all tests pass (including all mock updates)

## Dashboard

- [x] [grafana] Add stale-items panel to `deploy/k8s/monitoring/dashboards/backend-observability.json`

## Deploy verification (before commit)

- [ ] Build + push with **git SHA tag** and `cloudrun-dev-latest`
- [ ] Deploy dev using `IMAGE=…:<sha>`
- [ ] Check `/metrics` endpoint for `backend_dispatcher_stale_items`
- [ ] Check Grafana dashboard for new panel

## PR

- [ ] PR template filled; Linear CYB-3691 linked

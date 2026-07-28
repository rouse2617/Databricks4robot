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
- [x] [backend] Push the gauge to GCP Cloud Monitoring via `monWriter.WriteInt64Metric(ctx, "dispatcher/stale_items", n)` — the writer was wired but never called (see decisions.md 2026-07-27)

## API contract sync

No new/changed HTTP API — metrics only.

## Local verification

- [x] [backend] `go build ./...` — compile check
- [x] [backend] `go test ./...` — all tests pass (including all mock updates)

## Dashboard

- [x] [grafana] Add stale-items panel to `deploy/k8s/monitoring/dashboards/backend-observability.json`

## Deploy verification (dev via CICD)

- [x] Local `go build ./...` + `go test ./internal/usecase/backfill/...` green
- [ ] Merge to `dev` → `deploy-dev.yml` builds + deploys backend to Cloud Run dev
- [ ] Confirm `custom.googleapis.com/dispatcher/stale_items` receives data in GCP Cloud Monitoring (Metrics Explorer)
- [ ] "Dispatcher Metrics (Batch / Submitter)" GCP dashboard shows the gauge (was empty before — writer never called)

## PR

- [x] PR template filled; Linear CYB-3691 linked

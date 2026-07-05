# Proposal — CYB-3058

## Why

Run status is synced by polling Argo every 3s, capped at `active_scan_limit=100`; at batch scale (e.g. 3× 1000-asset backfills) this causes stale/incomplete status and constant load on the Argo server + DB. The `pipeline_run_events` ledger is already idempotent, so the status source can switch from poll to push with a localized change.

## What Changes

### New Capabilities
- **runtime-os**: An Argo workflow can push its own status to DataBrew — a transpiled workflow carries an exit/lifecycle hook that notifies DataBrew when it reaches a terminal phase, so terminal status lands within seconds instead of waiting for the next poll cycle.

### Modified Capabilities
- **runtime-os**: The polling watcher is demoted from primary status source to a low-frequency reconcile backstop; poll interval and scan limit become configurable.
- **runtime-os**: Observed run status is monotonic — a stale or out-of-order status delivery cannot regress a run that already advanced to a later/terminal phase.

## Impact
- **Affected code**: `backend/internal/transpiler` (inject hook), `backend/internal/usecase/pipeline` (apply pushed status, monotonicity guard, watcher interval), `backend/internal/handlers` (new webhook), `backend/routes`, `backend/internal/config`, `backend/cmd/server`
- **New APIs**: `POST /api/v1/runs/{id}/status-webhook` (idempotent, authenticated) — receives Argo phase notifications
- **Dependencies**: none new (reuses existing `argo-workflows` Go types, existing auth middleware, existing `pipeline_run_events` sink)

## Scope
- **In scope**: transpile-time hook injection; idempotent authenticated webhook that writes `pipeline_run_events` and updates run (and batch item) status; phase-monotonicity guard; configurable watcher interval/scan-limit with a lower default.
- **Out of scope**: full Argo Events (EventSource + Sensor) deployment; per-node-phase push granularity beyond what the hook emits; the Argo-native concurrency refactor (parallelism/quota/semaphore, removing the backfill worker pool / SKIP LOCKED / reaper) — tracked separately.

## Success Criteria
- [ ] A finished Argo workflow's terminal status appears in DataBrew within seconds, without depending on the 3s poll.
- [ ] Duplicate webhook deliveries produce no duplicate events (idempotent) and out-of-order deliveries never regress a run's status.
- [ ] Watcher poll interval is configurable and defaults lower (30–60s); dropped push events are still eventually reconciled by the poll backstop + `backfillRunStatus`/`ledger_state`.
- [ ] Unauthenticated or malformed webhook calls are rejected without mutating state.

## Goals (SLO)
- **Latency**: terminal status visible in DataBrew p95 < 10s after workflow completion (push path).
- **Load**: steady-state Argo `GetWorkflow` poll volume reduced by ≥ 80% vs the 3s baseline at equal run count.
- **Correctness**: zero duplicate `pipeline_run_events` rows and zero status regressions under duplicate/out-of-order delivery.

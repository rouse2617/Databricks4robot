# Tasks — CYB-3744 定时任务

## Context files

- `backend/internal/grace/client.go`, `grace/sync.go` — HTTP client + existing sync loop (extend)
- `backend/cmd/server/core.go:203` — where grace client/syncer is wired
- `backend/internal/usecase/backfill/*` — batch creation (reuse)
- `backend/internal/usecase/pipeline/usecase.go` — StartJobReconciler ticker pattern (mirror)
- `backend/internal/models/dispatcher_config.go`, `postgres/dispatcher_config_repo.go`, `handlers/backfill/dispatcher_handler.go` — config table + admin API + UI pattern to mirror
- `Frontend/src/pages/PipelinePage.tsx` (tabs), `ExecutionRecordsPanel.tsx` — where the tab goes
- `services/grace-sync/main.go` — the exact Grace query to replicate (retire after)
- `api/openapi.yaml`, `docs/review/api-guide.md`, `sdk/` — API contract sync

## Migration (approved, first)

- [ ] Migration `backend/migrations/<ts>_scheduled_tasks.sql` — `scheduled_tasks` table (see design.md schema); hand-written SQL; `make db-migrate-hash` + `atlas migrate validate`
- [ ] Apply to dev via `scripts/apply-migration-dev.sh` before backend deploy

## Backend — source abstraction

- [ ] `AssetSource` interface + `Window` type
- [ ] `restSource` impl: config-driven REST-JSON fetch (auth via secret_ref, query+window templating, page/size paging, id_path extraction); reuse `grace.Client` HTTP handling
- [ ] Secret Manager deref for `secret_ref` (no plaintext); unit tests with mocked secret + HTTP

## Backend — scheduler + rules

- [ ] `scheduled_tasks` repo (Postgres CRUD + claim/cursor update)
- [ ] Scheduled-task usecase: `resolveWindow` (incremental/rolling/range/ids), run-once (Fetch → build batch via backfill), cursor advance, last_run observation, failure = record + no cursor advance
- [ ] Scheduler loop (ticker) with single-runner claim (conditional UPDATE / advisory lock); wire in `core.go`
- [ ] Reuse backfill batch creation (template+version, target, scheduling, asset-ids, name)

## Backend — alerting (reuse existing 飞书)

- [ ] Reuse existing 飞书 sender (mirror `services/grace-sync/main.go:notifyFeishu` / CYB-3071 batch notify); do NOT add a new channel
- [ ] Notify on: rule failure (source or batch), rule stuck (no successful run for > threshold)
- [ ] Empty-fetch notification behind a per-rule toggle (default off)
- [ ] (optional, follow-up) Prometheus gauge for failure/stuck counts, mirror CYB-3691

## Backend — API (contract sync, same PR)

- [ ] Handlers: `GET/POST/PUT/DELETE /scheduled-tasks`, `POST /scheduled-tasks/:id/{pause,resume,run-now}`
- [ ] `api/openapi.yaml` + `docs/review/api-guide.md` (curl, headers, error paths)
- [ ] SDK client + unit tests (if public surface)
- [ ] `scripts/smoke-scheduled-tasks-dev.sh` (happy + error path)

## Frontend

- [ ] New tab `schedules` / 「定时任务」in `PipelinePage.tsx`
- [ ] Rules list (name/pipeline/source/mode/interval/status/last-run/actions)
- [ ] Create/edit drawer: reuse template picker + target/scheduling picker; REST-source form; trigger-mode selector; enable/pause; 立即运行
- [ ] API client + types aligned with OpenAPI

## Verification (Tier L — cross-module + API + UI)

- [ ] `make fmt && make vet && go test ./...`
- [ ] `npm run lint && npm run build && npm run test -- --run` (touched)
- [ ] Deploy dev; smoke the API; **Chrome DevTools MCP** on the 定时任务 tab (create rule, run-now, batch appears in 执行记录)

## Retirement (after in-app verified in prod)

- [ ] prod: create equivalent Grace REST rule; run **in parallel** with external grace-sync; reconcile output
- [ ] pause Cloud Scheduler (`grace-sync-prod`/`grace-sync-daily`) → observe → delete Cloud Scheduler + Cloud Run Jobs (`grace-sync`/`grace-sync-prod`)
- [ ] keep `services/grace-sync/deploy.sh` (rollback recreate); note in decisions.md

## PR

- [ ] Split if >200 lines (backend source+scheduler / API / frontend). Template filled; Linear CYB-3744; contract rows checked

# Tasks — CYB-3058

Branch: `feat/CYB-3058-argo-push-status` (from `origin/dev`).
Verification tier for this change: **L** (API contract sync + cross-module).

## 1. Config & wiring
- [ ] [backend] Add `ArgoRunWebhookURL`, `ArgoRunWebhookToken` env to `config.go` (struct 137-139, env read 238-242 pattern).
- [ ] [backend] Add `PipelineRunWatcherIntervalSec` (default 30–60s) and `PipelineRunWatcherScanLimit` env to `config.go`.
- [ ] [backend] Add `SetArgoRunWebhook(url, token)` setter + getter + Usecase fields in `usecase/pipeline/usecase.go` (mirror `SetArgoWorkflowTTLSecondsAfterCompletion` 225-233).
- [ ] [backend] In `cmd/server/core.go`: call `SetArgoRunWebhook` from `cfg`; replace hardcoded `StartRunEventWatcher(ctx, 3*time.Second, 100)` (line 147) with config-driven interval + scan limit.

## 2. Transpiler — inject exit hook (Scenario: Workflow success/failure is pushed)
- [ ] [backend] Add `ExitHookURL`, `ExitHookToken` fields to `transpiler.Options` (transpiler.go 38-52).
- [ ] [backend] In `Transpile` (transpiler.go 80-107): when `ExitHookURL != ""`, append an HTTP `wfv1.Template` (POST, auth header, JSON body with `{{workflow.name}}`/`{{workflow.status}}`/`{{workflow.uid}}`/message/finish time) and set `Spec.Hooks[wfv1.ExitLifecycleEvent] = {Template: <name>}`.
- [ ] [backend] Pass `ExitHookURL/Token` into `wfOpts` in `Deploy` (usecase.go 3092-3103).
- [ ] [backend] Unit test in `transpiler_test.go`: hook present when URL set; absent when URL empty (Scenario: Notification delivery fails without blocking — hook is best-effort, absence path).

## 3. Webhook endpoint (Scenario: duplicate dedup / unauthenticated rejected / unknown workflow rejected)
- [ ] [backend] Add `ArgoWebhookAuth(token)` middleware (constant-time compare, mirror `ingest_auth.go` 17/80). Rejects missing/invalid token → 401 (Scenario: Unauthenticated call is rejected).
- [ ] [backend] Add `HandleRunWebhook` to pipeline handler (parse `{workflowName, namespace, uid, phase, message, finishedAt}`).
- [ ] [backend] Register `POST /api/v1/pipeline-runs/webhook` under a dedicated group with `ArgoWebhookAuth` in `routes.go` (near 198/213).
- [ ] [backend] Unknown workflow name → 404, no mutation (Scenario: Notification for an unknown workflow is rejected safely).

## 4. Poke → authoritative refresh + monotonicity guard (Scenario: dedup / stale-after-terminal ignored / forward progress applied)
- [ ] [backend] Add `RefreshRunFromWorkflowByName(ctx, workflowName, uid)` in `usecase/pipeline/usecase.go`: `runRepo.FindByWorkflowName` (NOT `GetRunByWorkflowName`, which re-enriches) + UID cross-check → single `GetWorkflow` → existing `applyWorkflowToRun` (captures status + nodes + events + backfill-item cascade). Hook payload phase is a trigger only; DataBrew pulls truth.
- [ ] [backend] Add monotonicity guard in `persistRunObservation` (line ~1973): if `existing` is terminal (`!isActiveDeploymentStatus`), reject an incoming active status (Scenario: Stale non-terminal delivery after terminal is ignored).
- [ ] [backend] Ensure run status is derived from business DAG nodes and the notify hook node is excluded from status derivation (`deriveTerminalRunFromWorkflowNodes` / node listing) so a failed notify node cannot mark the run failed.
- [ ] [backend] Confirm forward progress still applies (running→terminal) (Scenario: Forward progress is applied).
- [ ] [backend] Confirm `backfill_items` auto-updates via existing `syncBackfillItemStatusFromRun` cascade (no extra webhook code).
- [ ] [backend] Update ALL test mocks implementing the run repo interface if signatures change (`usecase_test.go` `mockRunRepo` etc.) — same commit.
- [ ] [backend] Unit tests: idempotent re-delivery writes no second event; stale active-after-terminal is ignored; duplicate produces no status change; unknown/mismatched UID rejected.

## 5. Watcher backstop (Scenario: Poll interval is configurable / Backstop reconciles dropped notification)
- [ ] [backend] Verify watcher runs on configured interval (Scenario: Poll interval is configurable).
- [ ] [backend] Verify a run with no delivered notification is reconciled by the next poll (Scenario: Backstop still reconciles a dropped notification) — can be exercised in dev smoke.

## 6. API contract sync (mandatory — new HTTP API)
- [ ] `api/openapi.yaml` — add `POST /api/v1/pipeline-runs/webhook` path, request body schema, `X-Databrew-Webhook-Token` header, 200/401/404 responses + error envelope. **(row 1)**
- [ ] `docs/review/api-guide.md` — section with `curl` example: happy path (valid token + known workflow) + ≥1 error path (bad token → 401, unknown workflow → 404). **(row 2)**
- [ ] `scripts/smoke-argo-push-dev.sh` (or extend `api-guide-smoke.sh`) — happy path + one error path against dev (`source scripts/dev-backend-env.sh`). **(row 5)**
- [ ] `openspec/changes/CYB-3058-argo-push-status/specs/runtime-os/spec.md` — behavior delta (done). **(row 7)**
- [ ] **Out-of-scope declaration**: SDK (row 3/4) and Frontend (row 8) NOT updated — this is a machine-only webhook called by the Argo controller, not a public REST surface consumed by SDK/Frontend. Recorded here + in Linear per AI-RULES "Out of scope declaration" (rows 1,2,5,7 satisfied).

## 7. Deploy verification (dev)
- [ ] `bash scripts/apply-migration-dev.sh ...` — N/A (no migration).
- [ ] Deploy backend dev (`deploy/cloudrun/backend-dev.sh` or `/deploy-cloudrun-dev`); `source scripts/dev-backend-env.sh`.
- [ ] Smoke: POST webhook with valid token + a known dev workflow name → 200 and run status/event updated (Scenario: Workflow success is pushed).
- [ ] Smoke: POST with bad token → 401; POST unknown workflow → 404, no mutation (Scenario: Unauthenticated / unknown-workflow rejected).
- [ ] Trigger a real dev backfill run with hook enabled; confirm terminal status lands via push (check `pipeline_run_events` + run status) before the (now slower) poll would have (Scenario: Workflow success/failure is pushed; SLO p95 < 10s).
- [ ] Confirm no status regression: after terminal push, let the poll run once and verify status stays terminal (Scenario: Stale non-terminal delivery after terminal is ignored).
- [ ] `GET /runs/watcher/status` — confirm reduced poll cadence / healthy watcher.

## 8. Verification tiers (before commit / PR)
- [ ] Tier L: `make fmt && make vet && go test ./...` (backend); OpenAPI/api-guide synced; smoke passed on dev.

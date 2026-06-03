# Tasks — CYB-1568

## OpenSpec
- [x] Create proposal, design, tasks, and spec delta.
- [x] Record OpenSpec checkpoint approval.

## Backend
- [x] [backend] Add migration extending `pipeline_run_watcher_state` with health fields.
- [x] [backend] Extend `PipelineRunWatcherState` model and repository scan/save logic.
- [x] [backend] Add watcher health/status usecase and handler.
- [x] [backend] Register `GET /api/v1/pipeline-runs/watcher/status`.
- [x] [backend] Define stable event type constants and user-facing event classification.
- [x] [backend] Ensure run submit/schedule/workflow-created events are appended during run creation.
- [x] [backend] Ensure workflow/node/pod events are appended idempotently during watcher refresh.
- [x] [backend] Ensure retry/resubmit/delete operation events record intent and result/failure where available.
- [ ] [backend] Repair recently terminal runs and mark expired Argo workflows without deleting DataBrew history.
- [ ] [backend] Keep `pipeline_run_nodes` and `pipeline_run_asset_nodes` snapshots readable after Argo expiry.
- [x] [backend] Add tests for event idempotency, operation events, Argo expired fallback, and watcher health failure/success.

## API contract sync
- [x] `api/openapi.yaml` — watcher status endpoint, watcher schema, new event types/details.
- [x] `docs/review/api-guide.md` — curl examples for watcher status and run events.
- [x] `sdk/src/cyber_databrew_sdk/` and `sdk/tests/unit/` — add watcher status if pipeline manager is public.
- [x] `scripts/smoke-pipeline-run-ledger-dev.sh` — run events + watcher status smoke.
- [x] `openspec/changes/CYB-1568-run-ledger-hardening/specs/pipeline/spec.md` — behavior delta.
- [x] [Frontend] API types/hooks aligned with OpenAPI.

## Frontend
- [x] [Frontend] Add typed watcher status client/hook.
- [x] [Frontend] Add compact watcher health indicator on execution records/detail page.
- [x] [Frontend] Translate stable event types into business labels.
- [x] [Frontend] Show "Argo expired, showing DataBrew history" state when detail data comes from stored ledger.
- [ ] [Frontend] Verify timeline filters/search/pagination against the expanded event set.

## Verification
- [x] `cd backend && go test ./internal/usecase/pipeline/... ./internal/handlers/pipeline/... ./internal/postgres/...`
- [x] `cd backend && go test ./...`
- [x] `cd Frontend && npm run lint`
- [x] `cd Frontend && npm run test -- --run WorkflowDetail WorkflowExecutionList pipelineApi`
- [x] `cd Frontend && npm run build`
- [x] `cd sdk && uv run pytest tests/unit/` if SDK changes.
- [x] Apply migration to dev before backend deploy.
- [x] Run `scripts/smoke-pipeline-run-ledger-dev.sh` against dev.
- [x] Chrome DevTools MCP: verify execution detail timeline, watcher health, and expired-history state.

## Deploy / PR
- [x] Deploy backend/frontend dev before commit.
- [x] Record Cloud Run revisions and image tags.
- [x] Update Linear CYB-1568 with verification evidence.
- [ ] Fill PR template and link CYB-1568 + OpenSpec change id.

### Deploy record — CYB-1568
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:7825a5f-cyb1568-091130` | `cyber-databrew-backend-dev-00498-5vc` | https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app |
| frontend-dev | `cyber-databrew-frontend:7825a5f-cyb1568-091130` | `cyber-databrew-frontend-dev-00307-j9w` | https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app |

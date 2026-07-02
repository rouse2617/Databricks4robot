# Tasks — CYB-1564

## Implementation
- [x] [backend] Add `PipelineRunEvent` model, event type constants, and list request/response structs.
- [x] [backend] Add `PipelineRunEventRepository` interface and update all test mocks that implement changed interfaces.
- [x] [backend] Add approved `pipeline_run_events` migration with unique idempotency and timeline indexes.
- [x] [backend] Implement Postgres event insert/list methods with conflict-safe idempotency behavior.
- [x] [backend] Append `run_submitted`, retry, stop, delete, and terminal run events in pipeline usecase flows.
- [x] [backend] Implement Argo polling watcher that observes active runs and records workflow/node/pod transition events.
- [x] [backend] Register and implement `GET /api/v1/pipeline-runs/{id}/events`.
- [x] [Frontend] Add typed API client/hook for pipeline run events.
- [x] [Frontend] Replace workflow detail run-events placeholder with a timeline using loading, empty, error, refresh, and node-click states.
- [x] [sdk] Add pipeline run events client method and unit tests.

## API contract sync (mandatory if HTTP API added/changed — same PR)
See [`docs/agents/AI-RULES.md` § API contract sync](../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [x] `api/openapi.yaml` — add `/api/v1/pipeline-runs/{id}/events`, parameters, response, and event schemas.
- [x] `docs/review/api-guide.md` — add curl examples, headers, success response, invalid cursor/unknown run error cases.
- [x] `sdk/src/cyber_databrew_sdk/` + `client.py` + `sdk/tests/unit/` — public REST client method for run events.
- [x] `scripts/smoke-pipeline-run-events-dev.sh` — happy path and invalid run/error path.
- [x] `openspec/changes/CYB-1564-run-events-watcher/specs/pipeline/spec.md` — behavior delta.
- [x] [Frontend] `Frontend/src/api/pipelineApi.ts` and workflow detail hook types align with OpenAPI.

## Local verification (Tier L)
- [x] `cd backend && go test ./internal/usecase/pipeline/... ./internal/handlers/pipeline/... ./internal/postgres/...`
- [x] `cd backend && go test ./...`
- [x] `cd Frontend && npm run lint`
- [x] `cd Frontend && npm run test -- --run WorkflowDetail pipelineApi useWorkflowDetail`
- [x] `cd Frontend && npm run build`
- [x] `cd sdk && uv run pytest tests/unit/`
- [ ] `pre-commit run --all-files`

## Deploy verification (before commit — runtime only)
- [ ] Apply approved migration to dev with `scripts/apply-migration-dev.sh`.
- [ ] Build + push backend/frontend with git SHA tag and deploy dev manually or via approved PR deploy flow.
- [ ] Backend smoke: create or find a run, call `/api/v1/pipeline-runs/{id}/events`, verify events persist after refresh.
- [ ] Frontend Chrome DevTools MCP: open dev workflow detail and verify timeline states, node-event click behavior, and console errors.
- [ ] Regression: pipeline executions list, workflow detail DAG, logs/Pod/monitoring/cost node actions.

## PR
- [ ] PR template filled with Linear `CYB-1564` and OpenSpec change-id.
- [ ] Linear issue updated with PR link and verification summary.

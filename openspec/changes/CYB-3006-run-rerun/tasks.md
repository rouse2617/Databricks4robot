# Tasks - CYB-3006

## Context files
- `backend/internal/runtimeos/run/service.go`
- `backend/internal/runtimeos/run/service_test.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/pipeline/usecase_test.go`
- `backend/internal/handlers/pipeline/handler.go`
- `backend/internal/handlers/pipeline/handler_test.go`
- `backend/routes/routes.go`
- `api/openapi.yaml`
- `Frontend/src/api/runApi.ts`
- `Frontend/src/api/runApi.test.ts`
- `sdk/src/cyber_databrew_sdk/managers/runs.py`
- `sdk/src/cyber_databrew_sdk/config/endpoints.py`
- `sdk/tests/unit/test_managers.py`
- `docs/review/api-guide.md`
- `scripts/smoke-runs-dev.sh`

## OpenSpec
- [x] [openspec] Create proposal, design, tasks, decisions, context files, and runtime-os spec delta for Run rerun.
- [x] [openspec] Continue without stopping for another checkpoint per user instruction.

## Implementation
- [x] [backend] Add `RerunRun` usecase method with requested/failed/created RunEvents.
- [x] [backend] Add Run Kernel service interface and facade methods for rerun.
- [x] [backend] Add `POST /api/v1/runs/{id}/rerun` handler and route.
- [x] [api] Add OpenAPI path for Run rerun.
- [x] [Frontend] Add `rerunRun` API wrapper and test.
- [x] [Frontend] Add Run Inspector rerun operation and confirm copy distinct from retry/resubmit.
- [x] [SDK] Add `runs.rerun` endpoint/method and unit coverage.
- [x] [docs] Document retry vs resubmit vs rerun semantics.
- [x] [scripts] Add safe unknown-run rerun smoke check.

## Verification
- [x] [backend] `cd backend && go test ./internal/runtimeos/run ./internal/usecase/pipeline ./internal/handlers/pipeline ./routes`.
- [x] [backend] `cd backend && go test ./...`.
- [x] [Frontend] `cd Frontend && npx biome check src/api/runApi.ts src/api/runApi.test.ts`.
- [x] [Frontend] `cd Frontend && npm run test -- src/api/runApi.test.ts --run`.
- [x] [Frontend] `cd Frontend && npm run build` (passed with existing CSS/chunk-size warnings).
- [x] [SDK] `cd sdk && uv run pytest tests/unit/test_managers.py -q`.
- [x] [SDK] `cd sdk && uv run ruff check src/`.
- [x] [repo] `bash -n scripts/smoke-runs-dev.sh`.
- [x] [repo] `git diff --check`.
- [ ] [repo] Non-Terraform pre-commit checks before final commit when the full runtime goal is ready.

## Deploy Verification
- [ ] Deploy backend/frontend dev with SHA images when the full runtime batch is ready for deploy gate.
- [ ] Smoke unknown-run rerun path and a controlled rerun path when safe.
- [ ] Because this slice touches frontend code, run Chrome DevTools MCP after dev frontend deploy before final commit.

## Push Strategy
- [ ] Do not open or push PRs for this slice alone.
- [ ] Keep work on local `dev`; push once only after the full Run as Kernel goal is complete and verified.

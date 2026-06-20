# Tasks - CYB-3005

## Context files
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/pipeline/usecase_test.go`
- `backend/internal/models/pipeline.go`
- `api/openapi.yaml`
- `Frontend/src/api/runApi.ts`
- `docs/review/api-guide.md`

## OpenSpec
- [x] [openspec] Create proposal, design, tasks, decisions, context files, and runtime-os spec delta for Config as Run Input.
- [x] [openspec] Continue without stopping for another checkpoint per user instruction.

## Implementation
- [x] [backend] Add sanitized materialized config input metadata to Run `PipelineJSON` at creation time.
- [x] [backend] Include deploy-level config selections in `/runs/{id}/inputs`.
- [x] [backend] Include node-level config bindings with `contentHash` in `/runs/{id}/inputs`.
- [x] [backend] Preserve historical fallback for Runs without `_run_config_inputs`.
- [x] [api] Add optional `contentHash` and `projectionKey` fields to `RunInput`.
- [x] [Frontend] Align `RunInput` TypeScript type with the OpenAPI fields.
- [x] [docs] Document Config as Run Input without exposing raw content.

## Verification
- [x] [backend] `cd backend && go test ./internal/usecase/pipeline ./internal/handlers/pipeline`.
- [x] [backend] `cd backend && go test ./...`.
- [x] [Frontend] `cd Frontend && npx biome check src/api/runApi.ts src/api/runApi.test.ts`.
- [x] [Frontend] `cd Frontend && npm run test -- src/api/runApi.test.ts --run`.
- [x] [Frontend] `cd Frontend && npm run build` (passed with existing CSS/chunk-size warnings).
- [x] [repo] `git diff --check`.
- [ ] [repo] Non-Terraform pre-commit checks before final commit when the full runtime goal is ready.

## Deploy Verification
- [ ] Deploy backend dev with SHA image and record revision when the full runtime batch is ready.
- [ ] Smoke `/api/v1/runs/{id}/inputs` for config input metadata after controlled Run creation or dry-run-compatible verification.
- [ ] If frontend files changed, run the required frontend verification path before commit.

## Push Strategy
- [ ] Do not open or push PRs for this slice alone.
- [ ] Keep work on local `dev`; push once only after the full Run as Kernel goal is complete and verified.

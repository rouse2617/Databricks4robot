# Tasks — CYB-1272 Asset Env Passthrough

## Context files
- `backend/internal/usecase/pipeline/usecase.go` - asset validation, deployment, run creation, retry paths.
- `backend/internal/transpiler/transpiler.go` - global env injection into Argo templates.
- `backend/internal/repository/asset_repository.go` - asset read contract.
- `backend/internal/models/asset.go` - available asset fields for env summaries.
- `backend/internal/usecase/pipeline/usecase_test.go` - existing asset validation tests and mocks.
- `backend/internal/usecase/pipeline/usecase_crud_test.go` - deployment and output registration regression tests.
- `docs/review/pipeline-dev-regression-log-2026-06-07.md` - dev pipeline regression context.
- `docs/agents/deploy-before-commit.md` - backend deploy gate.
- `docs/agents/deploy-verification.md` - backend dev verification.

## Implementation
- [x] [backend] Add a small asset-env assembler in `backend/internal/usecase/pipeline` for aggregate and per-asset env vars.
- [x] [backend] Include `ASSET_IDS`, `ASSET_COUNT`, `ASSETS_JSON`, indexed `ASSET_<n>_*` fields, `VIDEO_ID`, and `REQUEST_ID`.
- [x] [backend] Preserve no-asset behavior and omit indexed asset env vars when no assets are selected.
- [x] [backend] Keep missing-asset validation before manifest generation.
- [x] [backend] Ensure dry-run deployments and submitted runs use the same env assembly path.
- [x] [backend] Add or update tests to parse dry-run manifests and assert env vars are injected into all node templates.
- [x] [backend] Add tests for no-asset dry-run manifest behavior.

## API contract sync
No new or changed HTTP API is expected. Existing pipeline run endpoints keep the same request and response shape.

## Local verification
- [x] `cd backend && go test ./internal/usecase/pipeline/...`
- [x] `cd backend && go test ./internal/transpiler/...`
- [x] `cd backend && go test ./internal/handlers/pipeline/...`
- [x] `cd backend && make fmt`
- [x] `cd backend && make vet`
- [x] `cd backend && go test ./...`
- [x] `scripts/agent-harness/after-edit.sh`

## Deploy verification (before commit - runtime only)
- [x] Deploy backend dev following `docs/agents/deploy-before-commit.md`.
- [x] Backend smoke: core pipeline API endpoints on dev Cloud Run.
- [x] Dry-run with one selected known asset shows asset env vars in the rendered manifest.
- [x] User-requested Chrome DevTools MCP regression on `/pipeline` and a complex pipeline execution detail.

### Deploy record — CYB-1272
| Service | Image tag / digest | Revision | URL |
|---------|--------------------|----------|-----|
| backend-dev | `cyber-databrew-backend:04f3ce7` / `sha256:f54facea13acb2b2348936b0286b1ae703b75569e179183f96bebf8945fce210` | `cyber-databrew-backend-dev-00643-qzd` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

### Deploy smoke evidence
- Cloud Run traffic: 100% to `cyber-databrew-backend-dev-00643-qzd`.
- `GET /api/v1/pipelines`: 200, response body 42608 bytes.
- `GET /api/v1/assets/CYB10A01`: 200, asset storage `gs://cyb1100/CYB10A01`, MCAP `CYB10M01`.
- `POST /api/v1/deploy?dryRun=true` with `asset_ids=["CYB10A01"]`: 200 preview `cyb1272-env-smoke-04f3ce7-76c543`.
- Dry-run manifest includes `PIPELINE_DEPLOYMENT_ID`, `ASSET_IDS=CYB10A01`, `ASSET_COUNT=1`, `ASSET_0_ID`, `ASSET_0_STORAGE_URI`, `ASSET_0_MCAP_FILE_ID`, `ASSET_0_START_NS`, `ASSET_0_END_NS`, `ASSET_0_LOGICAL_ASSET_ID`, `ASSET_0_REVISION`, `ASSET_0_IS_CURRENT`, `ASSET_0_METADATA_JSON`, `ASSETS_JSON`, `VIDEO_ID`, and `REQUEST_ID`.
- Chrome DevTools MCP `/pipeline`: auth/me 200, pipeline-components 200, execution-targets 200, no console error/warn.
- Chrome DevTools MCP complex DAG detail: run/workflow/events/asset-nodes/cost-summary APIs all 200, 8 nodes and 52 events rendered, no console error/warn.

## PR
- [ ] PR template filled; Linear `CYB-1272` linked.
- [ ] PR body includes OpenSpec change id `CYB-1272-asset-env-passthrough`.
- [ ] Linear updated after merge with commit hash.

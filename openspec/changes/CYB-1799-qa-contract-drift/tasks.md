# Tasks — CYB-1799

## Context files
- `api/openapi.yaml`
- `docs/review/api-guide.md`
- `Frontend/src/pages/AlgoRunsPage.tsx`
- `Frontend/src/api/algoRuns.ts`
- `Frontend/src/api/mcapFiles.ts`
- `Frontend/src/api/pipelineApi.ts`
- `sdk/src/cyber_databrew_sdk/config/endpoints.py`
- `sdk/src/cyber_databrew_sdk/managers/algo_runs.py`
- `sdk/src/cyber_databrew_sdk/managers/storage.py`
- `sdk/src/cyber_databrew_sdk/managers/pipelines.py`
- `sdk/tests/unit/test_managers.py`
- `backend/internal/handlers/asset/handler.go`
- `backend/internal/handlers/algorun/handler.go`
- `backend/routes/routes.go`

## Implementation
- [x] [api/docs] Document asset event `event_type`, `algo_key`, `cursor`, and `next_cursor` contract.
- [x] [api/docs] Document MCAP list `owner` and `ingest_state` filters.
- [x] [api/docs] Document algo run list date filters and canonical filter vocabulary.
- [x] [api/docs] Document existing pipeline active-version, promote, and batch template-run routes.
- [x] [sdk] Add MCAP list filter parity.
- [x] [sdk] Align algo run list params with backend/frontend filters while preserving safe compatibility where practical.
- [x] [sdk] Add pipeline active-version, promote, and batch template-run helpers.
- [x] [Frontend] Show a visible algo-runs list load error instead of silently rendering empty results.

## API Contract Sync
- [x] `api/openapi.yaml` updated for all changed documented HTTP surfaces.
- [x] `docs/review/api-guide.md` updated with examples and error notes where applicable.
- [x] `sdk/src/cyber_databrew_sdk/` updated for public REST surface parity.
- [x] `sdk/tests/unit/` updated for SDK methods and query params.
- [x] Frontend types/hooks checked against updated contracts.

## Verification
- [ ] `cd sdk && uv run ruff check src/`
- [x] `cd sdk && uv run ruff check src/cyber_databrew_sdk/config/endpoints.py src/cyber_databrew_sdk/managers/storage.py src/cyber_databrew_sdk/managers/algo_runs.py src/cyber_databrew_sdk/managers/pipelines.py`
- [x] `cd sdk && uv run pytest tests/unit/test_managers.py -q`
- [x] `cd Frontend && npm run lint`
- [x] `cd Frontend && npm run test -- --run src/pages/AlgoRunsPage.test.tsx src/api/algoRuns.test.ts`
- [x] OpenAPI YAML parses with `python3 -c` / PyYAML.
- [x] Chrome DevTools MCP verification recorded after frontend dev deployment.

Notes:
- Full SDK ruff currently fails on pre-existing `A002` in `sdk/src/cyber_databrew_sdk/managers/search.py`; touched SDK files pass ruff.
- Frontend dev Worker deploy: `cyber-databrew-dev`, Version ID `dd311c79-eaf2-47a6-8cbd-48fa2ba927e9`, URL `https://cyber-databrew-dev.cyberorigin.ai/`.
- Chrome DevTools MCP opened `/algo-runs`, confirmed the dev build badge (`v0.1.1 · f04d3d9 · dev`) and loaded algo-runs table data. Screenshot: `deploy-verify-algo-runs-loaded.png`.

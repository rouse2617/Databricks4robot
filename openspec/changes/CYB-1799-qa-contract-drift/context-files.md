# Context files — CYB-1799

- `api/openapi.yaml` — public API contract source.
- `docs/review/api-guide.md` — human API guide examples.
- `backend/routes/routes.go` — existing route registrations to document.
- `backend/internal/handlers/asset/handler.go` — asset event filter behavior.
- `backend/internal/handlers/algorun/handler.go` — algo run list filter behavior.
- `Frontend/src/pages/AlgoRunsPage.tsx` — hidden failure state.
- `Frontend/src/api/algoRuns.ts` — algo run query params.
- `Frontend/src/api/mcapFiles.ts` — MCAP query params.
- `Frontend/src/api/pipelineApi.ts` — pipeline helper surface.
- `sdk/src/cyber_databrew_sdk/config/endpoints.py` — SDK route registry.
- `sdk/src/cyber_databrew_sdk/managers/algo_runs.py` — SDK algo run manager.
- `sdk/src/cyber_databrew_sdk/managers/storage.py` — SDK MCAP manager.
- `sdk/src/cyber_databrew_sdk/managers/pipelines.py` — SDK pipeline manager.
- `sdk/tests/unit/test_managers.py` — SDK route/query tests.

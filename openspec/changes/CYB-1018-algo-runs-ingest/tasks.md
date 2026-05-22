# Tasks — CYB-1018

- [x] OpenSpec proposal + tasks
- [x] `backend/migrations/031_algo_runs.sql` + orphan `run_id` cleanup + FKs
- [x] `models.AlgoRun`, repository, postgres repo
- [x] `usecase/algorun` + `handlers/algorun` + routes + `core.go` wiring
- [x] `algo_run_applied` schema + emit on `FinishAlgo` with registered `run_id`
- [x] Validate `run_id` on asset `algo` start/finish when provided (16-char + registered)
- [x] Unit tests (usecase state machine); `go test ./...` green
- [x] `api/openapi.yaml` + `docs/review/api-guide.md` §2.0
- [x] `scripts/api-guide-smoke.sh` algo-runs tracer
- [x] Deploy backend dev `47d8663` → revision `cyber-databrew-backend-dev-00152-r7q` (2026-05-22)
- [x] Migration `031_algo_runs.sql` on dev PG
- [ ] Linear CYB-1018 → Done (UI「来自 run」deferred)

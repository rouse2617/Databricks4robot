# Tasks — CYB-1123

## Context files
- `backend/internal/handlers/algorun/handler.go` — inspect current query parsing and response envelope
- `backend/internal/postgres/algo_runs.go` — inspect repo paging and total semantics
- `Frontend/src/api/algoRuns.ts` — inspect current request contract
- `Frontend/src/pages/AlgoRunsPage.tsx` — inspect UI pagination behavior
- `backend/internal/handlers/delivery/handler.go` — inspect draft/retry/commit/ack flows
- `backend/internal/postgres/repos.go` — inspect delivery index writes
- `backend/internal/models/delivery_transition.go` — inspect state transitions

## Implementation
- [ ] [backend] Reproduce and confirm root causes for algo-runs pagination, duplicate `run_id`, draft/retry indexing, and delivery ack behavior
- [ ] [backend] Fix `GET /algo-runs` request parsing and response semantics so backend pagination matches the frontend contract
- [ ] [Frontend] Align algo-runs page request params and pagination behavior with the backend contract
- [ ] [backend] Align duplicate `run_id` create behavior with the chosen documented API contract
- [ ] [backend] Prevent draft/retry delivery flows from mutating asset delivery indexes or delivery-committed semantics before final commit
- [ ] [backend] Align delivery acknowledgement behavior with the delivery state model and exposed API behavior
- [ ] [backend] Add or update focused tests for the corrected paths

## API contract sync (mandatory if HTTP API added/changed — same PR)
See [`docs/agents/AI-RULES.md` § API contract sync](../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [ ] `api/openapi.yaml` — algo-runs list/cancel/affected-assets, customers list, delivery operations, and any corrected response shapes or status semantics
- [ ] `docs/review/api-guide.md` — curl examples and error-path notes for corrected algo-runs and delivery behavior
- [ ] `sdk/src/asset_sdk/` + `client.py` (+ `sdk/tests/unit/` if SDK touched)
- [ ] `scripts/api-guide-smoke.sh` or targeted smoke scripts for corrected algo-runs and delivery behavior
- [ ] `openspec/changes/CYB-1123-*/specs/*/spec.md` — behavior delta
- [ ] [Frontend] `src/api/` or hooks — keep UI client aligned with final API contract

## Local verification (Tier L per AI-RULES)
- [ ] `cd backend && make fmt && make vet && go test ./...`
- [ ] `cd Frontend && npm run lint && npm run build`
- [ ] `cd sdk && env UV_CACHE_DIR=/private/tmp/uv-cache uv run pytest tests/unit/` if SDK changes

## Deploy verification (before commit — runtime only)
- [ ] Build + push with git SHA tag and `cloudrun-dev-latest` per `docs/agents/deploy-before-commit.md`
- [ ] Deploy using `IMAGE=…:<sha>` and record image tag, revision, and dev URL

### Frontend（仅当本 change 修改 `Frontend/` 代码 — 必填）
- [ ] Chrome DevTools MCP on dev: verify algo-runs list pagination and detail navigation
- [ ] Reduced regression pages per `deploy-verification.md`: `/algo-runs`, `/algo-runs/:run_id`, affected asset detail flow
- [ ] Screenshot saved in this change dir
- [ ] Console: no new errors

### Backend (if `backend/` changed)
- [ ] Smoke corrected algo-runs list and duplicate create behavior on dev
- [ ] Smoke draft/commit/retry/ack delivery behavior on dev

## PR
- [ ] PR template filled; Linear `CYB-1123` linked

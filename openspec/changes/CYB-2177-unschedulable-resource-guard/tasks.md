# Tasks — CYB-2177

## Context files
- `backend/internal/usecase/pipeline` — run submission, status refresh, watcher logic.
- `backend/internal/usecase/backfill` — batch item and job aggregation.
- `backend/internal/transpiler/transpiler.go` — resource rendering into Argo workflow manifests.
- `backend/internal/transpiler/pipeline.go` — pipeline component resource shape.
- `backend/internal/postgres` — run and backfill persistence.
- `docs/agents/deploy-before-commit.md` — deploy gate before commit.
- `docs/agents/deploy-verification.md` — backend dev smoke after deploy.

## Implementation
- [x] [backend] Locate the run watcher/status refresh path that persists Argo node messages and run terminal states.
- [x] [backend] Add unschedulable detection for messages containing scheduler causes such as `Unschedulable`, `Insufficient cpu`, `Insufficient memory`, `Insufficient ephemeral-storage`, and `untolerated taint`.
- [x] [backend] Gate unschedulable failure by a configurable threshold so short `Pending` windows remain non-terminal.
- [x] [backend] Persist the original scheduler reason on the run or node message when converting to `Error`.
- [x] [backend] Ensure backfill item and batch aggregation treats these errored child runs as failed/terminal.
- [x] [backend] Add configurable static resource ceilings for CPU, memory, disk, and GPU; unset values disable the corresponding check.
- [x] [backend] Validate pipeline node component resources before workflow submission and return `INVALID_ARGUMENT` with the exceeded resource and configured limit.
- [x] [backend] Add tests for below-threshold Pending, over-threshold Unschedulable, and deploy-time over-limit validation.

## API contract sync
No new HTTP routes are expected. Existing create-run/deploy/backfill endpoints may return existing error envelopes earlier for invalid resource requests.

- [x] `api/openapi.yaml` — not required; no route, status envelope, or response schema changed.
- [x] `docs/review/api-guide.md` — update if deploy/run validation errors are documented.
- [x] `scripts/api-guide-smoke.sh` or targeted smoke — not required; existing create-run endpoint was covered by targeted dev curl.
- [x] `openspec/changes/CYB-2177-unschedulable-resource-guard/specs/pipeline/spec.md` — behavior delta complete.

## Local verification
- [x] `cd backend && go test ./internal/usecase/pipeline/... ./internal/usecase/backfill/... ./internal/config`
- [x] Add narrower package tests if the implementation lands in repository/transpiler packages — not applicable; implementation stayed in usecase/config.
- [x] `cd backend && go test ./...`
- [x] `cd backend && go vet ./...`
- [x] `git diff --check`
- [x] `pre-commit run --files <changed files>`

## Deploy verification
- [x] Build backend image with pre-commit verification tag.
- [x] Deploy backend dev with the exact image.
- [x] Smoke the previously observed batch or a controlled unschedulable fixture and verify terminal status convergence.
- [x] Smoke a normal lightweight batch to confirm it still runs.
- [x] Record image tag, Cloud Run revision, and API smoke evidence before commit.

### Deploy record — CYB-2177
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:cyb-2177-afdc4fc-20260618124521` | `cyber-databrew-backend-dev-00901-vwf` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

### Smoke evidence — CYB-2177
- `GET /readyz` -> `200`, PG healthy.
- `GET /api/v1/pipeline-runs/watcher/status` -> `200`, `healthy=true`, `stale=false`.
- Over-limit `POST /api/v1/pipeline-runs` with `cpu=14`, `memory=55Gi`, `disk=50Gi` -> `400 INVALID_ARGUMENT`, message includes `请求 cpu=14，最大可用 cpu=8`.
- Existing stuck workflow `my-pipeline-00e7d3` -> DataBrew run `6e1a3ae6-b81d-4f21-b2de-8e0748e149c9` became `Error` with scheduler diagnostics preserved.
- Lightweight run `0d4af3e3-5a66-4ad4-87d2-9061b1c57931` (`busybox`, `100m/128Mi/1Gi`) -> `Succeeded`.

## PR
- [ ] PR template filled; Linear `CYB-2177` linked.
- [ ] Include risk note that ResourceQuota/LimitRange rollout is deferred.

# Current work context

Human-updated scratchpad for compact recovery. All agents should read this after context compaction alongside [`docs/agents/AI-RULES.md`](../../docs/agents/AI-RULES.md).

---

## Active

- **Branch**: `feat/pipeline-integration`

## Recently completed (2026-05-28)

### Pipeline tasks — all T-01~T-13 done ✓

| ID | Priority | Description | Status |
|----|----------|-------------|--------|
| T-01 | P0 | Workflow logs route registration | ✅ |
| T-02 | P0 | Workflow detail log UI | ✅ |
| T-03 | P1 | Fix template edit loading | ✅ |
| T-04 | P1 | Canvas deploy modal with optional assets | ✅ |
| T-05 | P1 | Workflow list pagination/filtering | ✅ |
| T-06 | P1 | Component registry API integration | ✅ |
| T-07 | P2 | Deploy success modal enhancement | ✅ |
| T-08 | P2 | Asset modal copy tweaks | ✅ |
| T-09 | P2 | React Flow multi-handles | ✅ |
| T-10 | P2 | Asset detail pipeline lineage (LineageTab) | ✅ |
| T-11 | P3 | OpenAPI sync (workflows/{name}/logs) | ✅ |
| T-12 | P1 | Asset existence validation before deploy (backend) | ✅ |
| T-13 | P2 | Display asset storage URI in asset picker (frontend) | ✅ |

### What was done this cycle (2026-05-28 cycle 4) — Pipeline handler tests

1. **New test file**: `backend/internal/handlers/pipeline/handler_test.go` (35 tests)
   - 9 template tests (SaveTemplate success/validation, ListTemplates empty/with items, GetTemplate success/not found/empty ID, DeleteTemplate success/empty ID, ListVersions)
   - 3 deploy tests (Deploy success/missing pipeline, DeployByTemplate success/not found/empty ID)
   - 13 deployment tests (ListDeployments empty/with items, GetDeployment success/not found/empty ID, DeleteDeployment success/empty ID, RetryDeployment success/not found, StopDeployment success/not found, SaveFromDeployment success/not found, GetResourceUsage not found)
   - 4 register output tests (success, missing deployment_id, invalid body, deployment not found)
   - 2 lineage tests (success, empty ID)
   - 2 error mapping tests (asset validation, template not found)

2. **New test file**: `backend/internal/handlers/pipeline_component/handler_test.go` (13 tests)
   - CreateComponent (success, invalid body)
   - ListComponents (empty, with items, with query/source filter)
   - GetComponent (success, not found, empty ID)
   - UpdateComponent (success, invalid body, empty ID)
   - DeleteComponent (success, empty ID)

3. **Verified**: `go build ./...` ✅, `go test ./internal/handlers/pipeline/...` (35/35 PASS) ✅, `go test ./internal/handlers/pipeline_component/...` (13/13 PASS) ✅, `go test ./internal/usecase/pipeline/...` ✅, `go test ./internal/usecase/pipeline_component/...` ✅, `go test ./internal/transpiler/...` ✅

## What was done this cycle (2026-05-28 cycle 5) — Orphaned routes + OpenAPI sync

1. **Fixed orphaned handler routes** — `RetryDeployment` and `StopDeployment` handler functions existed (with tests) but were never registered in `routes.go`. Added:
   - `POST /api/v1/deployments/:id/retry` → `pipelineHandler.RetryDeployment`
   - `POST /api/v1/deployments/:id/stop` → `pipelineHandler.StopDeployment`

2. **OpenAPI sync** — Added `/api/v1/deployments/{id}/retry` and `/api/v1/deployments/{id}/stop` endpoints with full request/response schemas. Fixed param name in `/api/v1/pipelines/{id}/diff/{id2}` from `id1` → `id` to match actual Gin route param.

3. **Verified**: `go build ./...` ✅, `go vet ./...` ✅, `npx tsc --noEmit` ✅, pipeline handler tests 35/35 PASS ✅

## What's next

- Review frontend pipeline pages for TypeScript type coverage / unused imports
- Check for integration-level route tests (routes_test.go lacks pipeline route tests)
- Review openapi.yaml for any other param naming inconsistencies across all domains

## Non-Done DataBrew

| Issue | Status |
|-------|--------|
| CYB-979 | In Progress — Cloud Run CI/CD + 飞书通知 |
| CYB-1030 | In Progress — AI Coding Governance |
| CYB-1045/1107 | Backlog — SDK parity pack (暂缓) |
| CYB-1040/1102/1103 | Backlog — 治理 HITL 决策 |
| CYB-1019/1064~1067 | Backlog — annotation_tasks (ON HOLD) |

---

See [memory.md](memory.md) for project stack, user profile, and workflow rules.

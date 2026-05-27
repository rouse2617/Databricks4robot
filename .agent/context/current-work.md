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

### What was done this cycle (2026-05-28 cycle 3) — Pipeline component usecase + workflow handler tests

1. **New test file**: `backend/internal/usecase/pipeline_component/usecase_test.go` (15 tests)
   - `TestCreate`: basic creation, validates ID/ports/source defaults
   - `TestCreate_EmptyList`: empty list on fresh repo
   - `TestCreateAndList`: create 2 items, list all + filter by query
   - `TestList_FilterBySource`: filter by system/custom source
   - `TestGet_Existing` / `TestGet_NonExistent`: get lifecycle
   - `TestUpdate_Existing` / `TestUpdate_NonExistent`: update lifecycle
   - `TestDelete_Existing` / `TestDelete_NonExistent`: delete lifecycle
   - `TestSeedSystemComponents_Empty` / `_Idempotent`: seeding behavior
   - `TestCreate_WithExplicitSource`: preserves custom source
   - `TestCreate_WithPorts`: preserves input/output port definitions
   - `TestUpdate_PreservesCreatedAt`: update preserves original CreatedAt

2. **New test file**: `backend/internal/handlers/workflow/handler_test.go` (7 tests)
   - `TestListWorkflows_Empty`: empty response with items: []
   - `TestListWorkflows_WithItems`: name, status, nodeCount in response
   - `TestGetWorkflow_Success`: full workflow with nodes array
   - `TestGetWorkflow_EmptyName`: Gin route matching for /workflows/
   - `TestGetWorkflowLogs_Success`: logs returned via mock
   - `TestGetWorkflowLogs_EmptyNodeId`: 400 without nodeId query param
   - `TestGetWorkflowLogs_EmptyName`: 400 with empty name

3. **Verified**: `go build ./...` ✅, `go test ./internal/usecase/pipeline_component/...` (15/15 PASS) ✅, `go test ./internal/handlers/workflow/...` (7/7 PASS) ✅, `go test ./internal/usecase/pipeline/...` ✅, `go test ./internal/transpiler/...` ✅, `npx tsc --noEmit` ✅

## What's next

All pipeline-next-steps.md tasks complete (T-01~T-13). All pipeline-related packages now have test coverage:
- `internal/usecase/pipeline` — 34 tests ✅
- `internal/usecase/pipeline_component` — 15 tests ✅
- `internal/handlers/workflow` — 7 tests ✅
- `internal/handlers/pipeline` — 0 tests (could add next)
- `internal/handlers/pipeline_component` — 0 tests (could add next)

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

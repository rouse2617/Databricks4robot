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

### What was done this cycle (2026-05-28 cycle 12) — AssetPicker frontend component tests

1. **Created `Frontend/src/components/pipeline/AssetPicker.test.tsx`** with 9 tests:
   - Basic rendering: search input with default/custom placeholder, empty state hint
   - Search flow: API call triggers with correct params, results displayed in table
   - storage_uri display: long URI truncated with ellipsis, missing URI shows dash
   - Row selection: checkbox click calls onSelectionChange with asset ID
   - Edge cases: empty search results message, API failure handled gracefully

2. **Verified**: `npx vitest run` ✅ (9/9 passed) + pipelineContract tests (23/23 passed), `npx tsc --noEmit` ✅, `go build ./...` ✅, all backend pipeline tests ✅

3. **Deployed**: Frontend dev Cloud Run (revision 00268-m8v)

### What's next

- Frontend component tests for DeployPanel and ComponentManager remain as future work
- All pipeline tasks T-01~T-13 are completed and verified
- All `any`/`unknown` type gaps in pipeline frontend code resolved
- OpenAPI is fully in sync with routes.go for all pipeline endpoints
- Pipeline contract now has comprehensive unit test coverage (23 tests)
- AssetPicker now has component test coverage (9 tests)
- No remaining pipeline implementation or quality tasks
- Next cycle: review entire codebase for cross-module improvements

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

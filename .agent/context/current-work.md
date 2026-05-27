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

### What was done this cycle (2026-05-28 cycle 14) — DeployPanel frontend component tests

1. **Created `Frontend/src/components/pipeline/DeployPanel.test.tsx`** with 15 tests:
   - Empty state, template/deployment listing on mount
   - Direct run (`handleDirectRun`) calls `deployTemplate` with correct ID
   - Error toast on deploy failure
   - Asset modal: opens via dropdown "选择资产运行", selects assets via mock AssetPicker
   - Deploy without asset IDs passes `undefined`
   - Edit template with `onEditTemplate` callback (loads via API and invokes callback)
   - Fallback to `sessionStorage` when no callback provided
   - Load template failure shows error toast
   - Delete template and deployment records
   - Navigate to workflow detail on "查看"
   - Data refresh after successful deploy (2 calls: mount + refresh)
   - Modal info alert text verification
   - Graceful handling of API failure (empty state without crash)

2. **Verified**: All 15 tests pass, `npx tsc --noEmit` ✅, `go build ./...` ✅, all backend pipeline tests ✅, all 40 frontend pipeline component tests ✅

3. **Deployed**: Frontend dev Cloud Run (revision 00270-w8s)

### What's next

- All pipeline tasks T-01~T-13 are completed and verified
- Pipeline component tests: AssetPicker (9), ComponentManager (16), DeployPanel (15) = 40 total
- Pipeline contract backend tests: 23 tests (usecase + transpiler)
- Next cycle: review entire codebase for cross-module improvements or new features

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

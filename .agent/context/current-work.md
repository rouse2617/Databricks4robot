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

### What was done this cycle (2026-05-28 cycle 13) — ComponentManager frontend component tests

1. **Created `Frontend/src/components/pipeline/ComponentManager.test.tsx`** with 16 tests:
   - Empty state, component list rendering, "新建" button click opens form
   - Opens edit form when clicking a component in the list
   - Save new component: local state (`onChange`) and API callback (`onSaveApi`) both triggered
   - Save existing component: update API called with `isNew=false`
   - Delete component: `onDeleteApi` called with correct ID, local state updated
   - API failure fallback: `message.warning` shown, local state preserved despite API error
   - Command JSON parsing on blur with proper array parsing
   - Cancel closes the form, no leftover form UI
   - New components don't show delete button
   - Active component highlighted with `.cm-item.active` class
   - Editing context switches correctly between components
   - Local-only mode (no API props) works as expected

2. **Verified**: `npx vitest run src/components/pipeline/ComponentManager.test.tsx` ✅ (16/16 passed), `npx tsc --noEmit` ✅, `go build ./...` ✅, all backend pipeline tests ✅

3. **Deployed**: Frontend dev Cloud Run (revision 00269-4tt)

### What's next

- Frontend component tests for DeployPanel remain as future work
- All pipeline tasks T-01~T-13 are completed and verified
- Pipeline contract test coverage: 23 tests
- AssetPicker component test coverage: 9 tests
- ComponentManager component test coverage: 16 tests
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

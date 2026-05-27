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

### What was done this cycle (2026-05-28 cycle 15) — PipelinePage 25-unit-test suite

1. **Fixed `Frontend/src/pages/PipelinePage.test.tsx`** — 25 tests covering the full canvas deploy flow:
   - Render & structure (3): tabs, toolbar, name input
   - Export/Import (4): JSON output, valid/invalid/cancel import
   - Save (2): success toast, error toast
   - Deploy dialog (10): modal open, node counts (0/1), deploy w/wo assets, error display, "查看 Workflow" navigation, "查看部署" tab switch, "关闭" button
   - Component registry (2): API load on mount, localStorage fallback
   - sessionStorage (3): load on mount, empty, invalid JSON

2. **Fixed 5 flaky tests** — Ant Design 5 inserts spaces in CJK button text ("部 署", "关 闭"). Replaced fragile `getAllByText("部署").pop()?.closest("button")` with `document.querySelector('.ant-btn-primary:not(.ant-btn-sm)')` and regex matchers for spaced text. Old deploy tests never actually clicked the modal deploy button due to this spacing issue.

3. **Verified**: All 25 PipelinePage tests pass ✅, all 40 pipeline component tests pass ✅, `npx tsc --noEmit` ✅, `go build ./...` ✅

### What's next

- All pipeline tasks T-01~T-13 are completed and verified
- Pipeline frontend tests: PipelinePage (25) + AssetPicker (9) + ComponentManager (16) + DeployPanel (15) + pipelineContract (unit) = 66 total
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

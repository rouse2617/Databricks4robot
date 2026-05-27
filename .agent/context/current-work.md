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

### What was done this cycle (2026-05-28 cycle 11) — Frontend test coverage: pipelineContract.ts

1. **Created `Frontend/src/lib/pipelineContract.test.ts`** with 23 tests:
   - `formatEdgeEndpoint` (7 tests): handles with explicit handles, empty/null/undefined/dotted handles, defaults, custom ports, pre-dotted nodeIds
   - `toTranspilerPipeline` (7 tests): empty conversion, single node correctness, resources inclusion/omission, edge handle conversion, version passthrough
   - `fromTranspilerPipeline` (5 tests): node/edge restoration, resource extraction, empty resource defaults, handle-less edges, sequential edge IDs
   - Round-trip (3 tests): full data preservation through to→from→to, name preservation, empty pipeline round-trip
   - 1 edge case: `fromTranspilerPipeline` handles `resources` absent in component

2. **Verified**: `npx vitest run` ✅ (23/23 passed), `npx tsc --noEmit` ✅, `go build ./...` ✅

### What's next

- All pipeline tasks T-01~T-13 are completed and deployed
- All `any`/`unknown` type gaps in pipeline frontend code resolved
- OpenAPI is fully in sync with routes.go for all pipeline endpoints
- Pipeline contract now has comprehensive unit test coverage
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

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

### What was done this cycle (2026-05-28 cycle 7) — Dead code cleanup + API client dedup

1. **Removed dead code**:
   - Removed `deploy()` function from `pipelineApi.ts` (never imported/called)
   - Removed `getComponent()` from `pipelineComponentApi.ts` (never imported/called)

2. **Extracted shared `pipelineClient.ts`**:
   - Moved duplicated `request()` function and `ApiError` class from 3 files into `Frontend/src/api/pipelineClient.ts`
   - Refactored `pipelineApi.ts`, `pipelineComponentApi.ts`, `workflowApi.ts` to import from shared module
   - Preserved original Axios-based `client.ts` (used by asset/auth modules) unchanged

3. **Verified**: `npx tsc --noEmit` ✅, `go build ./...` ✅
4. **Deployed**: frontend-dev ✅ (revision 00265)

### What's next

- Review remaining TypeScript any-types across the frontend (present in asset-related pages, not pipeline)
- Consider extracting `assetColumns` in DeployPanel and PipelinePage into a shared AssetPicker component

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

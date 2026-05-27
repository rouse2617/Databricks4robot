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

### What was done this cycle (2026-05-28 cycle 8) — AssetPicker extraction + PipelinePage fix

1. **Fixed broken PipelinePage.tsx deploy modal**:
   - Previous agent left the file in a broken state: removed asset search state variables (`assetSearching`, `assetSearchResults`, `assetQuery`) but the JSX still referenced them
   - Line 627 had corrupted indentation (literal `\t` characters instead of actual tabs)
   - `Table` was removed from imports but still used in JSX → would fail `tsc --noEmit`
   - Replaced the entire inline asset search/table code (60 lines) with `<AssetPicker>` component (21 lines)

2. **Extracted shared `AssetPicker` component**:
   - Created `Frontend/src/components/pipeline/AssetPicker.tsx` — reusable, stateless asset search + multi-select table
   - Props: `selectedIds`, `onSelectionChange`, `placeholder`, `maxHeight`
   - Columns: Asset ID, 类型, 状态, 存储路径 (with truncation)
   - Used by both `DeployPanel.tsx` and `PipelinePage.tsx` deploy modal

3. **Refactored `DeployPanel.tsx`** to use `AssetPicker` (removed ~80 lines of duplicated inline code)

4. **Added `credentials: "include"`** to `pipelineClient.ts` for cookie-based auth

5. **Verified**: `npx tsc --noEmit` ✅, `go build ./...` ✅
6. **Deployed**: frontend-dev ✅ (revision 00266)

### What's next

- All pipeline tasks T-01~T-13 are completed and deployed
- No remaining `any` types in pipeline frontend code
- Next step if continuing: review `api/openapi.yaml` for missing endpoints, or check for code improvements across the codebase

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

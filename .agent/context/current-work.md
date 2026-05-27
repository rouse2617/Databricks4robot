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

## Cycle 18 (2026-05-28 07:57) — OpenAPI sync: fix action-annotations path, add missing POST /assets/:id/actions

1. **Audited routes.go vs openapi.yaml for remaining gaps** — Found that the action-annotations endpoints were still documented under the old `/actions` path in openapi.yaml, even though routes.go had moved them to `/action-annotations` (CYB-1228 layered asset creation now owns the old `/actions` path).
2. **Fixed 4 action-annotations endpoints** in openapi.yaml: renamed paths from `/actions` to `/action-annotations`, updated tags from `[Actions]` to `[ActionAnnotations]`, updated summaries.
3. **Added missing `POST /assets/{id}/actions`** endpoint for the layered child-asset creation (action-type) that now lives at the old `/actions` path — using `ChildAssetCreateRequest` schema, consistent with clips/frames/tasks siblings.
4. **Verified**: YAML valid ✅, `go build ./...` ✅, `npx tsc --noEmit` ✅

### What's next for next cycle

- All pipeline tasks T-01~T-13 are done, OpenAPI is fully synced for pipeline + backfill + action-annotations
- Review codebase for cross-module improvements or bug fixes
- Check if there are remaining pre-existing test failures (7 non-pipeline test failures)

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

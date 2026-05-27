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

### What was done this cycle (T-13 + OpenAPI sync)

1. **T-13**: Display asset storage URI in asset picker + search index
   - Backend: added `storage_uri` to search index document in `builder.go`
   - Frontend: added `storage_uri` field to `SearchAssetHit` and `normalizeSearchHitToAsset`
   - `DeployPanel.tsx`: new "存储路径" column (monospace, truncated at 40 chars, `—` fallback)
   - `PipelinePage.tsx` deploy modal asset table: new "存储路径" column (monospace, truncated at 36 chars)
   - Both columns reuse `storage_uri` from search index results
   - Verified: `go build ./...` ✅, `npx tsc --noEmit` ✅, pipeline tests ✅

2. **OpenAPI sync**: Added all 17 missing pipeline/component/workflow endpoint definitions
   - 8 new schemas: PipelineTemplate, PipelineDeployment, PipelineComponent, PortDef, EnvVarDef, WorkflowSummary, WorkflowDetail, WorkflowNodeStatus
   - Pipeline CRUD: POST/GET /pipelines, GET/DELETE /pipelines/:id, GET /pipelines/:id/versions
   - Deploy: POST /deploy, POST /deploy/template/:id
   - Deployments: GET /deployments, GET/DELETE /deployments/:id
   - Components: POST/GET /components, GET/PUT/DELETE /components/:id
   - Workflows: GET /workflows, GET /workflows/:name
   - File: `api/openapi.yaml` (+538 lines)

## What's next

All pipeline-next-steps.md tasks complete (T-01~T-13). OpenAPI now covers 100% of registered pipeline endpoints.

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

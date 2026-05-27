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

### What was done this cycle (2026-05-28 cycle 6) — Integration route tests

1. **Added 3 new test functions** in `backend/routes/routes_test.go` covering **50 subtests**:
   - `TestPipelineRoutes_Registered` — 34 subtests (17 pipeline routes × 2: no-auth=401 + with-auth=expected)
   - `TestPipelineComponentRoutes_Registered` — 10 subtests (5 component routes × 2)
   - `TestWorkflowRoutes_Registered` — 6 subtests (3 workflow routes × 2)

2. **Mock implementations added**:
   - `routePipelineTemplateRepo` — mock for `PipelineTemplateRepository`
   - `routePipelineDeploymentRepo` — mock for `PipelineDeploymentRepository`
   - `routePipelineComponentRepo` — mock for `PipelineComponentRepository`
   - `mockWorkflowClient` — mock for `k8s.WorkflowClient` (implements all 7 methods)

3. **Routes verified as properly registered**:
   - Pipeline templates: POST/GET/DELETE /pipelines, GET /pipelines/:id/versions, GET /pipelines/:id/diff/:id2
   - Deploy: POST /deploy, POST /deploy/template/:id
   - Deployments: GET /deployments, GET /deployments/:id, GET /deployments/:id/resources, POST /deployments/:id/retry|stop|save-template, DELETE /deployments/:id
   - Assets: POST /pipeline-assets, GET /assets/:id/pipeline-lineage
   - Components: POST/GET/PUT/DELETE /components
   - Workflows: GET /workflows, GET /workflows/:name/logs, GET /workflows/:name

4. **Verified**: `go build ./...` ✅, `go test ./routes/...` (PASS) ✅, `go test ./internal/handlers/pipeline/...` ✅, `go test ./internal/handlers/pipeline_component/...` ✅, `npx tsc --noEmit` ✅

## What's next

- Review frontend pipeline pages for TypeScript type coverage / unused imports
- Review openapi.yaml for any other param naming inconsistencies across all domains

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

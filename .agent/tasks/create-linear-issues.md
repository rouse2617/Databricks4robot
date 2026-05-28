# Task: Create Linear Issues for Round 1-2 Work and Link Commits

We need to retroactively create Linear issues for all the work done and link them to commits. Going forward, every change must have a CYB-xxx before work starts.

## Completed Work (needs CYB issues)

### Round 1

1. **Backend bug fixes** — nil pointer panic in pipeline test, CSRF protection, migration idempotency
   - Commit: `9b2abf6` (branch: round1/fix-backend → merged to feat/pipeline-integration)
   - Files: backend/internal/handlers/pipeline/handler_test.go, backend/routes/routes.go, backend/migrations/...

2. **Frontend bug fixes** — ComponentManager error swallowing, AssetPicker race condition, lint, CSRF header
   - Commit: `9c7b45d` (branch: round1/fix-frontend → merged)
   - Files: Frontend/src/components/pipeline/ComponentManager.tsx, AssetPicker.tsx, PipelinePage.tsx

3. **Phase 0 backend** — 6 new Argo Workflow client methods + handlers + routes + extended response fields
   - Merged but NOT a single commit (part of the merge commit)
   - Files: backend/internal/argo/client.go, backend/internal/handlers/workflow/, backend/routes/routes.go
   - New methods: RetryWorkflow, ResubmitWorkflow, SuspendWorkflow, ResumeWorkflow, TerminateWorkflow

4. **Phase A — workflow summary counts + name search**
   - Commit: `d3cc529` (branch: round1/phase-a-nodep → merged)
   - Files: Frontend/src/pages/WorkflowListPage.tsx

### Round 2

5. **Workflow OperationsMap + shared utilities**
   - Commit: `b242576` (branch: round2/argo-ops-map)
   - Files: Frontend/src/lib/workflow-operations.ts, WorkflowDetailPage.tsx, WorkflowListPage.tsx
   - Also synced OpenAPI + api-guide + smoke script

6. **@ant-design/pro-flow canvas integration**
   - NOT committed yet (waiting deploy verification)
   - Files: Frontend/src/pages/PipelinePage.tsx, package.json

7. **Component Registry page** (in progress)
   - Worktree: /tmp/worktree-comp-registry
   - NOT committed yet
   - Backend CRUD + Frontend list page + modal

## How to Create Issues

Use Linear MCP to create issues in the cyber-databrew workspace. Each issue should:
- Title in Chinese describing the change
- Description with what was done, files changed
- Label: `bug` or `feature` as appropriate
- State: `Done` for completed work, `In Progress` for in-progress
- Add comment with commit SHA and link

## Steps

1. Check available Linear projects and workflow states via MCP
2. Create issues for each completed piece of work
3. Add comment linking the commit SHA
4. Set state to Done for completed items
5. Create issues for in-progress items (pro-flow, comp-registry) and set to In Progress

## Format

Issue title format: `CYB Round 2 - <feature description>` or just descriptive Chinese title
Description should include: what was done, why, key files changed, commit SHA

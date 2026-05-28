# Proposal — CYB-1344

## Why
Workflow detail DAG rendering loses dependency lines for omitted steps, so users cannot see that a failed upstream step caused a downstream step to be omitted.

## What Changes

### New Capabilities
- Pipeline workflow details expose normalized DAG edges from the backend so UI rendering does not infer workflow topology from partial runtime child links.

### Modified Capabilities
- Workflow detail DAG view renders backend-provided edges first, preserving logical dependencies such as `Failed -> Omitted`.

## Impact
- **Affected code**: `backend/internal/handlers/workflow`, `Frontend/src/pages/WorkflowDagView.tsx`, `Frontend/src/api/workflowApi.ts`
- **New APIs**: Changes response shape for `GET /api/v1/workflows/{name}` by adding `edges`
- **Dependencies**: No new dependencies expected

## Scope
- **In scope**: Normalize workflow DAG edges in the workflow detail response; update OpenAPI, API guide, frontend types, DAG rendering, and focused tests.
- **Out of scope**: Replacing React Flow, importing Argo UI GraphPanel, changing workflow execution semantics, or adding a separate workflow graph endpoint.

## Success Criteria
- [ ] A two-step workflow with step 1 `Failed` and step 2 `Omitted` returns a normalized edge from step 1 to step 2.
- [ ] The workflow detail DAG draws the edge between the failed and omitted nodes.
- [ ] Existing workflow detail callers remain compatible when `edges` is empty or absent.
- [ ] API contract documentation and frontend types match the backend response.

# Task: Analyze Argo Workflows UI Source Code for Porting

## Background

We need to port Argo Workflows v4.0.5 native UI features into our custom DataBrew pipeline UI. Currently our custom platform only covers ~30% of what Argo's native UI provides.

Our platform stack: Go (Gin) backend + React (Ant Design + React Flow) frontend. We use Argo Workflows REST API as the backend execution engine.

## Repository

The Argo source has been cloned to `/Users/rick/src/argo-workflows/` (branch v4.0.5).

## Task: Comprehensive Analysis

### Phase 1: Understand Argo UI Architecture

Analyze the Argo UI source code structure:

1. **UI directory:** `ui/` — React/TypeScript SPA
   - How is the routing structured?
   - What state management is used?
   - How does it communicate with the backend API?
   - What component library is used?

2. **Key pages/components:**
   - Workflow list page (`WorkflowsList.tsx` or equivalent)
   - Workflow detail page (with tabs: Summary, Events, Timeline, Workflow YAML)
   - Node detail panel
   - Log viewer
   - Template management
   - Cron workflow management

3. **API layer:**
   - How does the UI call the Argo API?
   - What endpoints are used for each feature?
   - Does it use GraphQL, REST, or something else?

### Phase 2: Feature Gap Analysis

Compare Argo native UI features vs what we've built. For each feature:

| Feature | Argo Has | We Have | What's Missing | API Available? |
|---------|----------|---------|----------------|----------------|
| Workflow list filtering | ✅ Rich filters | ✅ Status only | Labels, template, cron, date range, name search | Yes (same API) |
| Workflow summary counts | ✅ Top bar | ❌ | Running/Pending/Succeeded/Failed counts | Yes |
| Resubmit workflow | ✅ | ❌ | - | Yes (`POST /stop`, etc.) |
| Delete workflow | ✅ | ❌ | - | Yes |
| Workflow YAML view | ✅ Workflow tab | ❌ | YAML viewer | Yes (manifest in response) |
| Events tab | ✅ | ❌ | K8s events display | Maybe (k8s API) |
| Node parameters | ✅ Node details | ❌ | Container info, image, command, args, inputs/outputs | Yes (from workflow status) |
| Artifacts display | ✅ Toggle in DAG | ❌ | Input/output artifact display | Yes (from workflow status) |
| DAG search/filter | ✅ Search box | ❌ | Search within DAG nodes | - |
| DAG layout toggle | ✅ Horizontal/Vertical | ❌ | - | - |
| DAG expand/collapse | ✅ | ❌ | Collapse sub-DAGs | - |
| Log streaming | ✅ | ❌ (one-shot) | Real-time log updates | Yes (SSE/WebSocket?) |
| Template management | ✅ CRUD | ❌ | Template list/create/edit | Yes (Argo API) |
| Cron workflows | ✅ | ❌ | Schedule support | Yes |
| Namespace switching | ✅ | ❌ | Multi-namespace | Yes |

### Phase 3: Porting Strategy

For each missing feature, determine:

1. **Can we reuse Argo UI component source code?**
   - Argo uses React with TypeScript — same as our frontend
   - Can we extract specific components and adapt them?
   - What would need to change (styling, API endpoints, state management)?

2. **What backend changes are needed?**
   - Do we need new API endpoints?
   - Can we proxy through to the Argo API?
   - What workflow controller APIs are available?

3. **Effort estimate:**
   - Small (< 1 day): e.g., workflow summary counts, YAML view, delete button
   - Medium (1-3 days): e.g., rich filtering, node parameters, resubmit
   - Large (1-2 weeks): e.g., template management, cron, events

### Phase 4: Create Porting Plan

Write the analysis to `docs/review/argo-ui-porting-plan.md` with:

```markdown
# Argo UI Porting Plan

## Architecture Summary
(How Argo UI is structured, key learnings)

## Feature Migration Plan

### Phase A — Quick Wins (< 1 day each)
...

### Phase B — Core Missing Features (1-3 days each)
...

### Phase C — Advanced Features (1-2 weeks)
...

## Dependencies
(What must be done before what)

## Component Reuse Strategy
(Which Argo components can we extract, which need to be rebuilt)

## Backend API Changes Required
...
```

## Important Notes

- Do NOT modify any code in either repo — this is pure analysis
- Focus on the UI source code in `ui/`
- Argo's server/UI is in the main repo at https://github.com/argoproj/argo-workflows
- Our frontend is at `/Users/rick/cyber-databrew/frontend/` — compare component patterns
- Our backend uses `/api/v1/workflows/{namespace}` REST API to communicate with argo-server
- Our backend already has: `listWorkflows`, `getWorkflow`, `getWorkflowLogs`, `createWorkflow`, `deleteWorkflow`, `stopWorkflow` via argo client

## Tools

- Read files to analyze source code
- Compare with our codebase structure

## Deliverable

Write `docs/review/argo-ui-porting-plan.md` in the `/Users/rick/cyber-databrew/` repo.

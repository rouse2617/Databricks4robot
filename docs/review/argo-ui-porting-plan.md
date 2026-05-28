# Argo UI Porting Plan

> Analysis of Argo Workflows v4.0.5 native UI features vs. DataBrew custom pipeline UI,
> with backend gap analysis and implementation roadmap.
> Generated: 2026-05-28

---

## 1. Architecture Summary

### Argo UI (`ui/src/`)
- **Stack:** React 18 + TypeScript + custom `argo-ui` component library
- **State management:** None — React Context (popup/notifications/navigation) + hooks (useState/useEffect/useReducer) + localStorage for preferences + URL query params for page state
- **HTTP client:** `superagent` (wrapped in `shared/services/requests.ts`), plus browser `EventSource` (SSE) for real-time
- **Reactive streams:** RxJS for watch events, log streaming, and auto-reconnect (`list-watch.ts`, `retry-watch.ts`)
- **DAG layout:** `dagre` + custom layout algorithms (DFS sorter, Coffman-Graham sorter, pretty/fast layout)
- **YAML/JSON editing:** Monaco editor with JSON Schema autocompletion
- **Terminal logs:** xterm
- **Icons:** Font Awesome
- **CSS:** Foundation-style 12-column grid + custom SCSS
- **Module pattern:** Page-per-directory with `shared/components/`, `shared/services/`, `shared/models/`

### Our Frontend (`frontend/src/`)
- **Stack:** React 18 + TypeScript + Ant Design 5 + React Flow (@xyflow/react v12)
- **State management:** None — React Context (auth only) + hooks (useState/useEffect/useReducer)
- **HTTP client:** Axios (`api/client.ts`) + fetch (`api/pipelineClient.ts`)
- **Real-time:** None — polling refresh only
- **DAG layout:** React Flow (automatic layout)
- **YAML/JSON editing:** None (inline `<pre>` only)
- **Terminal logs:** None (plain `<pre>` blocks)
- **Icons:** `@ant-design/icons`
- **CSS:** Tailwind + Ant Design theme overrides via CSS variables
- **Module pattern:** Feature directories in `components/` + page-per-file in `pages/`

### Key Architectural Differences

| Aspect | Argo UI | Our UI | Porting Impact |
|--------|---------|--------|----------------|
| Component library | Custom argo-ui | Ant Design 5 | Cannot reuse Argo components directly; must reimplement with Ant Design |
| DAG/graph | dagre-based custom `GraphPanel` | React Flow | React Flow is more capable — port rendering logic, not the component |
| Real-time | RxJS + SSE EventSource | No real-time | Need to add EventSource support to our API layer |
| Logs | xterm terminal | Plain `<pre>` | Can skip xterm for MVP, but SSE streaming is essential |
| State serialization | URL query params | URL query params (assets only) | Pattern already exists in our codebase |
| Architecture | Page-per-directory | Page-per-file | N/A — structural choice, not blocking |

---

## 2. Feature Migration Plan

### Phase 0 — Backend Foundation (prerequisite, ~3 days)

The backend is the critical bottleneck. The Argo client interface and HTTP handlers must be extended before any frontend features work.

**Backend client additions** (`backend/internal/argo/client.go` — extend `WorkflowClient` interface):
- `RetryWorkflow(ctx, name, namespace)` → `PUT /api/v1/workflows/{ns}/{name}/retry`
- `ResubmitWorkflow(ctx, name, namespace)` → `PUT /api/v1/workflows/{ns}/{name}/resubmit`
- `SuspendWorkflow(ctx, name, namespace)` → `PUT /api/v1/workflows/{ns}/{name}/suspend`
- `ResumeWorkflow(ctx, name, namespace)` → `PUT /api/v1/workflows/{ns}/{name}/resume`
- `TerminateWorkflow(ctx, name, namespace)` → `PUT /api/v1/workflows/{ns}/{name}/terminate`

**Backend handler additions** (`backend/internal/handlers/workflow/handler.go`):
- `POST /api/v1/workflows/:name/retry` → calls RetryWorkflow
- `POST /api/v1/workflows/:name/resubmit` → calls ResubmitWorkflow
- `POST /api/v1/workflows/:name/suspend` → calls SuspendWorkflow
- `POST /api/v1/workflows/:name/resume` → calls ResumeWorkflow
- `POST /api/v1/workflows/:name/terminate` → calls TerminateWorkflow
- `DELETE /api/v1/workflows/:name` → calls DeleteWorkflow (already exists but not exposed on monitoring routes)

**Backend route additions** (`backend/routes/routes.go`, around line 343):
Add the new handlers alongside existing workflow routes:
```go
api.POST("/workflows/:name/retry", workflowHandler.RetryWorkflow)
api.POST("/workflows/:name/resubmit", workflowHandler.ResubmitWorkflow)
api.POST("/workflows/:name/suspend", workflowHandler.SuspendWorkflow)
api.POST("/workflows/:name/resume", workflowHandler.ResumeWorkflow)
api.POST("/workflows/:name/terminate", workflowHandler.TerminateWorkflow)
api.DELETE("/workflows/:name", workflowHandler.DeleteWorkflow)
```

**Response model upgrade** (`backend/internal/handlers/workflow/handler.go`):
The current response strips most workflow data. To support node details
(container info, parameters, artifacts), the response needs to include:
- `wf.Status.Nodes[*].Inputs` / `Outputs` (parameters, artifacts)
- `wf.Status.Nodes[*].TemplateName` → resolved template info
- `wf.Status.Nodes[*].ResourcesDuration`
- `wf.Status.Nodes[*].HostNodeName`
- `wf.Status.Nodes[*].Daemoned`
- `wf.Status.Nodes[*].MemoizationStatus`
- `wf.Status.Progress`
- `wf.Status.EstimatedDuration`
- `wf.Labels` / `wf.Annotations`

Currently the handler returns a hand-crafted JSON struct. Extend it to be
more comprehensive while still protecting against raw K8s type leakage.

**Optional: List endpoint upgrade** — support label selector and phase filters
by passing query params through to the Argo API.

---

### Phase A — Quick Wins (< 1 day each)

#### A1 Workflow Summary Counts Bar
- **Estimate:** 0.5 day
- **Backend impact:** None (data already available in `ListWorkflows` response)
- **Frontend:** Add a horizontal stats bar above the workflow list table showing counts per status (Running, Succeeded, Failed, Error, Pending)
- **Implementation:** Compute from `items` array client-side. Use Ant Design `Statistic` or `Card` components
- **Argo reference:** `workflows-summary-container/` — a pure UI calculation

#### A2 Delete Workflow Button
- **Estimate:** 0.5 day
- **Backend impact:** Add `DELETE /api/v1/workflows/:name` handler (the `WorkflowClient.DeleteWorkflow` method already exists)
- **Frontend:** Add a delete button to the workflow detail header and/or list row. Ant Design `Popconfirm` for confirmation
- **Argo reference:** Uses popup context with checkbox for "also delete from archive"

#### A3 Stop/Terminate Workflow Button
- **Estimate:** 0.5 day
- **Backend impact:** Add `POST /api/v1/workflows/:name/stop` and `POST /api/v1/workflows/:name/terminate` handlers (the client's `StopWorkflow` method already exists; `TerminateWorkflow` needs adding)
- **Frontend:** Add a stop/terminate button on the workflow detail page, visible only for Running workflows
- **Argo reference:** `workflow-operations-map.ts` — operations dynamically disabled based on workflow phase

#### A4 Workflow Name Search
- **Estimate:** 0.25 day
- **Backend impact:** None (client-side filtering or pass `nameFilter` query param)
- **Frontend:** Add an `Input.Search` bar above the workflow list table
- **Argo reference:** `InputFilter` in workflow-filters

#### A5 Workflow YAML Viewer
- **Estimate:** 0.5 day
- **Backend impact:** None (the raw Workflow CRD is already returned by `GetWorkflow`, but currently stripped to a custom struct). Need to extend handler to return serialized YAML (`wf.Spec` + `wf.Status`)
- **Frontend:** Show as a read-only code block. Can use Monaco editor (lazy-loaded) or a simpler `<pre>` block with syntax highlighting. Ant Design `Collapse` to separate sections
- **Argo reference:** `workflow-yaml-viewer/` — uses Monaco with serializing object editor

#### A6 Node Container Info Panel
- **Estimate:** 0.75 day
- **Backend impact:** Extend `GetWorkflow` response to include node template details (image, command, args, resources). The data is in `wf.Status.Nodes[*]` but currently stripped
- **Frontend:** Add a "容器" (Containers) tab in the node side panel showing resolved container/script spec
- **Argo reference:** `workflow-node-info/` — CONTAINERS tab

#### A7 Resubmit Workflow
- **Estimate:** 0.5 day
- **Backend impact:** Add `POST /api/v1/workflows/:name/resubmit` handler + `ResubmitWorkflow` client method
- **Frontend:** Add a resubmit button on the workflow detail page
- **Argo reference:** `resubmit-*` panels

#### A8 Retry Workflow
- **Estimate:** 0.5 day
- **Backend impact:** Add `POST /api/v1/workflows/:name/retry` handler + `RetryWorkflow` client method
- **Frontend:** Add a retry button, visible only for Failed/Error workflows
- **Argo reference:** `retry-*` panels

---

### Phase B — Core Missing Features (1-3 days each)

#### B1 Workflow List Rich Filtering
- **Estimate:** 2 days
- **Backend impact:** 
  - Extend `ListWorkflows` to support query params: `phases`, `labels`, `nameFilter`, `createdAfter`, `finishedBefore`
  - The Argo client's `ListWorkflows` already passes `listOptions.labelSelector` — extend to support additional query params
  - The `ListWorkflows` handler currently passes an empty label selector — wire up frontend filter values
- **Frontend:**
  - Sidebar filter panel with: phase checkboxes, label input/tags, date range picker
  - URL query param serialization for shareable filter state
  - Argo uses `WorkflowFilters` sidebar + `CheckboxFilter` + `InputFilter` + `TagsInput`
  - Our version: Ant Design `Checkbox.Group`, `DatePicker.RangePicker`, `Input` + `Tag` for labels
- **Argo reference:** `workflow-filters/` directory

#### B2 Real-Time Log Streaming (SSE)
- **Estimate:** 2-3 days
- **Backend impact:**
  - **Major change.** The current `GetWorkflowLogs` fetches the complete log at once (blocks until the Argo log stream ends). To support streaming:
  - Option A: Server-side SSE proxy — our backend opens an Argo log EventSource and streams NDJSON events to the frontend via SSE
  - Option B: Direct frontend → Argo Server SSE (requires exposing Argo Server URL to the browser — security concern)
  - Option A is preferred. The backend handler would:
    1. Open `GET /api/v1/workflows/{ns}/{name}/log` (SSE) from Argo
    2. Stream each NDJSON line as an SSE event to the frontend
    3. Handle reconnection logic
  - Note: The existing `parseLogStream` in `internal/argo/logs.go` concatenates all log content. For streaming, a new `StreamWorkflowLogs` method returning a channel or callback is needed
- **Frontend:**
  - Use browser `EventSource` or fetch-stream-read API
  - Progressive log display in a scrollable container
  - "Follow" toggle (auto-scroll to bottom)
  - Pod filter, container filter, grep/highlight
- **Argo reference:** `workflow-logs-viewer/` — full-featured SSE-based log viewer with xterm, pod filter, container filter, grep (all using RxJS Observable pattern)

#### B3 Node Input/Output Artifacts Display
- **Estimate:** 1 day
- **Backend impact:** Extend `GetWorkflow` response to include node input/output artifacts. The data exists in `wf.Status.Nodes[*].Outputs.Artifacts` and `Inputs.Artifacts`
- **Frontend:** Add "输入/输出" (Inputs/Outputs) tab in the node side panel showing parameters and artifacts with download links
- **Argo reference:** `workflow-node-info/` — INPUTS/OUTPUTS tab

#### B4 Workflow Events Tab
- **Estimate:** 1.5 days
- **Backend impact:** Add an endpoint that proxies K8s events for a workflow. Argo doesn't have a direct event endpoint for workflows — events come from the Kubernetes API. Options:
  - Add `GET /api/v1/workflows/:name/events` that calls `listOptions.fieldSelector=involvedObject.name={workflow-name}` on the K8s events API
  - Or configure the Argo Server to expose events (it has a `GET /api/v1/stream/events/{namespace}` endpoint)
- **Frontend:** New "事件" tab in workflow detail page showing K8s events in a table with Timestamp, Type, Reason, Message columns
- **Argo reference:** `events-panel/`

#### B5 Timeline View Improvement
- **Estimate:** 1 day
- **Backend impact:** None (data already available)
- **Frontend:** Our current timeline is a basic bar chart. Enhance with:
  - Better time axis (include date when spanning days)
  - Color-coded phases matching DAG
  - Click handling to select the node
  - Estimated duration for running nodes
  - Progress indicator for running nodes
- **Argo reference:** `workflow-timeline/` — also available as a separate `/timeline` route

#### B6 DAG Enhancements (Search, Layout Toggle, Artifacts)
- **Estimate:** 2 days
- **Backend impact:** None (purely frontend)
- **Frontend (React Flow enhancements):**
  - **Search/filter:** Add a search box above the DAG that highlights/filters nodes by name
  - **Layout toggle:** Add a toggle for horizontal/vertical layout (React Flow handles orientation via node positions — recalculate positions)
  - **Expand/collapse:** Group nested nodes (step groups, sub-DAGs) into collapsible group nodes
  - **Node coloring:** Color nodes by phase consistently with the list page
  - **Progress bars:** Show progress bar on running nodes
- **Argo reference:** `workflow-dag/` — dagre-based, search box, render options panel, legend

#### B7 DAG Node Details (Summary Tab Enrichment)
- **Estimate:** 1 day
- **Backend impact:** Extend `GetWorkflow` response with more node fields
- **Frontend:** Enhance the "详情" tab in the node side panel with:
  - Node type (Pod, StepGroup, DAG, Retry, Suspend, Skipped)
  - Duration (calculated)
  - Progress (if available)
  - Memoization status
  - Pod name
  - Host node name
  - Action buttons: LOGS, EVENTS (open respective panels)
- **Argo reference:** `workflow-node-info/` — SUMMARY tab

#### B8 Batch Selection + Toolbar
- **Estimate:** 1 day
- **Backend impact:** None (individual operations already covered)
- **Frontend:** Add checkbox selection to workflow list table rows, plus a batch action toolbar showing:
  - Delete selected
  - Retry selected (if all Failed/Error)
  - Resubmit selected
  - Terminate selected (if all Running)
- **Argo reference:** `workflows-toolbar/`

#### B9 Workflow Suspend/Resume
- **Estimate:** 0.5 day
- **Backend impact:** Add `POST /api/v1/workflows/:name/suspend` and `POST /api/v1/workflows/:name/resume` handlers
- **Frontend:** Add suspend/resume buttons on workflow detail page, disabled based on current state
- **Argo reference:** `workflow-operations-map.ts`

---

### Phase C — Advanced Features (3-10 days each)

#### C1 Workflow Template Management (Argo-native templates)
- **Estimate:** 5 days
- **Backend impact:**
  - Add Argo template CRUD proxying:
    - `GET /api/v1/workflow-templates/{namespace}` → list
    - `POST /api/v1/workflow-templates/{namespace}` → create
    - `GET /api/v1/workflow-templates/{namespace}/{name}` → get
    - `PUT /api/v1/workflow-templates/{namespace}/{name}` → update
    - `DELETE /api/v1/workflow-templates/{namespace}/{name}` → delete
  - Also cluster-scoped variants: `ClusterWorkflowTemplate`
- **Frontend:**
  - Template list page (similar to workflow list but for templates)
  - Template detail page with YAML viewer/editor
  - "Submit from template" flow
  - Note: This is distinct from our existing `PipelineTemplate` system. Our pipeline templates are DataBrew-specific DAGs; Argo templates are raw Workflow CRDs. These could be complementary — our templates deploy to Argo, Argo templates can be submitted directly
- **Argo reference:** `workflow-templates/` and `cluster-workflow-templates/`

#### C2 Submit Workflow
- **Estimate:** 1.5 days
- **Backend impact:** Add `POST /api/v1/workflows/:namespace/submit` handler that calls `POST /api/v1/workflows/{ns}/submit` on Argo (submits from a template)
- **Frontend:** A submit panel/form that allows:
  - Selecting a template (from Argo template list)
  - Overriding parameters
  - Setting labels/annotations
- **Argo reference:** `submit-workflow-panel/`, `workflow-creator/`

#### C3 Cron Workflow Management
- **Estimate:** 7-10 days
- **Backend impact:** Full CRUD for CronWorkflows:
  - `GET /api/v1/cron-workflows/{namespace}` → list
  - `POST /api/v1/cron-workflows/{namespace}` → create
  - `GET /api/v1/cron-workflows/{namespace}/{name}` → get
  - `PUT /api/v1/cron-workflows/{namespace}/{name}` → update
  - `DELETE /api/v1/cron-workflows/{namespace}/{name}` → delete
  - `PUT /api/v1/cron-workflows/{namespace}/{name}/suspend` → suspend
  - `PUT /api/v1/cron-workflows/{namespace}/{name}/resume` → resume
  - New client interface `CronWorkflowClient` or extend existing
- **Frontend:**
  - CronWorkflow list page with schedule display, suspend toggle, last scheduled time
  - CronWorkflow detail page with YAML editor
  - CronWorkflow create/edit with schedule validator and human-readable "next run" preview
  - Need a cron expression UI component (Argo uses `cron-parser` + `cronstrue` + custom `ScheduleValidator`)
- **Argo reference:** `cron-workflows/` — full CRUD with schedule display, suspend toggle

#### C4 Namespace Switching
- **Estimate:** 1 day
- **Backend impact:** Our handler currently hardcodes a single namespace. Need to support namespace parameter in routes
- **Frontend:** Add namespace selector dropdown in the nav bar or workflow page header, stored in localStorage. All workflow pages become namespace-aware
- **Argo reference:** `NamespaceFilter` component + URL-based namespace routing

#### C5 Workflow Archival Support
- **Estimate:** 2-3 days
- **Backend impact:** Add archived workflow endpoints:
  - `GET /api/v1/archived-workflows/{uid}` → get archived
  - `DELETE /api/v1/archived-workflows/{uid}` → delete archived
  - `GET /api/v1/archived-workflows` → list archived
- **Frontend:** Add a toggle to show archived workflows in the list, handle in detail page fallback (try live first, then archived)
- **Argo reference:** `getArchived`, `deleteArchived`, `retryArchived`, `resubmitArchived` in workflows-service.ts

#### C6 DAG Graph Render Options
- **Estimate:** 1 day
- **Backend impact:** None
- **Frontend:** A render options panel (could be a popover or collapsible section) with:
  - Show/hide artifacts as graph nodes
  - Show/hide template names
  - Template ref grouping toggle
  - Node size control
- **Argo reference:** `RenderOptionsPanel` in workflow-dag

#### C7 Event Streaming (Workflow Watch)
- **Estimate:** 3-4 days
- **Backend impact:**
  - SSE endpoint that proxies `GET /api/v1/workflow-events/{namespace}` from Argo Server (SSE stream of workflow status changes)
  - This is the foundation for real-time workflow list updates without polling
- **Frontend:**
  - Use browser `EventSource` to receive workflow status change events
  - Update workflow list in real-time (phase transitions, new workflows appearing)
  - Reconnect with backoff on connection loss
- **Argo reference:** `ListWatch` (list + watch pattern), `RetryWatch` (auto-reconnect)

#### C8 Log Archival / Artifact Logs
- **Estimate:** 2 days
- **Backend impact:** Add artifact download endpoint. Argo's `artifact-files/` path serves artifacts stored in S3/MinIO
- **Frontend:** Fall back to artifact logs when cluster logs are unavailable (completed workflows). Show download links for artifact files
- **Argo reference:** `getContainerLogsFromArtifact` in workflows-service.ts

---

## 3. Dependency Graph

```
Phase 0 (Backend Foundation)
  ├── A2 (Delete)
  ├── A3 (Stop/Terminate)
  ├── A7 (Resubmit)
  ├── A8 (Retry)
  ├── B2 (Log Streaming) — requires new SSE proxying
  ├── B9 (Suspend/Resume)
  ├── C1 (Templates)
  ├── C3 (Cron)
  └── C5 (Archival)

Phase A (Quick Wins — can start in parallel with Phase 0)
  ├── A1 (Summary Counts) — no backend deps
  ├── A4 (Name Search) — no backend deps
  ├── A5 (YAML Viewer) — requires handler response extension
  └── A6 (Node Container Info) — requires handler response extension

Phase B (Core Features)
  ├── B1 (Rich Filtering) — backend needs to wire up query params
  ├── B2 (Log Streaming) — depends on SSE proxy
  ├── B3 (Artifacts Display) — depends on handler response extension
  ├── B4 (Events) — depends on K8s events API proxying
  ├── B5 (Timeline Improvement) — no backend deps
  ├── B6 (DAG Enhancements) — no backend deps
  ├── B7 (Node Details) — depends on handler response extension
  ├── B8 (Batch Toolbar) — depends on individual operations
  └── B9 (Suspend/Resume) — depends on backend endpoints

Phase C (Advanced)
  ├── C1 (Templates) — new route group
  ├── C2 (Submit Flow) — depends on C1
  ├── C3 (Cron) — new route group + models
  ├── C4 (Namespace Switching) — touches all pages
  ├── C5 (Archival) — new route group
  ├── C6 (DAG Options) — no deps
  ├── C7 (Event Streaming) — SSE endpoint, foundational for real-time
  └── C8 (Artifact Logs) — depends on artifact storage setup
```

### Recommended sequencing:
1. **Sprint 1:** Phase 0 (backend foundation) + A1, A4 (no backend deps)
2. **Sprint 2:** A2, A3, A5, A6, A7, A8 (depends on Phase 0)
3. **Sprint 3:** B1, B5, B6, B7, B8 (core frontend features)
4. **Sprint 4:** B2, B3, B4, B9 (log streaming + events — high impact)
5. **Sprint 5:** C4, C6, C7 (real-time + multi-namespace)
6. **Sprint 6+:** C1, C2, C3, C5, C8 (full parity)

---

## 4. Component Reuse Strategy

### Can Extract (adapt to Ant Design):
| Argo Component | Adaptation Strategy | Effort |
|----------------|-------------------|--------|
| `WorkflowOperationsMap` | Pure logic — extract the operation definitions + enable/disable rules as a utility | 0.5 day |
| `ListWatch` pattern | Implement without RxJS — use EventSource + useState/useCallback instead | 1 day |
| `RetryWatch` pattern | Same — EventSource + reconnection logic | 0.5 day |
| `CronPrettySchedule` / `ScheduleValidator` | Pure display logic for cron expressions — adapt from `cronstrue` + `cron-parser` | 0.5 day |
| `WorkflowLabels` | Simple label rendering — adapt to Ant Design `Tag` components | 0.25 day |
| `Phase` / `PhaseIcon` | Phase color mapping + icon — adapt to Ant Design `Tag` | 0.25 day |
| `DurationPanel` | Duration formatting + progress — extract the formatting logic | 0.25 day |
| `PodName` resolution | Pure logic — extract name resolution from node ID patterns | 0.25 day |
| `TemplateResolution` | Pure logic — extract template resolution helpers | 0.5 day |
| `CronUtils` | Pure cron schedule computation | 0.25 day |
| `LinkifiedText` | URL auto-linking utility | 0.25 day |
| `Artifacts` helpers | Pure logic for artifact path resolution | 0.5 day |

### Must Rebuild (cannot reuse):
| Argo Component | Why | Our Alternative |
|----------------|-----|-----------------|
| `GraphPanel` / `WorkflowDag` | Built on dagre + canvas, React Flow is fundamentally different | Port layout/rendering logic to React Flow nodes |
| All `shared/components/` UI | Custom argo-ui SCSS, no Ant Design | Rebuild with Ant Design equivalents |
| `WorkflowFilters` | Custom checkbox/input components | Use Ant Design `Checkbox`, `DatePicker`, `Input` |
| `WorkflowLogsViewer` | xterm-based terminal | Plain `<pre>` for MVP, consider xterm later |
| `WorkflowYamlViewer` | Monaco-based | Use Monaco (already available in ecosystem) or `react-json-view` |
| `WorkflowNodeInfo` | Custom tabbed layout | Use Ant Design `Tabs` + `Descriptions` |
| `ObjectEditor` | Monaco + custom YAML/JSON schema | Use a lighter alternative or lazy-loaded Monaco |
| `SliderPanel` | argo-ui sliding panel | Use Ant Design `Drawer` |
| `WorkflowsToolbar` | Custom toolbar | Use Ant Design `Space` + `Button` |
| Namespace handling | URL-based routing | Add namespace to URL or use Ant Design `Select` in header |

### Interesting Patterns to Reimplement:
- **WorkflowOperationsMap** (`workflow-operations-map.ts`): A clean declarative pattern. Each operation has a `title`, `iconClassName`, `disabled(wf)` predicate, and `action(wf)` function. This could be directly adapted to our codebase as a utility that returns available actions with Ant Design button props. **Definitely worth porting.**
- **ListWatch** (`list-watch.ts`): The Kubernetes-style list + watch pattern. Initial list fetch, then subscribe to watch events for real-time updates. Reconnect on failure. **Port to React-friendly version without RxJS.**

---

## 5. API Coverage Matrix

### Current Backend API — Workflow Endpoints
| Method | Path | Status |
|--------|------|--------|
| GET | `/api/v1/workflows` | ✅ Implemented (strips most fields) |
| GET | `/api/v1/workflows/:name` | ✅ Implemented (strips most fields) |
| GET | `/api/v1/workflows/:name/logs` | ✅ Implemented (one-shot) |
| DELETE | `/api/v1/deployments/:id` | ✅ Indirect via pipeline handler |
| POST | `/api/v1/deployments/:id/stop` | ✅ Indirect via pipeline handler |

### Missing Backend Endpoints — Priority Ordered

**High Priority (Phase 0 / needed for Phase A):**
| Method | Path | Argo API | For Feature |
|--------|------|----------|-------------|
| DELETE | `/api/v1/workflows/:name` | `DELETE /api/v1/workflows/{ns}/{name}` | A2 Delete |
| POST | `/api/v1/workflows/:name/stop` | `PUT /api/v1/workflows/{ns}/{name}/stop` | A3 Stop |
| POST | `/api/v1/workflows/:name/terminate` | `PUT /api/v1/workflows/{ns}/{name}/terminate` | A3 Terminate |
| POST | `/api/v1/workflows/:name/retry` | `PUT /api/v1/workflows/{ns}/{name}/retry` | A8 Retry |
| POST | `/api/v1/workflows/:name/resubmit` | `PUT /api/v1/workflows/{ns}/{name}/resubmit` | A7 Resubmit |

**Medium Priority (Phase B):**
| Method | Path | Argo API | For Feature |
|--------|------|----------|-------------|
| POST | `/api/v1/workflows/:name/suspend` | `PUT /api/v1/workflows/{ns}/{name}/suspend` | B9 Suspend |
| POST | `/api/v1/workflows/:name/resume` | `PUT /api/v1/workflows/{ns}/{name}/resume` | B9 Resume |
| GET | `/api/v1/workflows/:name/logs/stream` | (SSE proxy) `GET .../log` with `follow=true` | B2 Log Streaming |
| GET | `/api/v1/workflows/:name/events` | K8s events API or Argo events stream | B4 Events |

**Low Priority (Phase C):**
| Method | Path | Argo API | For Feature |
|--------|------|----------|-------------|
| GET | `/api/v1/workflow-templates/{namespace}` | Argo template CRUD | C1 Templates |
| POST | `/api/v1/workflow-templates/{namespace}` | Argo template CRUD | C1 Templates |
| GET/PUT/DELETE | `/api/v1/workflow-templates/{namespace}/{name}` | Argo template CRUD | C1 Templates |
| GET | `/api/v1/cluster-workflow-templates` | Cluster scope templates | C1 Templates |
| GET/POST/PUT/DELETE | `/api/v1/cron-workflows/{namespace}` | Argo cron CRUD | C3 Cron |
| POST | `/api/v1/workflows/{namespace}/submit` | Submit from Argo template | C2 Submit |
| GET | `/api/v1/archived-workflows` | Archived list | C5 Archival |
| GET/DELETE | `/api/v1/archived-workflows/{uid}` | Archived CRUD | C5 Archival |
| GET | `/api/v1/workflow-events/{namespace}` | Workflow watch stream | C7 Event Streaming |

### Backend Response Enrichment (non-endpoint changes)

The `ListWorkflows` and `GetWorkflow` handlers need response structure upgrades:

**ListWorkflows — add per-item fields:**
- `labels` (map)
- `startedAt` (timestamp)
- `estimatedDuration` (int)
- `progress` (string, e.g. "10/20")

**GetWorkflow — add per-node fields:**
- `type` (string — "Pod", "StepGroup", "DAG", "Retry", "Suspend", "Skipped")
- `inputs` — parameters, artifacts
- `outputs` — parameters, artifacts  
- `templateName` / `templateScope`
- `resourcesDuration` (ResourceDuration map)
- `hostNodeName` (string)
- `memoizationStatus` (object)
- `progress` (string)
- `estimatedDuration` (int64)
- `children` ([]string)
- `boundaryID` (string)
- `daemoned` (bool)

---

## 6. Frontend File Map

### New API Module
```
frontend/src/api/workflowApi.ts   ← EXTEND existing
  - addWorkflowOperation(name, operation)  // retry, resubmit, etc.
  - deleteWorkflow(name)
  - stopWorkflow(name)
  - terminateWorkflow(name)
  - retryWorkflow(name)
  - resubmitWorkflow(name)
  - suspendWorkflow(name)
  - resumeWorkflow(name)
  - streamWorkflowLogs(name, nodeId)  // returns EventSource
```

### New/Modified Pages
```
frontend/src/pages/
  WorkflowListPage.tsx           ← EXTEND: summary bar, rich filters, search, batch actions
  WorkflowDetailPage.tsx         ← EXTEND: operations menu, events tab, DAG enhancements

frontend/src/components/workflows/       ← NEW directory (mirror Argo pattern)
  SummaryBar.tsx                          — status count cards
  RichFilters.tsx                         — phase checkboxes, date range, labels
  OperationsMenu.tsx                      — retry/resubmit/suspend/resume/stop/terminate/delete
  NodeInfoPanel.tsx                       — SUMMARY, CONTAINERS, INPUTS/OUTPUTS tabs
  LogStreamViewer.tsx                     — streaming log viewer with pod/container/grep filters
  YAMLViewer.tsx                          — read-only YAML display
  EventsPanel.tsx                         — K8s events table
  TimelineView.tsx                        — enhanced timeline (replace inline component)
  BatchToolbar.tsx                        — batch selection actions
  WorkflowOperations.ts                   — extracted operations map utility
```

### Shared Utilities
```
frontend/src/lib/
  workflowPhase.ts                        — phase color/icon mappings, status helpers
  workflowOperations.ts                   — WorkflowOperationsMap pattern from Argo
  argoStreaming.ts                        — EventSource wrapper + reconnection
  workflowDuration.ts                     — duration formatting
```

---

## 7. Risk Assessment

### High Risk (with mitigation):
1. **SSE streaming via backend proxy** — The backend needs a fundamentally new pattern (streaming HTTP responses). The current codebase uses Gin with standard request-response. Mitigation: Gin supports `c.Stream()` with `text/event-stream` content type. Can be implemented as a new handler pattern.

2. **Real-time workflow watch** — Adding SSE for workflow events requires careful reconnection handling. Mitigation: The Argo UI uses a proven pattern (ListWatch with RxJS). We can implement a simpler version with EventSource + manual reconnection.

3. **Backend response enrichment** — The current handlers carefully strip most fields to avoid leaking K8s types. Adding more fields needs careful consideration of what to expose. Mitigation: Keep the current response struct approach but add the needed fields explicitly, avoiding any raw wfv1 type leakage.

### Medium Risk:
4. **Large file state management** — The workflow detail page needs to track: selected node, active tab, log state, side panel state, operation loading states. URL query param serialization would be ideal but adds complexity. Mitigation: Use React useState for the detail page (simpler than URL sync for this view).

5. **Ant Design + React Flow integration** — Ant Design `Tabs` wrapping a `ReactFlowProvider` can have rendering issues. Mitigation: Already working in the current detail page — pattern is validated.

### Low Risk:
6. **No RxJS in our stack** — Argo relies heavily on RxJS for streaming. We don't use it. Mitigation: Modern browser APIs (EventSource, async iterators) can replace RxJS for our use cases without adding a dependency.

7. **Reusing Argo models** — Argo uses `wfv1` Go types and their own TypeScript model interfaces. Mitigation: Define our own focused TypeScript interfaces (like we already do with `WorkflowSummary` and `WorkflowDetail`).

---

## 8. Summary

| Category | Backend Changes | Frontend Changes | Effort |
|----------|----------------|-------------------|--------|
| Phase 0 (Backend Foundation) | New client methods + handlers + routes | None | ~3 days |
| Phase A (8 quick wins) | Minimal handler additions | New components in existing pages | ~4 days |
| Phase B (9 core features) | Streaming + response enrichment | New page sections + components | ~13 days |
| Phase C (8 advanced features) | New route groups + models | New pages + complex components | ~25 days |
| **Total** | | | **~45 days (9 sprints)** |

The most impactful features relative to effort are:
1. **A2/A3 - Delete/Stop** (1 day combined) — enables basic workflow lifecycle management
2. **A1 - Summary Counts** (0.5 day) — instant UX improvement
3. **B2 - Log Streaming** (2-3 days) — transforms debugging capability from painful to smooth
4. **A5 - YAML Viewer** (0.5 day) — enables deep inspection without external tools
5. **B1 - Rich Filtering** (2 days) — essential as workflow count grows
6. **C7 - Event Streaming** (3-4 days) — eliminates manual refresh, foundational for "live" feel

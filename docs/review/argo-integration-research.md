# Argo Workflows Integration Research

> **Date:** 2026-05-28
> **Purpose:** Research how teams/products embed Argo Workflows into custom UIs and platforms, and provide technical recommendations for the cyber-databrew stack (React + Ant Design + Go).

---

## 1. Summary of Integration Patterns

There are three dominant patterns for integrating Argo Workflows into a custom platform:

### Pattern A: Full Custom UI via Argo API (Recommended for cyber-databrew)

Build a standalone React UI that talks to a Go backend, which proxies the Argo Workflows API.

**How it works:**
- Go backend authenticates with the Kubernetes API (service account or kubeconfig)
- Backend wraps Argo's REST API — list workflows, submit workflows, get logs, watch events
- Frontend never talks to Argo directly; all requests go through the platform backend
- The platform handles auth, authorization, and multi-tenancy

**Pros:**
- Full control over UI/UX — no dependency on Argo's UI components
- Unified auth — users authenticate once to the platform, not separately to Argo
- Can add platform-specific features (approval gates, cost tracking, custom metrics)
- No CORS issues — single backend origin
- The Argo Workflows API is stable and well-documented (gRPC + REST)

**Cons:**
- More development effort to build and maintain workflow views
- Need to stay in sync with Argo API changes
- Must re-implement workflow visualization (DAG rendering)

**Who uses it:** Equinor Flowify, most internal platforms, Kubeflow Pipelines (before v2 decoupling)

### Pattern B: Embed Argo UI via Iframe

Serve the native Argo Workflows UI in an iframe within your platform.

**How it works:**
- Deploy the standard argo-server alongside your platform
- Embed the Argo UI in an iframe, passing auth tokens via URL parameters or cookies
- Use postMessage API for cross-frame communication

**Pros:**
- Zero UI development — get all Argo features (logs, DAG view, YAML editing) for free
- Automatic updates when Argo releases new UI versions
- Full workflow debugging capabilities included

**Cons:**
- Clunky UX — the Argo UI has different styling, navigation, and UX patterns
- SSO redirect issues in iframes (callback URLs, cookie blocking)
- Limited customization — Argo UI has no official theming API (open issues: #7054, #10447)
- Base-href and ingress configuration can be finicky
- User confusion from having two separate UI paradigms

**Tradeoff:** Lower dev cost, lower UX quality.

### Pattern C: Hybrid — Custom UI + Argo UI for specific views

Use a custom UI for workflow management while linking to the native Argo UI for detailed views (logs, YAML, DAG).

**How it works:**
- Custom list views, submission forms, and status dashboards in your platform
- Deep-link to the Argo UI for workflow details, logs, and DAG visualization
- Usually combined with Pattern A — the custom UI calls the API for summary data

**Pros:**
- Balance of custom UX and Argo's specialized views
- Users can access Argo's superior DAG visualization and log browsing
- Less custom code needed for complex views

**Cons:**
- Context switching for users between two UIs
- Two sets of auth — need SSO between platform and Argo
- Still need to configure and maintain the Argo UI deployment

**Who uses it:** Many internal platforms at mid-to-large companies.

---

## 2. Reference Implementations Worth Studying

### 2.1 Equinor Flowify — Closest to Our Stack

> **Stack:** Go backend + React UI + Argo Workflows
> **GitHub:** [equinor/flowify-workflows-server](https://github.com/equinor/flowify-workflows-server) · [flowify-workflows-UI](https://github.com/equinor/flowify-workflows-UI)

Equinor's Flowify is an open-source, no-code workflow manager built on top of Argo Workflows. It is the closest reference to our architecture:

- **Go backend** that communicates with the Argo Workflows API, handles workflow transpilation (visual graph → Argo YAML), and manages secrets/volumes
- **React frontend** with a drag-and-connect visual workflow builder
- Two-way communication with Argo for workflow execution status
- Workspace-based access control for multi-tenancy

**Key takeaway:** The Go backend acts as an intermediary — it doesn't expose the Argo API directly but wraps it with platform-specific logic.

### 2.2 Argo Workflows Native UI — Reference Architecture

> **Source:** [argoproj/argo-workflows/ui/src](https://github.com/argoproj/argo-workflows/tree/master/ui/src)

The native Argo UI is a React + TypeScript app with these patterns worth studying:

- **Feature-based directory structure** — each domain gets its own folder (workflows/, cron-workflows/, workflow-templates/, sensors/, event-sources/)
- **SSE-based real-time updates** — uses browser `EventSource` to receive workflow events at `/workflow-events`
- **URL-driven filter state** — workflow filters persisted in URL query parameters and localStorage
- **Direct REST calls** to the Argo Server API (port 2746) — no GraphQL layer
- **`argo-ui` component library** — a custom internal library (not Ant Design or MUI)
- **Immutable state patterns** — cloned before mutation to avoid subtle bugs
- **`useMemo` for derived state** — avoids unnecessary re-renders

**Key takeaway:** The native Argo UI is straightforward React; there is no complex state management library (Redux/Zustand). It relies on React hooks and URL state.

### 2.3 Apache Airflow UI — Modern React Workflow UI

> **Source:** [apache/airflow](https://github.com/apache/airflow) — React frontend

Airflow's modern UI (v3.x+) uses:
- **React + TypeScript + Chakra UI** (not Ant Design, but similar component library pattern)
- **React Flow (`@xyflow/react`)** for DAG graph visualization — this is a key library for workflow DAGs
- **TanStack React Query** for server state and caching
- **OpenAPI code generation** (`@hey-api/openapi-ts`) for type-safe API clients
- **Plugin system (AIP-68)** — allows injecting React components into specific UI locations

**Key takeaway:** React Flow is the industry standard for DAG visualization in workflow UIs (used by Airflow, Temporal community, and many others).

### 2.4 Ant Design ProFlow — Most Direct Tech Match

> **GitHub:** [ant-design/pro-flow](https://github.com/ant-design/pro-flow) — MIT License

This is an official Ant Group project that brings flow-based UI to the Ant Design ecosystem:

- Built on top of **React Flow** with **Ant Design-styled** nodes and components
- **Dagre auto-layout** built in
- `FlowView` and `FlowEditor` components
- Out-of-the-box: MiniMap, Inspector, copy/paste, undo/redo
- Custom nodes/edges with Ant Design styling

**Key takeaway:** For our React + Ant Design + DAG visualization needs, ProFlow is the most natural choice. It combines React Flow's power with Ant Design's look and feel.

### 2.5 visual-argo-workflows — Simple Reference Implementation

> **GitHub:** [omhq/visual-argo-workflows](https://github.com/omhq/visual-argo-workflows) — MIT License

A smaller, simpler open-source project that provides a visual drag-and-drop interface for creating Argo Workflows:

- **TypeScript + React** (similar to our stack)
- Visual construction → Argo YAML generation
- ~100 GitHub stars, MIT licensed
- Runs locally via `npm start` or Docker

**Key takeaway:** A good starting point for understanding how to translate visual workflow graphs into Argo Workflow YAML.

---

## 3. Competitive Analysis — How Similar Platforms Handle Workflow UI

| Feature | Argo Workflows | Airflow (v3+) | Kubeflow Pipelines | Prefect | Dagster | Temporal |
|---------|---------------|---------------|-------------------|---------|---------|----------|
| **Frontend** | React + argo-ui | React + Chakra UI | React + MUI | React | React + Styled Components | React |
| **DAG Viz** | Custom SVG | React Flow | React Flow | Custom SVG | Custom SVG | Timeline view |
| **Real-time** | SSE (EventSource) | Polling | Polling / WebSocket | SSE | GraphQL polling/subscriptions | WebSocket |
| **API Layer** | REST + gRPC | REST (FastAPI) | REST + gRPC | GraphQL | GraphQL | gRPC |
| **Multi-tenant** | Not native | RBAC + folders | Namespace-based | Workspaces | Code locations | Namespaces |
| **DB Backend** | etcd (CRDs) | PostgreSQL (metadata) | MySQL/PostgreSQL | PostgreSQL | PostgreSQL | Cassandra/PostgreSQL |
| **Log Viewer** | Native in UI | Native in UI | Native in UI | Native in UI | Native in UI | Web UI + CLI |
| **Component Library** | Custom (argo-ui) | Chakra UI | MUI | Custom | Custom | Custom |

### Key Differences and Patterns:

1. **Airflow's approach to DAG visualization** is the most mature — dedicated Graph, Grid, and Gantt views, with React Flow as the rendering engine.

2. **Kubeflow Pipelines** historically relied on Argo as its execution engine but decoupled in v2 with its own Intermediate Representation. Its UI provides ML-specific views (artifact browsing, experiment comparison) that Argo lacks.

3. **Dagster and Prefect** both use React frontends with modern tooling. Dagster uses GraphQL (Apollo Client) for all frontend-backend communication, while Prefect uses SSE for real-time run updates.

4. **Temporal** takes a different approach — workflow-as-code with less focus on DAG visualization and more on timeline/summary views.

---

## 4. Technical Recommendations for Our Stack

### 4.1 Recommended Architecture: Full Custom UI (Pattern A)

We recommend **Pattern A (Full Custom UI via Go backend)** for cyber-databrew:

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│  React + Ant  │────▶│  Go Backend  │────▶│  Argo API    │────▶│  K8s / Workflow  │
│  Design UI    │◀────│  (proxy)     │◀────│  (REST/gRPC) │◀────│  Controller      │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────────┘
```

**Rationale:**
- Our stack (React + Ant Design + Go) is well-suited to Pattern A
- We already have the Go backend infrastructure
- Ant Design provides a mature component library for data tables, forms, layouts
- ProFlow (@ant-design/pro-flow) gives us Ant Design-styled DAG visualization
- Full control over multi-tenancy and authorization

### 4.2 DAG Visualization

Use **React Flow (`@xyflow/react`)** with **ProFlow (@ant-design/pro-flow)**:

```tsx
import { FlowView } from '@ant-design/pro-flow';

// ProFlow provides Ant Design-styled nodes with dagre auto-layout
function WorkflowDAG({ workflow }) {
  const { nodes, edges } = useWorkflowToGraph(workflow);
  return <FlowView nodes={nodes} edges={edges} />;
}
```

Alternative: Use React Flow directly if we need more customization — it's what Airflow, Kubeflow, and most workflow UIs use.

### 4.3 Real-Time Updates via SSE

Argo Workflows supports SSE at `/api/v1/workflow-events/{namespace}` for receiving workflow status updates. For our Go backend:

1. **Backend:** Open an SSE connection to the Argo API, stream events to the frontend
2. **Frontend:** Use React's `useSyncExternalStore` or a custom hook to manage the EventSource connection
3. **Connection management:** Implement exponential backoff (start 1s, max 30s), track connection state for UI feedback (green/red indicator)

```typescript
// Pattern: useSyncExternalStore for SSE (from SSE research)
function createSSEStore<T>(url: string, parser?: (e: MessageEvent) => T) {
  let data: T[] = [];
  let eventSource: EventSource | null = null;
  const listeners = new Set<() => void>();

  const subscribe = (listener: () => void) => {
    listeners.add(listener);
    if (!eventSource) { /* connect */ }
    return () => { /* cleanup */ };
  };

  return () => useSyncExternalStore(subscribe, () => data);
}
```

**SSE Protocol Best Practices:**
- Always include `id` field for `Last-Event-ID` resume on reconnect
- Use single-line compact JSON in `data:` fields (avoid chunk truncation)
- Set `Cache-Control: no-cache` on SSE responses
- Cap accumulated events in memory (e.g., 500 max) to prevent leaks
- Use `@microsoft/fetch-event-source` if POST + custom headers are needed

### 4.4 Proxying Argo API

Our Go backend should proxy relevant Argo API endpoints rather than exposing Argo directly:

| Endpoint | Purpose | Method |
|----------|---------|--------|
| `GET /api/v1/workflows/{namespace}` | List workflows | GET |
| `POST /api/v1/workflows/{namespace}` | Submit workflow | POST |
| `GET /api/v1/workflow-events/{namespace}` | Watch workflow events | SSE |
| `GET /api/v1/workflows/{namespace}/{name}` | Get workflow details | GET |
| `GET /api/v1/workflows/{namespace}/{name}/log` | Stream pod logs | SSE |

**Auth pattern:**
- Backend uses a Kubernetes service account with RBAC for Argo API access
- Backend validates user JWT/session, then proxies the request with the SA credential
- No direct user authentication to Argo — Argo API is only accessible via the Go backend

### 4.5 Multi-Tenancy Strategy

For cyber-databrew's use case (per-user pipeline execution):

**Recommended: Labels-per-namespace (single namespace with label-based isolation)**

- Deploy Argo Workflows in a shared namespace (e.g., `argo-workflows`)
- Label all workflows with `owner: <user-id>` and `team: <team-id>`
- The Go backend filters by these labels when listing workflows
- More efficient resource usage than namespace-per-user
- Simpler monitoring — one namespace to watch

**Alternative — Namespace-per-user:** True Kubernetes isolation, but higher overhead (CRDs must exist in each namespace, more cluster resources). Use only if strong security isolation is required.

### 4.6 Ant Design + Real-Time Data Patterns

For real-time workflow data in Ant Design components:

- **Ant Design Table** with polling or SSE updates for workflow lists
- **Ant Design Badge** with status colors for workflow state indicators
- **Ant Design Progress** for running workflow completion tracking
- **ProLayout** from Ant Design Pro for the overall dashboard layout
- **ProTable** for workflow lists with built-in filtering, sorting, and pagination
- **Ant Design Charts** (or Recharts) for workflow execution metrics

**Connection state feedback pattern:**
```tsx
import { Badge, Tag } from 'antd';

function SSEStatus({ status }: { status: 'connected' | 'reconnecting' | 'error' }) {
  const colorMap = { connected: 'green', reconnecting: 'orange', error: 'red' };
  return <Badge status={colorMap[status] as any} text={status} />;
}
```

---

## 5. Pitfalls to Avoid

### 5.1 Direct Frontend-to-Argo API calls

**Problem:** Exposing the Argo API directly to the browser creates CORS issues, auth complexity, and security surface area.

**Fix:** Always proxy through the Go backend. The frontend should never know about the Argo API URL.

### 5.2 SSE Connection Timeouts

**Problem:** The Argo UI has a known issue where SSE connections drop after 60s (NGINX default keepalive timeout). The JS `EventSource` reconnection logic doesn't always fire because HTTP 200 is returned instead of an error. (GitHub issue #4301)

**Fix:** Implement explicit reconnection in the backend: set appropriate keepalive timeouts, detect closed connections, and reconnect proactively. On the frontend, treat any SSE close as a reconnect trigger.

### 5.3 Over-reliance on etcd for Workflow History

**Problem:** Argo stores workflow state in etcd (via CRDs), which has a 1MB object size limit and no built-in archival. Listing many workflows becomes slow.

**Fix:** Workflow archival:
- Configure Argo's workflow archive with a PostgreSQL/MySQL database
- The Go backend should query the archive DB for historical data and CRD API for active workflows
- Set up TTL on completed workflows to auto-archive from etcd

### 5.4 Assuming the Native Argo UI Can Be Themed

**Problem:** The Argo UI has no official theming API. Issues #7054 (skinnable/themable UI) and #10447 (custom navigation links) are open but not merged.

**Fix:** If going with Pattern B or C, be prepared for a UI that doesn't match your platform's look and feel. Consider reverse-proxy injection of custom CSS as a workaround, or accept the mismatch.

### 5.5 Ignoring Argo's SSO Callback Conflicts

**Problem:** If using SSO with Argo Workflows behind a reverse proxy, the callback path (`/oauth2/callback`) can conflict with `oauth2-proxy` (Issue #11252).

**Fix:** If proxying Argo (Pattern B or C), configure the reverse proxy to exempt Argo's callback path from the outer auth layer, or use `--base-href` to host Argo at a sub-path.

### 5.6 Blocking the Main Thread with Large Workflow Lists

**Problem:** Fetching and rendering hundreds of workflows can block the React main thread.

**Fix:**
- Use virtual scrolling (`react-window` or Ant Design Table virtual scroll)
- Paginate API requests (Argo API supports pagination via `pageSize` and `pageToken`)
- Use `useMemo` for filtered/sorted lists
- Only fetch active workflows on initial load; archive older ones

### 5.7 Inefficient Polling vs SSE

**Problem:** Polling the Argo API for workflow status updates creates unnecessary load on etcd and the API server.

**Fix:** Use SSE for real-time updates whenever possible. Fall back to polling only for operations that don't support streaming (e.g., listing archived workflows).

---

## 6. Key Tools and Libraries Summary

| Need | Recommended Library | Why |
|------|-------------------|-----|
| DAG Visualization | `@xyflow/react` + `@ant-design/pro-flow` | Industry standard for React DAGs; ProFlow adds Ant Design styling |
| Real-Time SSE | `useSyncExternalStore` (React built-in) | No external deps; works with any EventSource |
| Workflow Data Table | Ant Design `ProTable` | Built-in filtering, sorting, pagination, column customization |
| Dashboard Layout | Ant Design `ProLayout` | Consistent with our Ant Design stack |
| Charts / Metrics | Apache ECharts (via Ant Design Charts) or Recharts | Recharts is simpler; ECharts is more powerful for complex dashboards |
| SSE POST Support | `@microsoft/fetch-event-source` | Only if we need custom headers or POST-based SSE |
| Workflow Archival | Argo's built-in PostgreSQL archiver | Avoid etcd limitations for historical data |

---

## 7. Sources

- [Argo Workflows GitHub](https://github.com/argoproj/argo-workflows)
- [Argo Server Documentation](https://argo-workflows.readthedocs.io/en/latest/argo-server/)
- [Equinor Flowify](https://github.com/equinor/flowify-workflows-server)
- [Ant Design ProFlow](https://github.com/ant-design/pro-flow)
- [visual-argo-workflows](https://github.com/omhq/visual-argo-workflows)
- [Apache Airflow UI Architecture](https://deepwiki.com/apache/airflow/9.2-release-process)
- [Kubeflow Pipelines UI Modernization](https://blog.kubeflow.org/modernizing-kubeflow-pipelines-ui/)
- [Dagster Frontend Architecture](https://deepwiki.com/dagster-io/dagster/7.1-frontend-architecture)
- [MLflow Web UI Architecture](https://deepwiki.com/mlflow/mlflow/14.3-web-ui-architecture)
- [Argo Workflows UI Customization Issue #7054](https://github.com/argoproj/argo-workflows/issues/7054)
- [Argo Workflows UI Custom Columns Issue #10447](https://github.com/argoproj/argo-workflows/issues/10447)
- [Argo Workflows SSE Timeout Issue #4301](https://github.com/argoproj/argo-workflows/issues/4301)
- [Argo Workflows SSO Callback Issue #11252](https://github.com/argoproj/argo-workflows/issues/11252)
- [Platform Engineering on K8s with Argo (Musana Engineering)](https://musana.engineering/platform-engineering-on-k8s-part2/)
- [SSE Best Practices for React (dev.to)](https://dev.to/raxxostudios/server-sent-events-beat-websockets-for-80-of-my-ai-streaming-uis-5-patterns-49ac)

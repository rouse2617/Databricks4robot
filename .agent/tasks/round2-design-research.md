# Task: UX/UI/Product Design Research — Porting Argo Features

We need to port Argo Workflows v4.0.5 native UI features into our custom DataBrew pipeline platform. Before coding Round 2, we need a solid design thinking exercise.

## Context

We have a React + Ant Design 5 frontend with a React Flow DAG canvas for pipeline creation. Currently we have:
- Workflow list page (basic table + status filter)
- Workflow detail page (React Flow DAG + timeline + per-node log viewer)
- Pipeline creation canvas (drag-and-drop components)

We need to add: workflow operations (delete/stop/retry/resubmit/suspend/resume), rich filtering, YAML viewer, node details, log streaming, etc.

## What to Research & Deliver

### 1. Interaction Design for Workflow Operations

Where should operation buttons live? Options:
- **Detail page header** (like Argo does: RESUBMIT / DELETE / LOGS / SHARE buttons)
- **List page row actions** (expandable actions per workflow)
- **Context menu** (right-click on DAG node? long-press?)
- **Batch toolbar** (select multiple → act on all)

Deliver a recommended layout with pros/cons for each.

### 2. Filter UI Design

Argo has a sidebar with many filter types. How should we present them?
- Sidebar panel (like Argo)
- Collapsible section above table (like Jira)
- Popover/drawer
- Which filters genuinely matter for our users?

### 3. Node Detail Panel Enhancement

Currently shows: name, status, message, startedAt, finishedAt.
Need to add: container info (image, command, args), parameters, inputs/outputs, artifacts.

Design the enhanced panel layout. What tabs? What info per tab?

### 4. Log Viewer Design

Current: plain `<pre>` block in a tab panel.
Needed: real-time streaming, follow mode, pod/container filter.

Design approach: start simple and iterate, or full xterm integration like Argo?

### 5. Information Architecture

Where do these new pages/features fit in the existing navigation?
- Workflow templates page? (new nav item)
- Cron workflows page? (new nav item)
- Or embed as tabs within existing pages?

### 6. Design Principles (most important)

Define 3-5 design principles that should guide ALL feature development. Examples:
- "User actions should be two clicks or fewer"
- "Every error state should have a recovery path"
- "Node names in Argo must match canvas names exactly"

## Deliverable

Write a comprehensive design brief to `docs/review/round2-design-brief.md` in the repo at `/Users/rick/cyber-databrew/`. Include:
- Wireframe-style layout recommendations (ASCII diagrams or descriptions)
- Design principles
- Component hierarchy (what goes where)
- User flow diagrams for key operations
- Priority recommendations (build order within Round 2)

## Tools
- Read our existing frontend code at `/Users/rick/cyber-databrew/frontend/src/`
- Read the Argo porting plan at `/Users/rick/cyber-databrew/docs/review/argo-ui-porting-plan.md`
- Read our UX report at `/Users/rick/cyber-databrew/docs/review/ux-analysis-report.md`
- Read our product analysis at `/Users/rick/cyber-databrew/docs/review/product-perspective-analysis.md`
- Use web search for inspiration from similar platforms

# UX/UI Analysis Report — Pipeline Flow

> Generated: 2026-05-28
> Method: Browser inspection (Chrome DevTools) + Code analysis (WorkflowDetailPage.tsx, WorkflowListPage.tsx, Pipeline handler + usecase)
> Analyst: Combined product + UX perspective

## Summary

Pipeline creation → deploy → monitor flow is functional end-to-end. The core user journey works, but the UI reveals itself as an MVP with rough edges. No critical blockers, but several P1 issues that break the user's mental model.

**Top 3 issues to fix:**
1. **Node name mapping** — canvas names vs Argo display names are completely different
2. **Post-deploy guidance** — no auto-navigation to running workflow
3. **Log viewer UX** — nodeId filter doesn't map to user-visible names

## Screens Examined

1. **Pipeline canvas** (`/pipeline`) — React Flow canvas + component panel + config panel
2. **Workflow list** (`/workflows`) — Table with status filter + pagination
3. **Workflow detail** (`/workflows/:name`) — DAG view + Timeline view + Node side panel

---

## Findings

### P1 (Critical)

| # | Screen | Issue | Fix | Effort |
|---|--------|-------|-----|--------|
| 1 | Workflow detail | **Node names are Argo internal IDs (step-step-1), not user's canvas names.** User drops "Pass Through" on canvas, sees "step-step-3" in DAG. Complete cognitive disconnect. | Store node display names in deployment metadata. Backend should pass `display_names: {"step-1": "Pass Through"}` in API response. Frontend uses them instead of the node ID. | Medium |
| 2 | Pipeline canvas → Deploy | **Deploy success modal says "部署成功" but doesn't explain what that means.** User asked "部署成功就代表它能跑了吗？" — they don't know the workflow was submitted and is running in Argo. | Change copy to "已提交运行" with status indicator. Add prominent "查看运行状态" button that navigates to workflow detail. Consider auto-redirect after 3s with a countdown. | Small |
| 3 | Workflow detail | **DAG shows 4 nodes for a 3-step pipeline.** The extra node is the Argo DAG root (workflow-level node named after the workflow itself). User sees "my-pipeline-e1cba5 (Error)" and is confused about which node actually failed. | Filter out DAG root node from the display, or style it differently (smaller, non-interactive). Show only the user-defined step nodes. | Medium |

### P2 (Major)

| # | Screen | Issue | Fix | Effort |
|---|--------|-------|-----|--------|
| 4 | Workflow list | **No auto-refresh.** User deploys a pipeline, goes to workflow list, they have to click "刷新" to see the new entry. Running workflows don't auto-update status. | Add polling (5s interval) when any workflow is in Running/Pending state. Use React Query or `setInterval` with a "live" toggle. | Small |
| 5 | Workflow detail | **Log viewer doesn't auto-fetch for running nodes.** Logs are fetched once on tab switch. For a running workflow, the user gets a snapshot, not a stream. | Add polling (2s interval) for log content when node is Running. Show "等待日志输出…" during initial loading. | Small |
| 6 | Pipeline canvas | **Right-side panel shows "选择一个节点进行配置" but clicking a node doesn't show a parameter form.** The config panel exists in code but may not have editable fields yet — user can't configure commands, env vars, or resource limits for their Python Script / Pass Through nodes. | Implement or verify the node config panel. For each node type, show editable fields: image, command/args (for Pass Through), script (for Python Script), env vars. | Large |
| 7 | Pipeline canvas | **Only 2 components.** Pass Through (busybox:latest) and Python Script (python:3.12-slim). No way to add custom components from the UI. | Add component registry API integration. Show component descriptions. Allow custom image pull with validation. | Large |

### P3 (Minor / Enhancement)

| # | Screen | Issue | Fix | Effort |
|---|--------|-------|-----|--------|
| 8 | Workflow list | **Status colors are consistent but subtle.** "Error" uses red, "Failed" uses red — indistinguishable without reading the text. Same for "Succeeded" green vs "Running" blue. | Add status icon prefix (✅ ❌ ⏳) alongside colored tags for quick scanning. | Small |
| 9 | Workflow detail | **DAG node layout is auto-positioned in a grid (4 per row) with no edge routing from user's connections.** The edges are inferred from Argo node ID hierarchy (parent.child), not from user's canvas connections. This means the DAG display structure doesn't match what the user built. | Store edge topology from the canvas in deployment metadata. Use it to position nodes and route edges in the DAG view. Consider React Flow's built-in layout algorithms (dagre). | Medium |
| 10 | Workflow list | **Workflow name is the full Argo name (databrew-pl-a0e6708a-dag-fix-test-cv6rl).** Displayed as pipeline name + random suffix. The long format is hard to scan in a table column. | Show only the user-defined pipeline name. The full workflow name can be in a tooltip or a separate "ID" column. | Small |
| 11 | Pipeline canvas | **Empty state hint "从左侧拖入组件 → 连接圆点 → 保存 / 部署" is a static text below the canvas.** Not very visible — gets lost against the canvas background. | Move hint into the canvas itself as a centered overlay (visible only when canvas is empty). Use a more inviting illustration. | Small |
| 12 | Pipeline canvas | **Canvas controls (zoom, fit) use React Flow defaults with "React Flow" attribution link visible.** The "React Flow" watermark is visible at bottom. | Add `proOptions={{ hideAttribution: true }}` to hide the watermark. Style the control panel to match the app's design system. | Trivial |
| 13 | Workflow detail | **Timeline view has Gantt-style bars with duration (seconds), but bars are tiny for fast-running steps.** A 1-second step is a 1 pixel bar. | Set minimum bar width (8px). Consider capping the timeline range to [globalStart, max(globalEnd, globalStart+10s)] for very fast workflows. | Small |
| 14 | Workflow detail | **No "Stop" or "Retry" buttons on the workflow detail page.** Backend has StopWorkflow and RetryDeployment endpoints. | Add action buttons in the workflow detail header (conditionally: show Stop when Running/Pending, show Retry when Failed/Error). | Small |
| 15 | Workflow list | **Created/Finished times show "2026/5/28 12:32:20" format.** Missing seconds padding causing alignment issues in narrow columns. | Use `YYYY-MM-DD HH:mm:ss` or relative time ("2分钟前"). Add a time tooltip with exact timestamp. | Trivial |

### P4 (Nice to Have / Future)

| # | Screen | Issue | Fix | Effort |
|---|--------|-------|-----|--------|
| 16 | Pipeline canvas | No component search/filter | Add search box above component list | Small |
| 17 | Pipeline canvas | No undo/redo for canvas operations | Add keyboard shortcuts (Ctrl+Z) and undo/redo buttons | Medium |
| 18 | Workflow detail | No YAML manifest viewer | Add "Manifest" tab to detail page showing the Argo workflow YAML | Small |
| 19 | Pipeline canvas | No visual indicator of which nodes are required vs optional | Add a visual indicator (asterisk, border) for parameter state | Small |
| 20 | Workflow list | No search by workflow name | Add search input alongside status filter | Small |

## Quick Wins (< 30 min each)

1. **P1 #2:** Change deploy success copy + add auto-navigate prompt
2. **P2 #4:** Add 5s polling for workflow list when Running items exist
3. **P3 #8:** Add status icon prefixes (✅ ❌ ⏳)
4. **P3 #12:** Remove React Flow attribution
5. **P3 #14:** Add Stop/Retry buttons (backend already has it)
6. **P3 #15:** Fix time format alignment

## Top Recommendations

1. **Node name mapping** — This is the single biggest UX problem. Fix this and everything downstream (logs, DAG, monitoring) becomes comprehensible.
2. **Config panel** — Currently the canvas is mostly decorative. Until users can configure what a node actually does, it's a visual editor that can't edit.
3. **Post-deploy flow** — Make the gap between "I deployed" and "it's running" smaller. Auto-navigate, auto-refresh, show status changes.
4. **Component ecosystem** — More components + custom component support is the key to making the pipeline useful beyond PoC.

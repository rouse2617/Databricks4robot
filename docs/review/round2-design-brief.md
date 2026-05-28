# Round 2 Design Brief — Argo Feature Port

> UX/UI/Product design research for porting Argo Workflows v4.0.5 features into DataBrew pipeline platform.
> Generated: 2026-05-28

---

## Table of Contents

1. [Design Principles](#1-design-principles)
2. [Interaction Design for Workflow Operations](#2-interaction-design-for-workflow-operations)
3. [Filter UI Design](#3-filter-ui-design)
4. [Node Detail Panel Enhancement](#4-node-detail-panel-enhancement)
5. [Log Viewer Design](#5-log-viewer-design)
6. [Information Architecture](#6-information-architecture)
7. [Component Hierarchy](#7-component-hierarchy)
8. [User Flow Diagrams](#8-user-flow-diagrams)
9. [Priority Recommendations & Build Order](#9-priority-recommendations--build-order)

---

## 1. Design Principles

These principles guide ALL Round 2 feature development. Every design decision trades off against these.

### P1: Names Must Match — The Canvas Is Truth

The node name a user assigns on the pipeline canvas MUST be the name shown everywhere in the monitoring UI: the DAG view, the timeline, the log viewer, the node detail panel. No Argo internal IDs, no `step-step-3` suffixes visible to users.

- **Why:** The UX report identified this as the #1 cognitive disconnect. Users build a mental model on the canvas, then hit the workflow detail page and cannot map "step-step-3" back to "Pass Through". This breaks the entire monitoring experience.
- **How:** Store `display_name` mappings in deployment metadata. Backend passes `{display_names: {"step-1": "Pass Through"}}` in the API response. Frontend uses display name everywhere, falls back to node ID only when display name is absent.

### P2: Every Action Belongs Where the User Needs It

Operations (stop, retry, delete, etc.) must appear in multiple surfaces: the workflow list row, the detail page header, and the DAG node interaction. A user should never have to navigate away to perform an action they can already see.

- **Why:** The product analysis shows that users feel the workflow is a "black box". Putting operations where users are looking reduces the cognitive gap between "I want to act" and "I can act".
- **How:** Use Argo's pattern of a `WorkflowOperationsMap` — a declarative utility that maps each operation to its valid phases, icons, labels, and action handlers. All UI surfaces consume this same map, ensuring consistency.

### P3: Progressive Disclosure — Start Simple, Layer On

Build the MVP of each feature with the simplest possible implementation, then iterate. A plain `<pre>` log viewer with SSE streaming is better than no streaming while we build xterm integration. A basic filter bar above the table is better than a full sidebar panel that never ships.

- **Why:** Quick wins build momentum and user trust. The porting plan identifies 8 Phase A features that take <1 day each. Delivering those before tackling Phase C builds confidence.
- **How:** Each feature spec in this document has an MVP tier and an Enhanced tier. Ship MVP first.

### P4: Every Error State Must Have a Recovery Path

Failed workflow? Show the error message, highlight the failed node, offer retry. Failed to fetch logs? Show a meaningful message and a retry button. API error? Surface it in-context, not as a generic toast.

- **Why:** The current UX shows "获取日志失败" with no next step. Users hit dead ends. The product analysis notes "用户能到终点，但不知道自己到了终点".
- **How:** Every async operation has three render states: loading, success, error. Error states include: (a) what went wrong in user language, (b) what the user can do about it, (c) a one-click recovery action.

### P5: The Workflow List Is a Launchpad, Not a Log

The workflow list is the user's home base for operations. From this single surface, a user should be able to: see aggregate status at a glance, find any workflow by search or filter, perform batch actions, navigate to detail, and see live state changes without manual refresh.

- **Why:** Users spend most of their time on the list page. Every round trip to detail costs a page load and context switch. Empowering the list surface reduces friction.
- **How:** Summary bar for aggregate counts, search bar for quick lookup, batch selection toolbar for multi-workflow actions, and auto-polling for live state.

---

## 2. Interaction Design for Workflow Operations

### 2.1 Operation Placement Strategy

We have four surfaces where operations can live. Use ALL of them, with different operation subsets per surface.

```
Surface Matrix:

┌──────────────────────────────┬────────────────────────────────┬────────────────────────────┐
│ Surface                      │ Operations                     │ Best For                   │
├──────────────────────────────┼────────────────────────────────┼────────────────────────────┤
│ List page — row actions      │ View, Delete, Stop             │ Quick single-workflow ops  │
│ List page — batch toolbar    │ Delete Selected, Retry Sel.,   │ Bulk operations            │
│                              │ Resubmit Sel., Stop Sel.       │                            │
│ Detail page — header bar     │ Stop, Retry, Resubmit, Delete, │ Full action set per WF     │
│                              │ Suspend, Resume                │                            │
│ DAG node — right-click       │ View Logs, View Events         │ Node-level contextual ops  │
│ Detail page — side panel     │ Copy name/download artifacts   │ Information-heavy actions  │
└──────────────────────────────┴────────────────────────────────┴────────────────────────────┘
```

### 2.2 Detail Page Header — Primary Operations Hub

The detail page header is the primary operations surface, mirroring Argo's approach.

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  [← 返回]  my-pipeline-name        [Running]  创建: 12:32  │  完成: 12:45   │
│                                                           ┌──┬──┬──┬──┬──┐  │
│                                                           │⏸│▶│↻│⟳│🗑│  │  │
│                                                           └──┴──┴──┴──┴──┘  │
│                                                           [DAG | 时间线]    │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Operation Buttons (right-aligned, adjacent to view mode toggle):**

| Icon | Action | Visible When | Confirmation |
|------|--------|-------------|--------------|
| ⏸ Suspend | Running | None (toggle to Resume) |
| ▶ Resume | Suspended | None (toggle to Suspend) |
| ⏹ Stop | Running/Pending | Confirm dialog: "将终止该工作流的运行，确定？" |
| ⟳ Retry | Failed/Error | Confirm dialog: "将重新运行失败节点，确定？" |
| ↻ Resubmit | Any terminal state | Confirm dialog with options (see below) |
| 🗑 Delete | Any | Popconfirm with checkbox "同时删除归档" |

**Design details:**
- Buttons use `Ant Design Button` with `type="text"` and `danger` for destructive actions
- Disabled buttons show tooltip explaining why (e.g., "只有失败的工作流可以重试")
- The `WorkflowOperationsMap` utility (see Section 7) drives which buttons render
- Resubmit button opens a dropdown with two options: "重新提交" (same params) and "重新提交并修改参数"
- Argo reference pattern: `workflow-operations-map.ts` — a static map of `{key: {title, icon, disabled(wf), action(wf)}}`

### 2.3 List Page Row Actions

Each table row shows the most-needed operations directly:

```
┌────────────────────────────────────────────────────────────────────┐
│  Workflow Name              Status    Nodes  Created     Actions   │
│ ────────────────────────────────────────────────────────────────── │
│  my-pipeline-e1cba5    [Running]      3    12:32:20    [查看] [⏹] │  ← single-click View
│  test-pipeline-cv6rl   [Failed]       3    12:28:15    [查看] [⟳] │  ← single-click action
└────────────────────────────────────────────────────────────────────┘
```

**Rules:**
- "查看" is always present (navigates to detail)
- Secondary action button appears based on status (Stop for Running, Retry for Failed/Error)
- Secondary actions use IconButton only (no label) to save space
- Clicking row navigates to detail (current behavior — preserve this)

### 2.4 Batch Toolbar

When rows are selected (via checkbox column), a floating toolbar appears above the table:

```
┌──────────────────────────────────────────────────────────────────┐
│  ☑ 选中 3 项                                                    │
│  ┌────────┐ ┌────────┐ ┌──────────┐ ┌──────────┐               │
│  │ 🗑 删除 │ │ ⟳ 重试 │ │ ↻ 重新提交│ │ ⏹ 停止  │               │
│  └────────┘ └────────┘ └──────────┘ └──────────┘               │
└──────────────────────────────────────────────────────────────────┘
```

**Design details:**
- Toolbar appears as a sticky bar above the table when `selectionCount > 0`
- Each action button is disabled if no selected item is in a valid phase for that action
- Ant Design `Table` `rowSelection` enables checkbox column
- Batch toolbar uses Ant Design `Alert` or custom `Affix` bar pattern

### 2.5 DAG Node — Context Menu

Right-clicking a DAG node shows a context menu with node-level operations:

```
┌────────────────────┐
│ 📋 查看日志         │
│ 📋 查看事件         │
│ ────────────────── │
│ 📋 复制节点名称     │
│ 📋 复制 Pod 名称    │
└────────────────────┘
```

**Design details:**
- Uses Ant Design `Dropdown` with `trigger={['contextMenu']}` on the React Flow wrapper
- Node-level only (not workflow-level operations)
- "查看日志" switches the side panel to the Logs tab
- "查看事件" switches to the Events tab (Phase B feature)

### 2.6 Operation Confirmation Pattern

All destructive operations use this consistent confirmation pattern:

```
┌─────────────────────────────────────┐
│  ⚠ 确认停止工作流                   │
│                                      │
│  将终止「my-pipeline」的当前运行。    │
│  已完成的节点不会重新执行。           │
│                                      │
│  ☐ 同时从归档中删除                  │
│                                      │
│       [取消]    [确定停止]            │
└─────────────────────────────────────┘
```

- Use Ant Design `Modal.confirm` with `danger: true` for destructive actions
- Non-destructive (Suspend, Resume) use `message.success` toast with no confirmation
- Include workflow name in confirmation text
- Show processing state during operation (button loading spinner + "处理中…")

---

## 3. Filter UI Design

### 3.1 Recommendation: Collapsible Filter Panel Above Table

After evaluating three approaches:

| Approach | Pros | Cons | Verdict |
|----------|------|------|---------|
| **Sidebar panel** (Argo style) | Familiar to Argo users, always visible | Takes 240px of horizontal space, competes with nav sidebar | ❌ Too much horizontal space lost |
| **Collapsible above table** | Zero permanent space cost, expandable on demand, matches Ant Design patterns | Filters disappear when collapsed | ✅ **Recommended** |
| **Drawer/popover** | Clean default state | Hidden filters = hard to discover, extra click to apply | ❌ Poor discoverability |

### 3.2 Recommended Layout

```
┌──────────────────────────────────────────────────────────────┐
│  流水线运行                           [🔍 搜索工作流名称...] │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌────┐ ┌────┐                    │
│  │ 全部 │ │ ▶ 运│ │ ✅ 成│ │ ❌ 失│ │ ⏳ 等│                    │
│  │  (10)│ │ 行  │ │ 功  │ │ 败  │ │ 待  │                    │
│  │      │ │ (3) │ │ (5) │ │ (2) │ │ (0) │                    │
│  └─────┘ └─────┘ └─────┘ └────┘ └────┘                    │
│  [🎛 筛选 ▼]  [标签: ___________  ↵]  [📅 时间范围: ______] │
│  ─────────────────────────────────────────────────────────── │
│  (expanded filter panel, shown when "🎛 筛选" is toggled)   │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 状态: ☑ Running  ☐ Succeeded  ☐ Failed  ☐ Error  ☐ P │ │
│  │      结束于: 全部 │ 1小时内 │ 24小时内 │ 7天内 │ 自定义  │ │
│  │      标签: [key=value , key=value , ...]                │ │
│  │                                     [应用] [重置]       │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

### 3.3 Filter Components

| Filter | Component | Detail |
|--------|-----------|--------|
| **Quick status counts** | Ant Design `Statistic` cards or `Segmented` | Clickable — clicking a status sets the filter. Shows count per status. |
| **Search** | `Input.Search` with `allowClear` | Filters by workflow name. Client-side debounced (300ms). |
| **Phase checkboxes** | `Checkbox.Group` | Multi-select. Argo sends as comma-separated `phases=Running,Failed`. |
| **Date range** | `DatePicker.RangePicker` | Filters by creation time. Options: 1h, 24h, 7d, custom. |
| **Label filter** | `Input` + `Tag` | Key=value input. Multiple labels OR-ed. |
| **Applied filters chips** | `Tag` with `closable` | Shows active filters as removable chips. Same pattern as `ActiveFilterChipsRow.tsx` in assets. |

### 3.4 URL State Serialization

All active filters must be serialized to URL query params:

```
/workflows?status=Running,Failed&search=test&tags=env:prod&since=2026-05-01&until=2026-05-28
```

- Enables sharing filtered views (bookmarks, links)
- Preserves filter state on browser back/forward
- Uses `useSearchParams` from React Router

### 3.5 Filters That Genuinely Matter (User-Facing Value)

| Filter | Value | Phase |
|--------|-------|-------|
| Status/Phase | Primary filter — users always sort by status | A |
| Name search | Find specific workflow among many | A |
| Time range | Find workflows from "today" or "last deployment" | B |
| Labels | Advanced — team/owner/environment tags | B |

**Deferred:** `createdBy`, workflow template name, node count, duration range.

---

## 4. Node Detail Panel Enhancement

### 4.1 Current State

The side panel (360px, right side of detail page) has two tabs: "详情" and "日志".

```
┌──────────────────────────────────┐
│  Pass Through                    │  ← Node name
│                                  │
│  ┌─────────┬──────────┐         │
│  │  📋 详情  │  📋 日志  │         │  ← Tabs
│  └─────────┴──────────┘         │
│                                  │
│  名称: step-1                    │
│  状态: [Running]                 │
│  开始时间: 2026/5/28 12:32:20    │
│  完成时间: 2026/5/28 12:32:25    │
│                                  │
│              (empty space)       │
└──────────────────────────────────┘
```

### 4.2 Enhanced Panel Layout

```
┌──────────────────────────────────┐
│  🔄 Pass Through                 │  ← Node name + type icon
│  [Succeeded]   ⏱ 5.2s          │  ← Status tag + duration
│                                  │
│  ┌────┬────┬────┬─────┬─────┐   │
│  │详情│容器│输入│ 输出 │事件 │   │  ← Tabs (NEW!)
│  └────┴────┴────┴─────┴─────┘   │
└──────────────────────────────────┘
```

### 4.3 Tab Contents

#### Tab 1: 详情 (Summary) — Enhanced

```
┌──────────────────────────────────┐
│  📋 详情                          │
│                                  │
│  ┌─ Ant Design Descriptions ──┐  │
│  │ 节点名称     Pass Through  │  │
│  │ 节点类型     Pod           │  │
│  │ 状态         [Succeeded]    │  │
│  │ 消息         (empty or msg) │  │
│  │ 开始时间     12:32:20       │  │
│  │ 完成时间     12:32:25       │  │
│  │ 耗时         5.2s           │  │  ← NEW
│  │ 进度         5/5            │  │  ← NEW (for Running)
│  │ Pod 名称     my-pod-xyz     │  │  ← NEW
│  │ 宿主机       node-1         │  │  ← NEW
│  └─────────────────────────────┘  │
│                                    │
│  [📋 查看日志] [📋 查看事件]      │  ← NEW: action links
└──────────────────────────────────┘
```

**New fields vs. current state:**
- `node type` (Pod, StepGroup, DAG, Retry, Suspend, Skipped)
- `duration` (calculated from startedAt → finishedAt, or elapsed for running)
- `progress` (e.g. "3/5" for steps within a step group)
- `podName` (resolved from node ID)
- `hostNodeName` (K8s node)
- `memoizationStatus` (cached or not)
- Action links to logs and events tabs

#### Tab 2: 容器 (Container Info) — NEW

```
┌──────────────────────────────────┐
│  🐳 容器                          │
│                                  │
│  镜像: python:3.12-slim          │
│  入口: python                    │
│  参数: [-c, script.py]           │
│  命令: python /scripts/run.py    │
│  工作目录: /workspace            │
│                                  │
│  资源:                           │
│  CPU 请求: 500m                  │
│  内存请求: 512Mi                 │
│  GPU: 0                          │
│                                  │
│  环境变量:                        │
│  ┌──┬────────────┬───────────┐  │
│  │  │ 变量名      │ 值        │  │
│  │  │ PYTHONPATH │ /app      │  │
│  │  │ LOG_LEVEL  │ INFO      │  │
│  └──┴────────────┴───────────┘  │
└──────────────────────────────────┘
```

- Displays resolved container spec from the Argo workflow node status
- Fields: image, command, args, workingDir, resources (CPU, memory, GPU), environment variables
- Uses Ant Design `Descriptions` column=1 for field list
- Environment variables use a small inline `Table` if there are > 3 vars

#### Tab 3: 输入 (Inputs) — NEW

```
┌──────────────────────────────────┐
│  📥 输入参数                      │
│                                  │
│  ┌─ Parameters ───────────────┐ │
│  │  name: "world"             │ │
│  │  greeting: "hello"         │ │
│  └────────────────────────────┘ │
│                                  │
│  ┌─ Artifacts ────────────────┐ │
│  │  📎 input-data (S3)        │ │
│  │     path: /data/input.csv  │ │
│  │     size: 2.3 MB           │ │
│  └────────────────────────────┘ │
└──────────────────────────────────┘
```

- Parameters shown as key-value pairs in `Descriptions`
- Artifacts shown as a list with download links
- Artifact download button triggers `GET /api/v1/artifacts/:name/download`

#### Tab 4: 输出 (Outputs) — NEW

```
┌──────────────────────────────────┐
│  📤 输出参数                      │
│                                  │
│  ┌─ Parameters ───────────────┐ │
│  │  result: "processed_data"   │ │
│  │  status: "ok"              │ │
│  └────────────────────────────┘ │
│                                  │
│  ┌─ Artifacts ────────────────┐ │
│  │  📎 output-data (S3)       │ │
│  │     path: /data/output.parq│ │
│  │     size: 15.2 MB          │ │
│  │     [⬇ 下载]               │ │
│  │                            │ │
│  │  📎 logs (S3)              │ │
│  │     path: /logs/node-1.log│ │
│  │     size: 1.1 KB           │ │
│  │     [⬇ 下载]               │ │
│  └────────────────────────────┘ │
└──────────────────────────────────┘
```

- Same structure as Inputs tab
- Download links for artifacts (S3/MinIO paths from Argo)
- Show artifact size as a "badge" or additional info

#### Tab 5: 事件 (Events) — NEW (Phase B)

```
┌──────────────────────────────────┐
│  📋 事件                          │
│                                  │
│  时间         类型     内容       │
│  12:32:25    Normal   Pulled     │
│  12:32:26    Normal   Started    │
│  12:32:50    Normal   Created    │
│                                  │
│  [加载更多...]                    │
└──────────────────────────────────┘
```

- Small table: Timestamp, Type (colored tag), Reason, Message
- Shows K8s events for this node's pod
- Paginated (10 per page)

### 4.4 Panel Width Consideration

The current 360px panel doesn't fit 5 tabs comfortably. Options:
- **Option A:** Keep 360px, use Ant Design `Tabs` small size. Works for summary, but artifacts and container sections will be cramped. **MVP choice.**
- **Option B:** Make panel resizable (drag handle) with min 360px / max 600px. Better for content-rich nodes but adds complexity. **Enhanced choice.**

---

## 5. Log Viewer Design

### 5.1 Current State

```
┌──────────────────────────────────────┐
│  📋 日志                              │
│                                      │
│  ┌─ plain `<pre>` block ───────────┐ │
│  │                                  │ │
│  │  [STEP] 2026/05/28 12:32:20     │ │
│  │  [STEP] Processing data...       │ │
│  │  [STEP] Done.                    │ │
│  │  (end of logs)                   │ │
│  │                                  │ │
│  └──────────────────────────────────┘ │
│  ⚠ Issues:                            │
│  1. One-shot fetch — no streaming     │
│  2. No follow mode — manual scroll    │
│  3. No pod/container filter           │
│  4. No search/highlight              │
└──────────────────────────────────────┘
```

### 5.2 MVP Log Viewer (Phase A)

```
┌──────────────────────────────────────┐
│  📋 日志                              │
│                                      │
│  ┌── Toolbar ──────────────────────┐ │
│  │ [🔍 filter...]  [⏸ Live]  [📋] │ │
│  │             共 142 行            │ │
│  └─────────────────────────────────┘ │
│                                      │
│  ┌── Scrollable Log Area ──────────┐ │
│  │                                  │ │
│  │ ⏳ 等待日志输出…                  │ │  ← shown during initial SSE connect
│  │                                  │ │
│  │ 12:32:20  [STEP] Starting...     │ │
│  │ 12:32:21  [STEP] Processing...   │ │
│  │ 12:32:22  [STEP] Done.           │ │
│  │ ...                              │ │
│  │                                  │ │
│  │ ⟦ new logs appear in real-time ⟧│ │  ← streamed via SSE
│  │                                  │ │
│  └─────────────────────────────────┘ │
│                                      │
│  [🔄 重新加载]                       │  ← shown if fetch fails
└──────────────────────────────────────┘
```

**MVP features:**
1. **SSE streaming** — Replace one-shot `getWorkflowLogs()` call with an `EventSource` connection to `GET /api/v1/workflows/:name/logs/stream?nodeId=...`
2. **Follow mode toggle** — "⏸ Live" toggle button. When enabled, auto-scrolls to bottom. When user manually scrolls up, auto-pauses. "跳至最新" button appears when scrolled away from bottom.
3. **Loading state** — Show "⏳ 等待日志输出…" during initial connect and "⟳ 重新连接中…" if SSE disconnects
4. **Error state** — Show error message with retry button
5. **Basic search** — Simple client-side filter input that highlights/matches lines
6. **Line count** — "共 N 行" footer

**SSE connection architecture:**

```typescript
// argoStreaming.ts — new utility
export function createLogStream(
  workflowName: string,
  nodeId: string,
  onLog: (line: string) => void,
  onError: (err: Error) => void,
  onDone: () => void,
): { close: () => void } {
  // Uses fetch + ReadableStream for SSE
  // Reconnect on error with exponential backoff: 1s, 2s, 4s, 8s (max 30s)
  // Auto-close when workflow reaches terminal state
}
```

### 5.3 Enhanced Log Viewer (Phase B+)

```
┌──────────────────────────────────────────────┐
│  📋 日志                                      │
│                                              │
│  ┌── Toolbar ──────────────────────────────┐ │
│  │ [🔍 filter...]             [📋] [⬇]    │ │
│  │ 容器: [All ▾]   Pod: [pod-1 ▾]         │ │
│  │ 级别: [INFO ▾]  [⏸ Live]  共 142 行    │ │
│  └─────────────────────────────────────────┘ │
│                                              │
│  ┌── Syntax-highlighted Log Area ──────────┐ │
│  │                                          │ │
│  │ 12:32:20  INFO  [STEP] Starting...       │ │  ← color by level
│  │ 12:32:21  INFO  [STEP] Processing...     │ │
│  │ 12:32:22  WARN  [STEP] Slow query...     │ │  ← yellow highlight
│  │ 12:32:23  ERROR [STEP] Failed!           │ │  ← red highlight
│  │ 12:32:24  INFO  [STEP] Retrying...       │ │
│  │ ...                                      │ │
│  │                                          │ │
│  └─────────────────────────────────────────┘ │
│                                              │
│  [跳至最新 ↓]                         1.2s ago│
└──────────────────────────────────────────────┘
```

**Enhanced additions:**
1. **Container/Pod selector** — Dropdown to pick specific container or pod for parallel workflows
2. **Log level filter** — Strip/filter DEBUG and INFO, highlight WARN/ERROR
3. **Syntax highlighting** — Color-code log levels (ERROR=red, WARN=orange, INFO=default, DEBUG=gray)
4. **Download** — Download full log as `.txt` file
5. **Timestamps** — Show "1.2s ago" relative time alongside absolute timestamps
6. **Virtual scrolling** — Use `react-window` or similar for 10000+ log lines (if performance becomes an issue)

**Not in scope for Round 2:**
- xterm integration (Argo uses xterm for interactive shell — we don't need this unless we add `argo exec`)
- Log archival / artifact logs

### 5.4 Log Viewer Placement

The log viewer can be accessed from THREE places:
1. **Node detail side panel** — "日志" tab (current approach, maintain this)
2. **Full-page log view** — Optional `/workflows/:name/logs` route for dedicated log viewing
3. **DAG node context menu** — Right-click → "查看日志"

For Round 2, only #1 is needed. #2 and #3 are Phase B enhancements.

---

## 6. Information Architecture

### 6.1 Current Navigation

```
DataBrew
├── 概览         (/dashboard)
├── 资产管理     (/assets)
├── MCAP 文件    (/mcap-files)
├── 交付管理     (/deliveries)
├── 事件流       (/events)
├── 运行记录     (/algo-runs)
├── 算法处理     (/algo)
├── 流水线       (/pipeline)         ← Pipeline canvas (create/edit)
├── 流水线运行   (/workflows)        ← Workflow list + detail
├── 注册中心     (/registry)
├── 指标检索     (/metrics)
└── 设置         (/settings)
```

### 6.2 Recommended Navigation (Round 2)

Keep the current structure. Add new pages as sub-routes under `/workflows`:

```
├── 流水线       (/pipeline)                 ← Canvas (unchanged)
├── 流水线运行   (/workflows)                ← Workflow list (extended)
│   ├── /workflows/:name                    ← Workflow detail (extended)
│   └── /workflows/:name/logs               ← Full-page log view (future)
```

**What NOT to add in Round 2:**
- Workflow Templates page — defer to Phase C
- Cron Workflows page — defer to Phase C
- Archived Workflows — defer to Phase C

### 6.3 Future Navigation (Phase C Outlook)

When templates and cron are added, consider:

```
├── 流水线       (/pipeline)
│   ├── 画布      (/pipeline)              ← Current canvas
│   └── 模板管理  (/pipeline/templates)     ← NEW: template list + detail
├── 流水线运行   (/workflows)
│   ├── 运行列表  (/workflows)              ← Current list
│   └── 定时任务  (/workflows/cron)         ← NEW: cron workflow list
```

OR add new sidebar items:

```
├── 流水线模板   (/templates)              ← NEW top-level nav item
├── 定时任务     (/cron-workflows)          ← NEW top-level nav item
```

Decision point deferred; monitor user feedback before deciding.

### 6.4 Page Layout Strategy

```
Workflow List Page (extended):
┌─────────────────────────────────────────────────────┐
│  [Header: 流水线运行 + 搜索 + 刷新]                  │
│  [Summary Counts Bar — clickable status pills]      │
│  [Collapsible Filters — status, date, labels]       │
│  [Batch Toolbar — shown when rows selected]         │
│  [Table — name, status, nodes, created, actions]    │
│  [Pagination]                                       │
└─────────────────────────────────────────────────────┘

Workflow Detail Page (extended):
┌─────────────────────────────────────────────────────┐
│  [Header: back + name + status + timestamps +       │
│            operations buttons + view mode toggle]    │
├──────────────────────┬──────────────────────────────┤
│                      │  [Node Side Panel]            │
│  [DAG View]          │  ├── 详情 (enhanced)          │
│  or                 │  ├── 容器 (NEW)               │
│  [Timeline View]     │  ├── 输入 (NEW)               │
│                      │  ├── 输出 (NEW)               │
│                      │  └── 事件 (NEW, Phase B)      │
│                      │                              │
│                      ┊                              │
└──────────────────────┴──────────────────────────────┘
```

---

## 7. Component Hierarchy

### 7.1 New File Structure

```
frontend/src/
├── api/
│   └── workflowApi.ts                    ← EXTEND: add operations, streaming
├── components/
│   └── workflows/                        ← NEW directory
│       ├── SummaryBar.tsx                 ← Status count cards
│       ├── WorkflowFilters.tsx            ← Phase checkboxes, date range, labels
│       ├── ActiveFilterChips.tsx          ← Removable filter chips
│       ├── BatchToolbar.tsx              ← Multi-select action bar
│       ├── OperationsMenu.tsx             ← Workflow operation buttons
│       ├── WorkflowOperations.ts          ← Operations map utility (pure logic)
│       ├── NodeInfoPanel.tsx              ← Enhanced side panel with 5 tabs
│       ├── ContainerInfoTab.tsx           ← Container/script spec
│       ├── InputsOutputsTab.tsx           ← Parameters + artifacts
│       ├── EventsTab.tsx                  ← K8s events table
│       ├── LogStreamViewer.tsx            ← Streaming logs with toolbar
│       └── YAMLViewer.tsx                 ← Read-only YAML display
├── lib/
│   ├── workflowPhase.ts                  ← Phase colors, icons, labels
│   ├── workflowOperations.ts             ← Operations map utility
│   ├── argoStreaming.ts                  ← EventSource/SSE wrapper
│   └── workflowDuration.ts               ← Duration formatting
└── pages/
    ├── WorkflowListPage.tsx               ← EXTEND
    └── WorkflowDetailPage.tsx             ← EXTEND
```

### 7.2 Component Responsibility Map

```
WorkflowListPage
  ├── SummaryBar                   ← Status count cards with click handlers
  ├── WorkflowFilters              ← Collapsible filter panel
  │   ├── ActiveFilterChips        ← Removable filter chips (sub-component)
  ├── BatchToolbar                 ← Batch action bar (conditional)
  └── Table (Ant Design)
      └── Row actions → OperationsMenu

WorkflowDetailPage
  ├── OperationsMenu               ← Workflow-level action buttons in header
  ├── ReactFlowProvider
  │   └── Flow
  │       └── DAG nodes (React Flow)
  │           └── Context menu (Dropdown)
  ├── TimelineView                 ← Enhanced timeline
  └── NodeInfoPanel (Drawer/side panel)
      ├── Tabs
      │   ├── 详情 → Descriptions (enhanced fields)
      │   ├── 容器 → ContainerInfoTab
      │   ├── 输入 → InputsOutputsTab (mode=input)
      │   ├── 输出 → InputsOutputsTab (mode=output)
      │   └── 事件 → EventsTab
      └── LogStreamViewer (inside Logs tab of parent, or standalone)
```

### 7.3 State Management Architecture

No global state store needed. Pages manage their own state via hooks:

**Per-page state:**
```typescript
// WorkflowListPage state
const [items, setItems] = useState<WorkflowSummary[]>([]);
const [loading, setLoading] = useState(false);
const [filters, setFilters] = useState<WorkflowFilters>({});
const [searchQuery, setSearchQuery] = useState('');
const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
const [isLive, setIsLive] = useState(false); // auto-poll toggle

// WorkflowDetailPage state
const [wf, setWf] = useState<WorkflowDetail | null>(null);
const [loading, setLoading] = useState(true);
const [selectedNode, setSelectedNode] = useState<WorkflowNodeStatus | null>(null);
const [viewMode, setViewMode] = useState<'dag' | 'timeline'>('dag');
const [logStream, setLogStream] = useState<{close: () => void} | null>(null);
```

**Shared utilities (no state):**
- `workflowOperations.ts` — Pure function: `getOperations(wf) => Operation[]`
- `workflowPhase.ts` — Pure function: `getPhaseColor(phase) => string`
- `argoStreaming.ts` — Hook: `useLogStream(name, nodeId) => { lines, isConnected, error }`

### 7.4 Key Shared Utility: WorkflowOperationsMap

This is the single most important utility to extract from Argo. It centralizes ALL operation logic:

```typescript
// workflowOperations.ts
export interface WorkflowOperation {
  key: string;
  label: string;
  icon: ReactNode;
  danger?: boolean;
  confirm?: { title: string; content: string };
  isVisible: (wf: WorkflowDetail) => boolean;
  isEnabled: (wf: WorkflowDetail) => boolean;
  disabledReason?: (wf: WorkflowDetail) => string;
  action: (wf: WorkflowDetail) => Promise<void>;
}

export const WORKFLOW_OPERATIONS: WorkflowOperation[] = [
  {
    key: 'stop',
    label: '停止',
    icon: <StopOutlined />,
    danger: true,
    confirm: { title: '确认停止', content: '将终止该工作流的运行' },
    isVisible: (wf) => wf.status === 'Running' || wf.status === 'Pending',
    isEnabled: () => true,
    action: (wf) => workflowApi.stopWorkflow(wf.name),
  },
  {
    key: 'retry',
    label: '重试',
    icon: <ReloadOutlined />,
    isVisible: (wf) => wf.status === 'Failed' || wf.status === 'Error',
    isEnabled: (wf) => wf.status === 'Failed' || wf.status === 'Error',
    disabledReason: () => '只有失败的工作流可以重试',
    action: (wf) => workflowApi.retryWorkflow(wf.name),
  },
  // ... resubmit, suspend, resume, delete
];
```

This map is consumed by `OperationsMenu` (detail header), table row actions, and `BatchToolbar`.

---

## 8. User Flow Diagrams

### 8.1 Deploy → Monitor Flow (Critical Path)

```
User Action                     System Response                   UI State
─────────────                   ────────────────                   ────────
1. Configure pipeline           Canvas shows nodes                 ✅ Pipeline edit mode
   on canvas

2. Click "部署"                 Validate + save pipeline           ⏳ DeployPanel shows spinner
                                Call deploy API

3. —                            Deploy success                     ✅ Modal: "已提交运行"
                                                                   [查看运行状态] [留在画布]
                                                                   自动跳转: 3s countdown

4. Click "查看运行状态"         Navigate to workflow detail        ✅ WorkflowDetailPage
   OR auto-redirect                                                 Status: [Running]
                                                                   DAG: nodes animating

5. Watch DAG execution          Auto-poll workflow status          DAG nodes update:
                                5s interval                        Running → Succeeded (green)
                                                                   Failed node → red
                                                                   Progress bars on running

6. Failed node detected         Click failed node                  Right panel:
                                                                   Message: error text
                                                                   Logs tab: auto-show logs
                                                                   Retry button appears

7. Click "重试"                 Confirm dialog → Call retry API    ⏳ Loading → node restarts
                                                                   Status updates via poll
```

### 8.2 Batch Delete Flow

```
User Action                     UI State
─────────────                   ────────
1. Go to workflow list           WorkflowListPage with table

2. Check 3 rows                  Checkboxes selected
                                 BatchToolbar appears: [🗑 删除] [⟳ 重试] [↻ 重新提交]

3. Click "🗑 删除"              Modal: "确认删除选中的 3 个工作流？"
                                 [☐ 同时从归档中删除]

4. Confirm                       Loading spinners on selected rows
                                 Success toast: "已删除 3 个工作流"
                                 Table refreshes, toolbar disappears
```

### 8.3 Log Investigation Flow

```
User Action                     UI State
─────────────                   ────────
1. On workflow detail           DAG view showing nodes
   Running workflow

2. Click failed node            Right panel opens with "详情" tab

3. Switch to "日志" tab         LogStreamViewer:
                                 "⏳ 等待日志输出…" (connecting)

4. —                            SSE connected
                                 Log lines appear in real-time
                                 Follow mode ON → auto-scrolls

5. Scroll up to inspect          Follow mode auto-pauses
                                 "跳至最新 ↓" button appears

6. Type in filter box            Log lines filtered client-side
                                 Matching lines highlighted

7. Click "跳至最新"              Scroll to bottom, follow mode resumes
```

### 8.4 Suspend/Resume Flow

```
User Action                     UI State
─────────────                   ────────
1. Long-running workflow        Detail page, status [Running]
   is consuming resources

2. Click "⏸ Suspend"           Button shows loading
                                 API: suspend workflow
                                 Status: [Suspended] — amber tag
                                 Actions update: Suspend → Resume

3. Fix the issue (external)     Page auto-polls
                                 Still Suspended

4. Click "▶ Resume"             Button shows loading
                                 API: resume workflow
                                 Status: [Running]
                                 Nodes resume execution
```

### 8.5 Resubmit with Modification Flow

```
User Action                     UI State
─────────────                   ────────
1. Failed workflow              Detail page, status [Failed]

2. Click "↻ 重新提交"           Dropdown: [重新提交] [重新提交并修改参数]

3. Select "重新提交"             Confirm dialog → API call
                                 New workflow created with same params
                                 Navigate to new workflow detail

   — OR —

3. Select "重新提交并修改参数"   Modal with parameter override form
                                 (parameters extracted from original WF)
                                 User edits → Submit → new WF created
```

---

## 9. Priority Recommendations & Build Order

### 9.1 Round 2 Scope (Recommended Sprint Plan)

Round 2 should focus on **Phases A + selected B features** from the porting plan, ordered by user impact / effort ratio.

```
Sprint 1: Foundation + Quick Wins (3 days)
───────────────────────────────────────────
[P0] P1 #1 — Node name mapping (canvas → Argo names)
      Effort: Medium (backend + frontend)
      Impact: ★★★★★ — Fixes #1 cognitive disconnect
      Files: backend handler + WorkflowDetailPage.tsx

[P0] A1 — Summary Counts Bar
      Effort: 0.5 day
      Impact: ★★★★☆ — Instant list page improvement
      Files: SummaryBar.tsx, WorkflowListPage.tsx

[P0] A4 — Workflow Name Search
      Effort: 0.25 day
      Impact: ★★★★☆ — Essential as WF count grows
      Files: WorkflowListPage.tsx

[P0] Deploy success modal improvement
      Effort: 0.25 day
      Impact: ★★★★★ — Fixes #2 cognitive disconnect
      Files: DeployPanel.tsx

Sprint 2: Workflow Operations (3 days)
───────────────────────────────────────
[P0] A2 — Delete Workflow (list + detail)
      Effort: 0.5 day
      Impact: ★★★★☆ — Enables lifecycle management

[P0] A3 — Stop/Terminate button
      Effort: 0.5 day
      Impact: ★★★★★ — Immediate user need

[P0] A7 — Resubmit Workflow
      Effort: 0.5 day
      Impact: ★★★★☆ — Fast recovery path

[P0] A8 — Retry Workflow
      Effort: 0.5 day
      Impact: ★★★★★ — Fast recovery path

[P1] B9 — Suspend/Resume
      Effort: 0.5 day
      Impact: ★★★☆☆ — Useful for long-running WFs

[P1] WorkflowOperations utility
      Effort: 0.5 day
      Impact: ★★★★★ — Foundation for all operation buttons
      Files: WorkflowOperations.ts

Sprint 3: Detail Page Enhancement (4 days)
──────────────────────────────────────────
[P0] A5 — YAML Viewer
      Effort: 0.5 day
      Impact: ★★★★☆ — Deep inspection without external tools

[P0] A6 — Node Container Info tab
      Effort: 0.75 day
      Impact: ★★★★☆ — See what's running in each node

[P1] B3 — Node Inputs/Outputs (parameters + artifacts)
      Effort: 1 day
      Impact: ★★★☆☆ — Important for data pipelines
      Note: depends on backend response enrichment

[P1] B7 — Node Detail Summary enrichment
      Effort: 1 day
      Impact: ★★★★☆ — Duration, progress, pod name all valuable

[P1] NodeInfoPanel refactor (from inline to component, 5 tabs)
      Effort: 1 day
      Impact: ★★★★★ — Foundation for all node detail features

Sprint 4: Log Streaming + Filtering (4 days)
─────────────────────────────────────────────
[P0] B2 — Log Streaming (MVP: SSE + follow mode)
      Effort: 2-3 days
      Impact: ★★★★★ — Transforms debugging

[P1] B1 — Rich Filtering (status, date, labels, URL serialization)
      Effort: 2 days
      Impact: ★★★★☆ — Essential for many WFs

[P2] B8 — Batch Selection + Toolbar
      Effort: 1 day
      Impact: ★★★☆☆ — Nice to have for bulk ops

Sprint 5: Polish + DAG Enhancement (3 days)
────────────────────────────────────────────
[P1] B5 — Timeline Enhancement (min bar width, date span, color coding)
      Effort: 1 day
      Impact: ★★★☆☆ — Better visibility

[P1] B6 — DAG Enhancements (search, layout toggle, progress bars)
      Effort: 2 days
      Impact: ★★★★☆ — Better DAG comprehension

[P2] Auto-refresh/polling for Running workflows
      Effort: 0.5 day
      Impact: ★★★★★ — Live feel without manual refresh
```

### 9.2 Sprint Sequence Summary

```
Sprint 1: Foundation (3d)     →  Sprint 2: Operations (3d)
  ✓ Node name mapping            ✓ Stop/Retry/Resubmit/Delete
  ✓ Summary bar                  ✓ Suspend/Resume
  ✓ Name search                  ✓ Operations utility
  ✓ Deploy modal

         ↓                            ↓
Sprint 3: Detail Panel (4d)    →  Sprint 4: Streaming (4d)
  ✓ YAML Viewer                   ✓ Log SSE streaming
  ✓ Container info                ✓ Rich filtering
  ✓ Inputs/Outputs                ✓ Batch toolbar
  ✓ Enhanced summary

         ↓
Sprint 5: Polish (3d)
  ✓ Timeline enhancements
  ✓ DAG search/layout/progress
  ✓ Auto-refresh polling
```

**Total Round 2 effort: ~17 days** (vs. the full ~45 days in the porting plan).

### 9.3 What NOT to Build in Round 2

| Feature | Reason | Target |
|---------|--------|--------|
| Workflow Templates (C1) | Separate domain, 5+ days | Phase C |
| Cron Workflows (C3) | Separate domain, 7+ days | Phase C |
| Namespace Switching (C4) | Adds cross-cutting complexity | Phase C |
| Event Streaming / Watch (C7) | Backend-intensive SSE pattern | Phase C |
| Log Archival (C8) | Depends on artifact storage | Phase C |
| xterm integration | Overkill — `<pre>` + SSE is sufficient | Defer indefinitely |
| Workflow-of-workflows lineage | Rare use case | Phase C |
| Full-page log view | Nice to have, not essential | Phase B+ |

### 9.4 Risk Items Requiring Attention

1. **Backend response enrichment** — The current API handlers strip most fields. Node container info, inputs/outputs, and artifacts all require backend changes. **Start Sprint 1 backend work in parallel with frontend.**

2. **SSE streaming via backend proxy** — New backend pattern (Gin `c.Stream()`). Verify this works in a spike before Sprint 4.

3. **WorkflowOperationsMap** — Must be designed to handle loading/error states uniformly across all surfaces. An edge case where one operation fails while others work needs clear handling.

4. **URL state vs. local state for filters** — Filter state in URL enables sharing but adds complexity for client-side filters like search. **Decision: status + labels + date range → URL; search query → local state only.**

---

## Appendix A: Argo UI Screenshot References (Conceptual)

```
Argo Workflow Detail Layout:
┌──────────────────────────────────────────────────────────────┐
│ [Workflow Name]    [Phase Tag]    [创建时间]                  │
│ ┌───┬───┬───┬───┐  [DAG] [Timeline] [Raw] [Histogram]       │
│ │S │R │F │F │E │  ← Summary counts bar                       │
│ └───┴───┴───┴───┘                                           │
│ ┌─────────────────────┬────────────────────────────────────┐ │
│ │                     │  [RESUBMIT] [DELETE] [LOGS] [SHARE]│ │
│ │                     │                                    │ │
│ │    DAG Graph        │    Node Detail Panel               │ │
│ │                     │    ┌─ Summary ───────────────────┐ │ │
│ │    [node1]──→[node2]│    │ Name, Status, Type          │ │ │
│ │         ↘          │    │ Container, Image, Args      │ │ │
│ │          [node3]    │    │ Inputs, Outputs, Artifacts  │ │ │
│ │                     │    │ Events                      │ │ │
│ │                     │    └─────────────────────────────┘ │ │
│ └─────────────────────┴────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

## Appendix B: Key Technology Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| SSE library | Native `EventSource` + custom reconnect | No dependency needed; Argo's RxJS pattern not necessary in modern browsers |
| Filter state storage | URL params + `useSearchParams` | Shareable, bookmarkable, back/forward friendly |
| Log syntax highlighting | Simple regex-based colorization (no library) | Avoid heavy dependency for MVP; can add `react-syntax-highlighter` later |
| YAML viewer | Lazy-loaded Monaco editor | Already in ecosystem; code folding + YAML validation out of box |
| Virtual scrolling | Defer until perf issues arise | Current WF list is < 100 items; logs may need it but <pre> is fine for MVP |
| State management | useState (no new dependency) | Current pattern works fine; no Redux/Zustand needed for page-local state |

---

*This design brief is a living document. Update it as implementation reveals new constraints or user feedback changes priorities.*

# Fix Workflow Detail Page — Full DAG + Watcher Backfill + Auto-Layout + Source Badge + Terminal Empty Polish

## Problem

The workflow detail page (`/pipeline/executions/:name`) has seven data-quality and presentation problems that erode user trust:

1. **Lazy node creation:** A 4-node pipeline only shows the last node (`step-audit`) in the DAG canvas. The other 3 nodes (`step-prepare`, `step-checksum`, `step-release`) only appear AFTER the watcher observes them at runtime, which can be many seconds after page load. Users see a half-finished DAG and cannot understand the workflow's true shape without waiting or refreshing.

2. **Watcher coverage gap:** Workflows that complete BEFORE the watcher scans them, or that were submitted outside of DataBrew's deploy endpoint (e.g. via `kubectl` or Argo CLI), are never recorded in the DataBrew ledger (`pipeline_run_events` table). The detail page then shows `节点: 4` but `事件: 0` and `暂无资产节点明细` — a contradictory state that confuses users.

3. **Empty state ambiguity:** When a workflow has no DataBrew events, the alert says "这是历史工作流或外部提交的工作流，暂时没有 DataBrew 运行事件记录" but provides no fallback path (e.g. "click here to see Argo-native events") or actionable next step. The `type="warning"` styling also suggests a system failure.

4. **DAG canvas uses fixed coordinates, not auto-layout:** Nodes appear where their spec puts them — which for the 4-node linear pipeline happens to be a horizontal row, but for branching/fan-out/fan-in workflows produces visually messy and overlapping layouts. The 4 nodes occupy a ~600px band in the middle of a 1490px canvas, leaving ~50% of the canvas as wasted empty grid. There is no "fit view" or "auto-arrange" affordance.

5. **No zoom-to-fit on load:** When the canvas is wider than the rendered graph, nodes appear in a thin band; when the canvas is narrower, nodes are clipped on the right edge with no way to recenter.

6. **Duplicate "terminal" entry points confuse users:** A node card has 4 quick-action icons at the bottom (e.g. 概览 / 日志 / IO / 终端), AND a click on the card opens a side drawer that also contains a "运行环境" tab → "调试" sub-section. The two paths lead to similar functionality but with different UIs (one is a quick panel, the other is a full drawer), and users cannot tell which to use. This is a **navigation/redundancy** problem, not a feature gap.

7. **External workflows look like broken DataBrew workflows:** The detail page has no indication of *where the workflow came from*. An external workflow (kubectl, Argo CLI, or another orchestrator) shows the same UI as a DataBrew deployment, with the same empty-state placeholders. Users cannot tell whether the page is broken or whether the workflow is from a different source. The "资产节点明细" table is meaningless for external workflows (they were never bound to DataBrew assets) and should be hidden, not shown as "暂无".

8. **Terminal UI takes up vertical space even when disabled:** The "调试" sub-section in the runtime tab renders a 220px-tall black terminal panel and four disabled command buttons (`sh` `pwd` `ls` `env`) plus two disabled action buttons (`打开终端` / `终止`) — all of which is just visual noise when `execEnabled` is `false`. Users see a screenful of disabled controls and conclude the page is broken.

3. **Empty state ambiguity:** When a workflow has no DataBrew events, the alert says "这是历史工作流或外部提交的工作流，暂时没有 DataBrew 运行事件记录" but provides no fallback path (e.g. "click here to see Argo-native events") or actionable next step.

4. **DAG canvas uses fixed coordinates, not auto-layout:** Nodes appear where their spec puts them — which for the 4-node linear pipeline happens to be a horizontal row, but for branching/fan-out/fan-in workflows produces visually messy and overlapping layouts. The 4 nodes occupy a ~600px band in the middle of a 1490px canvas, leaving ~50% of the canvas as wasted empty grid. There is no "fit view" or "auto-arrange" affordance.

5. **No zoom-to-fit on load:** When the canvas is wider than the rendered graph, nodes appear in a thin band; when the canvas is narrower, nodes are clipped on the right edge with no way to recenter.

6. **Duplicate "terminal" entry points confuse users:** A node card has 4 quick-action icons at the bottom (e.g. 概览 / 日志 / IO / 终端), AND a click on the card opens a side drawer that also contains a "运行环境" tab → "调试" sub-section. The two paths lead to similar functionality but with different UIs (one is a quick panel, the other is a full drawer), and users cannot tell which to use. This is a **navigation/redundancy** problem, not a feature gap.

## Goals

- Render the **complete DAG from `wf.Spec.Templates`** on initial page load, marking unstarted nodes as `Pending`.
- **Auto-layout the DAG** with a top-down or left-right algorithm so branching workflows render cleanly.
- **Fit the view to nodes** on first render and after every node-set change, eliminating wasted canvas.
- Show **node count and event count from the DataBrew ledger** (`pipeline_run_events`), not from Argo's `Status.Nodes`, so the numbers stay self-consistent.
- **Backfill the run ledger** for any existing workflow that ran during the gap (one-shot scan at startup, plus a wider scan window).
- Improve empty state copy with a clear explanation of where to find events (Argo UI) and how to re-deploy via DataBrew to record the ledger going forward.
- Expose the DataBrew-recorded event count in the watcher status banner so users can spot coverage gaps.
- **Consolidate the node quick-actions and the drawer terminal entry points** so the user can find one clear way to open the pod terminal for any given node.
- **Add a "DataBrew 运行" / "外部 Workflow" source badge** at the top of the detail page so users immediately know where the workflow came from.
- **Suppress DataBrew-only modules** (资产节点明细, 事件时间线, 调试) on external workflows and show a single "this workflow was not deployed through DataBrew" placeholder instead.
- **Collapse the terminal UI to a one-liner message** when `execEnabled === false`, hiding the disabled command buttons and the 220px black panel.

## Non-Goals

- No real-time streaming of Argo-native events into DataBrew (just initial backfill).
- No reconciliation of the legacy `Status.Nodes` DAG with the new template-merged DAG when both disagree (template wins).
- No retroactive backfill of historical workflows older than 7 days.
- No approval workflow or PII redaction for command payloads.
- No full ELK/dagre configuration UI; only a sensible default layout direction (auto-pick LR vs TB based on graph aspect ratio).
- No custom drag-and-drop node positioning; the layout is read-only.
- No removal of the "debug" sub-section inside the runtime tab — terminal debugging is a legitimate feature, but the entry point should be **unified** (e.g. one primary "Debug" icon in the node card, drawer as the fullscreen escape hatch).

## Proposed Scope

### 1. Frontend: Always show full DAG from templates + auto-layout

In `WorkflowDetailPage.tsx`'s `useWorkflowDetail` hook:

```typescript
import dagre from '@dagrejs/dagre';
import { getNodesBounds, getViewportForBounds } from 'reactflow';

const layoutedNodes = useMemo(() => {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: 'LR', nodesep: 40, ranksep: 80 });
  nodes.forEach(n => g.setNode(n.id, { width: 180, height: 80 }));
  edges.forEach(e => g.setEdge(e.source, e.target));
  dagre.layout(g);
  return nodes.map(n => {
    const pos = g.node(n.id);
    return { ...n, position: { x: pos.x - 90, y: pos.y - 40 } };
  });
}, [nodes, edges]);

// Fit to view once after layout
useEffect(() => {
  if (layoutedNodes.length > 0) {
    const { x, y, width, height } = getNodesBounds(layoutedNodes);
    const padding = 60;
    const viewport = getViewportForBounds(
      { x, y, width, height },
      canvasWidth, canvasHeight, 0.5, 2, padding,
    );
    setViewport(viewport);
  }
}, [layoutedNodes]);
```

Add an `autoLayout: 'lr' | 'tb' | 'auto'` prop that picks LR for shallow linear graphs and TB for branching graphs.

### 2. Backend: Merge templates and status nodes

In `handlers/workflow/handler.go`'s `GetWorkflow` and `ListExecutions`:

```go
nodes := mergeTemplateAndStatusNodes(wf.Spec.Templates, wf.Status.Nodes)
for name, tpl := range wf.Spec.Templates {
    if _, ok := wf.Status.Nodes[name]; !ok {
        nodes = append(nodes, synthesizedNodeFromTemplate(tpl))
    }
}
return nodes
```

The synthesized node carries `Phase = "Pending"`, `Type = "DAG"`, and inherits the spec's `Dependencies` so the frontend can render edges between pending nodes and their parents.

### 3. Backend: Watcher scans a wider window

In `usecase/pipeline/usecase.go`'s watcher tick:
- Keep scanning `Running/Pending` workflows (current behavior).
- **Additionally** scan `Succeeded/Failed/Error` workflows that **finished within the last 7 days** and have **zero events** in `pipeline_run_events` — these are the backfill candidates.
- For each candidate, call Argo's `WatchWorkflow` once to drain its event log, then persist each event through `runEventRepo.Create(...)`.

### 4. Backend: Ledger counts surface in the watcher status

In `models.PipelineRunWatcherState`, add a `LedgerHealth` struct:

```go
type LedgerHealth struct {
    TotalRuns       int `json:"totalRuns"`
    RunsWithEvents  int `json:"runsWithEvents"`
    RunsWithoutEvents int `json:"runsWithoutEvents"`
    LastBackfillAt  *time.Time `json:"lastBackfillAt,omitempty"`
}
```

The `GetRunWatcherStatus` endpoint returns this so the UI banner can show `运行账本：123/127 workflows have events (4 missing)`.

### 5. Frontend: Improve empty-state copy

The current copy is a generic `Alert type="warning"` that says the workflow is missing events, but gives no positive context. Users land on the page expecting a full event timeline and panic when they see the warning. Replace the alert body with a copy that:

1. States the **fact**: events are not present.
2. Explains **why this is normal**: the workflow was likely submitted outside of DataBrew (Argo CLI, kubectl, or another orchestrator).
3. Tells the user **what they can still do**: inspect the DAG, view Pod logs, open the terminal, see cost breakdown — everything else on the page still works.
4. Offers a **remediation path** if they want full DataBrew tracking going forward: re-deploy the template from DataBrew, or click through to the Argo UI.

New copy:

> **"暂无 DataBrew 运行事件。该工作流可能由 Argo 外部提交，仍可查看 DAG、Pod 和日志。"**
>
> _(This addresses three problems with the current copy: it removes the assumption that the user knows what "DataBrew 运行事件" means; it frames the empty state as normal, not an error; and it points to the features on the same page that still work.)_

Use a quieter `Alert type="info"` instead of `type="warning"`, since the situation is not actually a system failure. Optionally append:

> "想要完整的事件追踪？[在 DataBrew 重新部署此模板](#) — 未来的运行会自动记录到运行账本。"

The copy is in Chinese, matching the rest of the page; the i18n pipeline can add English and other locales later.

### 6. Frontend: Unify the two terminal/debug entry points

Currently a node card shows four quick-action icons (`概览` / `日志` / `IO` / 隐式第四个), and clicking the card opens a side drawer that contains a `运行环境` tab → `调试` sub-section with `DebugTab` (terminal). This makes the **same capability discoverable from two places** with different affordances and is confusing.

In `WorkflowNodeDetailPanel.tsx` + the node quick-action component in `WorkflowDetailPage.tsx`:

- **Make the node card's quick actions explicit and single-purpose**: the four icons should be `概览` (opens drawer on summary tab), `日志` (opens drawer on logs tab OR the floating log viewer), `IO`, **`终端`** (only visible if `node.debug?.execEnabled === true`; otherwise greyed out with tooltip "Pod 终端未启用"). Removing the implicit "click anywhere to open" ensures the user always lands on the right tab.
- **Remove the duplicate `调试` section from the runtime tab**. Or — if keeping it for power users — rename to `高级调试` and move it behind a secondary click to make the drawer primary view the four main concerns: `概览 / 日志 / 运行环境 / 输入/输出`.
- **Add a `打开终端` button in the runtime tab** that scrolls the drawer to the `DebugTab` panel, so users who go through the drawer path still get to the terminal in one click.

The end state:

| Surface | Visible entry | Hidden entry |
|---|---|---|
| Node card (quick actions) | `终端` icon (greyed when disabled) | — |
| Node card (click anywhere) | — | (no-op, or opens drawer on summary) |
| Drawer → `运行环境` tab | `打开终端` button at top of section | — |
| Drawer → `调试` (renamed `高级调试`) | — | Hidden behind a "Show advanced" toggle |

### 7. Frontend: Collapse terminal UI when Pod exec is disabled

In `DebugTab` (`WorkflowNodeDetailPanel.tsx`), when `execEnabled === false`, **do not render the command buttons, the "打开终端" button, or the 220px black terminal panel**. Render only:

```jsx
<Empty
  image={<LockOutlined />}
  description={
    <Space direction="vertical" size={4}>
      <Typography.Text strong>当前执行目标未开启 Pod 终端</Typography.Text>
      <Typography.Text type="secondary">
        可在 设置 → 执行目标 中开启 Pod 终端 policy
      </Typography.Text>
    </Space>
  }
/>
```

This saves ~280px of vertical space per node and removes the visual "this is broken" feeling when the terminal is disabled. The full UI only appears when `execEnabled === true`.

### 8. Frontend + Backend: Source badge + modular suppression for external workflows

**Frontend:** In the detail page header (alongside the workflow name and status), add a `Tag` that indicates the workflow's origin:

| Source | Tag | Color | Condition |
|---|---|---|---|
| DataBrew deploy | `DataBrew 运行` | `green` | `run.id != null` OR `runEventState.items.length > 0` |
| External (kubectl, Argo CLI, etc.) | `外部 Workflow` | `orange` | `run.id == null` AND `runEventState.items.length === 0` AND workflow is not a pipeline template |

**Frontend:** When the source is "外部 Workflow", conditionally hide these modules and show a single info banner instead:

- ❌ **资产节点明细** (asset–node table) — hidden, replaced with: "此工作流不是由 DataBrew 部署，没有资产绑定信息。仍可使用 DAG、Pod 日志和终端调试。"
- ✅ **事件时间线** — kept, but with the improved empty-state copy from section 5.
- ✅ **节点成本/监控/计费** — kept (these come from Argo, not DataBrew).
- ✅ **终端调试** — kept, but optionally hidden if the node is not running.

**Backend:** Add a `source` or `ledger_state` field to the workflow detail API response so the frontend can make the source determination without a second query. The `ledger_state` column already covers this (see Schema section): `no_ledger` → external workflow.

## Open Questions

- Should we keep showing `节点: 4` from Argo even if `事件: 0` from the ledger, or unify to ledger-only?
- For backfill, do we replay events from start, or only from the watcher's last cursor? Replay may be a few minutes of pod logs.
- If a workflow was submitted via DataBrew but the watcher missed it (e.g. because the watcher is a single instance with `ActiveScanLimit=100` and the workflow is the 101st), is the new wider scan enough, or do we also need pagination?
- Does this need a new database table? (see **Schema** section below)
- **Linear workflow auto-layout:** confirm `LR` is the right default. TB may be more familiar for users coming from GitHub Actions / GitLab CI.
- **dagre vs elkjs:** dagre is smaller and faster, but only does the simple topological layout. elkjs supports more sophisticated algorithms. Stick with dagre unless we discover edge cases.

## Schema Decision

**No new tables.** The existing 4 tables already form a sufficient ledger:

| Table | Already exists | Used for |
|---|---|---|
| `pipeline_runs` | ✅ | Workflow-level state (4 nodes, 1m 33s, $0.0076) |
| `pipeline_run_events` | ✅ | Event timeline (33 business events) |
| `pipeline_run_nodes` | ✅ | Node-level cost breakdown (4 rows in the asset-node table) |
| `pipeline_run_watcher_states` | ✅ | Watcher health + scan lag |

The "0 events" mystery is **not** a missing table problem — it's a **missing denormalized column**. When `COUNT(e.id) = 0` for a run, the page can render the empty-state alert, but it cannot say *why* the run is missing its ledger (was it submitted outside DataBrew? was the watcher down? did the workflow finish before the watcher noticed it?). To make this distinction without a new table, **add a single denormalized column to `pipeline_runs`**:

```sql
ALTER TABLE pipeline_runs ADD COLUMN ledger_state TEXT NOT NULL DEFAULT 'pending';
-- values: 'pending' | 'has_ledger' | 'no_ledger' | 'backfilling'
```

The watcher updates this column on every scan tick:

- `pending` — never scanned, default for new rows
- `backfilling` — watcher is currently syncing events from Argo
- `has_ledger` — `COUNT(events) > 0` after the latest scan
- `no_ledger` — scan completed but the workflow has no events in Argo either (truly empty workflow, or external submission that never reported back)

This column is read by the frontend's detail page to render the right empty-state copy:

- `no_ledger` + workflow is `Succeeded`/`Failed` for > 24h → show "可能由 DataBrew 之外提交" with Argo UI link
- `pending` + workflow finished < 7d ago → show "watcher 正在补录事件" with a spinner
- `has_ledger` → no alert, render events

The `LedgerHealth` snapshot in `PipelineRunWatcherState` is computed by a single query (`SELECT ledger_state, COUNT(*) FROM pipeline_runs GROUP BY ledger_state`) — no new table, just an aggregation endpoint.

## Why not a new `pipeline_run_ledger_gaps` table?

1. **Coverage gaps are not an entity** — they're a derived state of an existing run. Modeling them as a separate table creates a "soft foreign key" that has to be kept in sync.
2. **The 3 existing tables already partition concerns** — runs (the subject), events (the log), nodes (the cost rows). Adding a 4th ledger table would overlap with all three.
3. **A denormalized column costs O(1) write per run** (the watcher already touches the row on every scan). A new table would require a separate INSERT/UPDATE per row plus referential integrity.
4. **Aggregation queries** (for the watcher health banner) work equally well from `pipeline_runs.ledger_state` as from a dedicated table — and don't need a new index if we add a partial index `CREATE INDEX idx_runs_ledger_state ON pipeline_runs(ledger_state) WHERE ledger_state != 'has_ledger'`.

## Linked Tickets

- **CYB-1569** — Pod terminal exec (terminal capability already works for ledger-recorded workflows).
- **CYB-1565** — Pipeline observability (DAG view + ledger integration).

## Rollout

1. Merge spec + designs.
2. Add `mergeTemplateAndStatusNodes` helper and update the two handlers.
3. Extend watcher scan window in usecase.
4. Add `LedgerHealth` to watcher status payload.
5. Update frontend detail page to render merged DAG.
6. **Add dagre-based auto-layout to `useWorkflowDetail`.**
7. **Add fit-to-view on initial render and after node-set changes.**
8. Update frontend empty-state copy + Argo UI link.
9. **Unify node quick-action icons and the drawer's `调试` sub-section — one visible primary entry per surface, with no duplicate path.**
10. **Collapse terminal UI to one-liner when `execEnabled === false`.**
11. **Add source badge (`DataBrew 运行` / `外部 Workflow`) to detail page header.**
12. **Conditionally hide `资产节点明细` for external workflows, replace with infobox.**
13. Run smoke test against the four existing test workflows; verify the missing-events workflow now shows backfilled events; verify the 4-node DAG fits the canvas with auto-layout; verify there is exactly one obvious way to open a Pod terminal; verify an external workflow shows `外部 Workflow` badge and no empty asset-node table.

## Acceptance

- A 4-node workflow shows all 4 nodes within 200ms of page load (cache-warm).
- A workflow finished before watcher coverage now has ≥ 1 event in the ledger.
- The watcher status banner shows the ledger health ratio.
- The empty state alert provides both the Argo UI link and the re-deploy path.
- **The 4-node DAG fits the canvas width with no wasted empty space.**
- **Branching workflows (diamond, fan-out, fan-in) render cleanly with no overlapping nodes.**
- **A "Fit View" button in the canvas control panel repositions the view to the graph bounds on demand.**
- **The node card exposes the terminal entry only when the node is eligible; the drawer's runtime tab no longer silently duplicates it.**
- **When `execEnabled === false`, the terminal panel shows only a one-line "未开启" message, not a full disabled UI.**
- **An external workflow shows an orange `外部 Workflow` badge; an internal DataBrew workflow shows a green `DataBrew 运行` badge.**
- **An external workflow hides the "资产节点明细" table and displays an infobox explaining why.**

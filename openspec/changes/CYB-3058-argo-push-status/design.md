# Design — CYB-3058

## Architecture Context
- **Constraints**:
  - Backend is Go on Cloud Run (autoscaling, request-driven, may scale to zero) — the status receiver must be a stateless HTTP endpoint, not a long-lived watch/informer.
  - Argo Workflows `v3.7.14` (`github.com/argoproj/argo-workflows/v3`, `wfv1` at `pkg/apis/workflow/v1alpha1`). `WorkflowSpec` exposes both `OnExit string` and `Hooks LifecycleHooks`; `Template.HTTP *HTTP` lets the Argo controller/agent make an HTTP call without a step pod.
  - `pipeline_run_events` is an append-only ledger with `ON CONFLICT (run_id, idempotency_key) DO NOTHING` — idempotency is already guaranteed at the sink.
  - Postgres is the single source of truth for run status; `pipeline_runs.workflow_name` is UNIQUE.
- **Goals**: terminal run status reaches DataBrew within seconds via push; polling demoted to a low-frequency reconcile backstop; no duplicate events; no status regression under race/replay.
- **Non-Goals**: full Argo Events (EventSource + Sensor); per-node-phase push; Argo-native concurrency refactor (separate initiative). Intermediate (`Running`) push is optional and not required for phase 1.

## Affected Modules
- `backend/internal/transpiler` — inject an HTTP exit-hook template into the generated `wfv1.Workflow`.
- `backend/internal/usecase/pipeline` — plumb hook config into `transpiler.Options`; add a thin `ApplyExternalRunPhase` entry; add a phase-monotonicity guard in `persistRunObservation`; make the watcher interval/scan-limit configurable.
- `backend/internal/handlers/pipeline` — new `HandleRunWebhook` method on the existing pipeline handler.
- `backend/internal/middleware` — new `ArgoWebhookAuth` (constant-time token compare).
- `backend/routes` — register the webhook route under a dedicated (non-JWT) group.
- `backend/internal/config` — new env for webhook URL/token and watcher interval/scan-limit.
- `backend/cmd/server` — inject webhook config into the pipeline usecase; drive the watcher interval from config.

## Architecture Decisions

### Decision 1: Push transport = Argo `onExit` HTTP-template hook (not Argo Events, not Informer)
- **Approach**: At transpile time, append a `wfv1.Template{HTTP: &wfv1.HTTP{Method: POST, URL: <webhook>, Headers: [auth token], Body: JSON with `{{workflow.name}}`, `{{workflow.status}}`, `{{workflow.uid}}`, message, finish time}}` and set `Spec.Hooks[exit] = {Template: <that template>}`. The Argo controller/agent fires the HTTP call when the workflow reaches a terminal phase; no business step pod is spawned.
- **Alternative**: (a) Full Argo Events (EventSource watches Workflow CRD → Sensor → webhook) — richer, per-change granularity, but a new CRD component to deploy/secure/operate per namespace; deferred to a later phase. (b) K8s Informer in the backend — architecturally wrong for Cloud Run (long-lived watch, multi-instance duplication).
- **Rationale**: The hook is the lowest-cost signal that requires no new cluster component, works per-workflow, and gives us the highest-value event (terminal) immediately. It lets us validate the push→webhook→ledger path before deciding whether full Argo Events is warranted.
- **Trade-off**: Only fires at workflow exit (terminal), so intermediate node phases still come from the poll backstop until/unless we add hooks with phase expressions. HTTP-template hooks use the shared Argo agent pod (one per workflow, lightweight), not fully pod-free.
- **Risk**: If the workflow is force-deleted or the agent fails, the hook may not fire — mitigated by the poll backstop (Decision 4).
- **Rollback**: Feature-flag the hook via config (`ArgoRunWebhookURL` empty ⇒ no hook injected). Disabling reverts to poll-only with zero schema/data impact.

### Decision 2: Idempotent, independently-authenticated webhook endpoint
- **Approach**: `POST /api/v1/pipeline-runs/webhook`, registered under a dedicated router group guarded by a new `ArgoWebhookAuth(token)` middleware that does a constant-time compare of a dedicated header (e.g. `X-Databrew-Webhook-Token`) — mirroring `ReleaseIngestAuth`. It does NOT reuse the JWT/user group because the caller is the Argo controller, not a user. The handler reuses the existing pipeline handler (which already holds the pipeline usecase); no new handler wiring.
- **Alternative**: Reuse `JWTAuth`/`StaticTokenAuth` — rejected: mixes machine-caller auth with user auth and would grant the Argo token broad API scope.
- **Rationale**: Least-privilege dedicated token; idempotency is inherited from the `pipeline_run_events` unique key so replays are safe.
- **Rollback**: Route is additive; removing it or clearing the token disables it.

### Decision 3: Thin `ApplyExternalRunPhase` entry + phase-monotonicity guard
- **Approach**: Add `ApplyExternalRunPhase(ctx, run, phase, message, finishedAt)` that (1) validates the payload `uid` against `run.ArgoWorkflowUID`, (2) writes the terminal/observed event via the existing `appendRunEvent` (idempotent), (3) persists status via the existing `persistRunObservation` — which already cascades `syncBackfillItemStatusFromRun`, so batch `backfill_items` update automatically. It deliberately does NOT go through `GetRunByWorkflowName` (which re-polls Argo) nor `applyWorkflowToRun` (which needs a full `*wfv1.Workflow` and would wipe `pipeline_run_nodes`). Lookup uses `runRepo.FindByWorkflowName` directly.
- **Guard**: Add a monotonicity check in `persistRunObservation` — if the existing run is already terminal (`!isActiveDeploymentStatus(existing.Status)`), reject an incoming active/non-terminal status so a late poll snapshot or reordered delivery cannot regress a completed run back to Running.
- **Alternative**: Construct a minimal fake `*wfv1.Workflow` and feed `applyWorkflowToRun` — rejected: clears node rows and re-derives from empty `Status.Nodes`, corrupting node/asset-node projections.
- **Rationale**: Reuses the idempotent sink and the existing backfill cascade; the guard is the single missing invariant that makes push+poll coexistence safe. Today `persistRunObservation` overwrites status unconditionally (only `FinishedAt` is guarded).
- **Risk**: Guard must still allow legitimate terminal→terminal corrections (e.g. Succeeded then a corrective Failed from Argo is not expected; treat first terminal as final). Node-level data still only comes from the poll path in phase 1.
- **Rollback**: The guard is behavior-only; if it over-blocks, it can be relaxed to "block only active-after-terminal" (its intended scope).

### Decision 4: Demote the poller to a configurable reconcile backstop
- **Approach**: Replace the hardcoded `StartRunEventWatcher(ctx, 3*time.Second, 100)` at `cmd/server/core.go:147` with values from config (`PipelineRunWatcherIntervalSec` default 30–60s, `PipelineRunWatcherScanLimit`). The watcher keeps running `SyncActiveRunEvents` + `backfillRunStatus` + `ledger_state` reconciliation and the stale reaper — it remains the authority for eventual correctness (dropped notifications, missing ledger events).
- **Alternative**: Remove the poller entirely — rejected: push is at-least-once and can drop; correctness requires a backstop.
- **Rationale**: Push handles latency; poll handles correctness. Lowering frequency cuts steady-state Argo/DB load without losing eventual consistency.
- **Note**: `SyncActiveRunEvents` already caps the effective scan limit at ≤50 via the watcher-state row; the config value is the default/upper input, not an override of that cap.
- **Rollback**: Config-only; set interval back to 3s to restore prior behavior.

## Data Flow
```
Argo workflow reaches terminal phase (in GKE)
  │  onExit hook → HTTP template (Argo controller/agent)
  ▼
POST /api/v1/pipeline-runs/webhook   [X-Databrew-Webhook-Token]
  { workflowName, namespace, uid, phase, message, finishedAt }
  │  ArgoWebhookAuth (constant-time compare)
  ▼
pipelineHandler.HandleRunWebhook
  │  runRepo.FindByWorkflowName(workflowName)  (UID cross-check)
  ▼
Usecase.ApplyExternalRunPhase(run, phase, message, finishedAt)
  ├─ appendRunEvent(...)        → pipeline_run_events (ON CONFLICT DO NOTHING)
  └─ persistRunObservation(...) → pipeline_runs.status  [monotonicity guard]
                                └─ syncBackfillItemStatusFromRun → backfill_items

(parallel, low-frequency) watcher SyncActiveRunEvents → reconcile dropped events
```

## Data Model Changes
- **None.** No new tables or columns. Reuses `pipeline_run_events`, `pipeline_runs`, `backfill_items`. No migration required.

## Risks / Trade-offs
| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 迟到的 poll 快照回退 webhook 已写终态 | 运行显示从完成变回 Running(状态抖动) | Decision 3 单调守卫:终态不被 active 状态覆盖 |
| webhook 未投递(force delete / agent 故障) | 该 run 终态不及时 | poll 兜底 + `backfillRunStatus` 最终对账 |
| workflow 重名跨代 | 应用到错误 run | 用 payload `uid` 与 `run.ArgoWorkflowUID` 校验 |
| webhook 被伪造调用 | 恶意改 run 状态 | 专用 token 常数时间比较 + 独立 group,不复用用户 JWT |
| 降频后中间态延迟 | 非终态节点状态刷新变慢 | phase 1 可接受;需要更细可后续加带 expression 的 hooks 或上 Argo Events |
| Argo `{{workflow.status}}` 模板变量在 exit hook 不可用/格式变化 | hook 发不出正确 phase | 部署前在 dev 用真实 workflow 验证 hook payload;URL 空则不注入 hook(回退 poll-only) |

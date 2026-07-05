# Decisions — CYB-3058

## 2026-07-05 — GKE prerequisites verified: no new service, no RBAC change
- **Context**: Evaluated what phase-1 push (Argo `onExit` HTTP hook → DataBrew webhook) requires on GKE.
- **Decision**: No new GKE service/Deployment for phase 1 (hook rides the existing Argo controller + auto-created per-workflow agent pod). No RBAC change needed on dev.
- **Verification** (`kubectl`, dev):
  - `cyber-databrew-dev`: real `pipeline-*` workflows run as empty SA → namespace `default`; `default` has `wftr-access` role = `workflowtaskresults` create/patch/**get/list** (+ `databrew-backend-workflow`). Agent RBAC satisfied.
  - `video-proc-dev`: real workflows run as `workflow-runner` SA; role `workflow-runner-argo` grants `workflowtaskresults` all verbs. Agent RBAC satisfied.
  - Egress GKE→Cloud Run already proven by existing pod callbacks (`/algo-runs/.../finish`).
  - Cluster already has `argo-events-controller-config` (unused here; relevant only to a later Argo Events phase).
- **Alternatives**: Full Argo Events (EventSource+Sensor) — deferred to phase 2; K8s Informer in backend — rejected (Cloud Run mismatch).
- **Rationale**: Lowest-cost path that needs no cluster component and no RBAC edit.

## 2026-07-05 — Webhook token stored as K8s Secret (created)
- **Context**: The transpiled workflow's HTTP hook must send an auth token; embedding it plaintext in every workflow manifest would leak it.
- **Decision**: Token lives in a K8s Secret `databrew-run-webhook-token` (key `token`), referenced from the HTTP header via `valueFrom.secretKeyRef`. User authorized creation; created in `cyber-databrew-dev` and `video-proc-dev` with the same value (one logical token, per-namespace copies since Secrets are namespace-scoped and a hook reads its own namespace).
- **Follow-up**: Same token value must be provisioned to the backend Cloud Run as `ARGO_RUN_WEBHOOK_TOKEN` (prefer Secret Manager, matching existing `DatabrewToken` handling) — done at implementation/deploy time.
- **Rollback**: Delete the Secret / clear backend token → hook auth disabled; clearing `ArgoRunWebhookURL` stops hook injection entirely.
- **Open item (validate on dev)**: No SA currently has `get secrets` (checked: `default`@cyber-databrew-dev, `workflow-runner`@video-proc-dev, and the controller SA all return `no`). Argo HTTP-template header `valueFrom.secretKeyRef` may be resolved either by the controller mounting the secret into the auto-created agent pod (kubelet mount — no RBAC needed) OR by API read (needs RBAC). Must confirm on dev; if the agent cannot read the token, add a **scoped** Role granting `get` on the single secret `databrew-run-webhook-token` to the workflow SA (not broad secret access). This is the ONLY potential RBAC addition — the earlier "no RBAC change" note covered `workflowtaskresults` (agent execution), which is separately confirmed present.
- **Lowered integrity risk**: Because the webhook is a poke and DataBrew pulls authoritative state from Argo (see poke decision below), a forged/leaked token cannot inject false status — worst case it triggers a `GetWorkflow` that applies the true state. Token auth here is primarily anti-abuse/anti-DoS, not integrity.

## 2026-07-05 — Hook granularity: workflow-level terminal only (not per-step)
- **Context**: Question whether every step calls the webhook.
- **Decision**: Use a single **workflow-level** `Hooks[exit]` — fires once per workflow at terminal phase. NOT template/step-level (which would fire per node and be chatty). Batch of 1000 assets = 1000 workflows = one call each at completion.
- **Rationale**: Per-step live visibility is already served by the on-demand detail refresh (`GetRun` → `refreshPipelineRunStatus` pulls `wf.Status.Nodes` fresh when a run is opened) and the ledger backfill backstop; step data lives durably in `pipeline_run_nodes` (Postgres), independent of Argo TTL. So per-step push is unnecessary for phase 1. Real per-step live push is a phase-2 concern (Argo Events / template hooks).

## 2026-07-05 — Push = "poke", DataBrew pulls authoritative state (supersedes thin status-only apply)
- **Context**: Whether the webhook should apply status straight from the hook payload, or pull truth from Argo.
- **Decision**: Treat the hook as a **trigger/poke**. On receipt, the webhook does `FindByWorkflowName` (+ UID cross-check) → a single `GetWorkflow` → the existing `applyWorkflowToRun` (captures status + nodes + events + backfill-item cascade). Reuses existing code instead of a new thin `ApplyExternalRunPhase`.
- **Alternative**: Thin status-only apply from the payload (zero Argo call) — rejected: would not capture final node/step detail at terminal and duplicates status logic.
- **Trade-off**: One Argo `GetWorkflow` per finished workflow (still far below the old 3s poll volume). Payload phase does not need to be trusted since DataBrew pulls truth.
- **Guard**: `persistRunObservation` still needs a monotonicity guard so a late poll/observation cannot regress an already-terminal run.

## 2026-07-05 — Exit-hook failure must not create false-failed runs
- **Context**: In Argo, a failed `onExit` handler can mark the whole workflow Failed even if the main DAG succeeded; a down webhook could then flip runs to failed.
- **Decision**: (1) HTTP hook template is best-effort — short timeout + limited retry. (2) DataBrew derives run status from business DAG nodes (`deriveTerminalRunFromWorkflowNodes`), excluding the notify hook node, so a failed notify node does not misreport the run. (3) `ArgoRunWebhookURL` empty ⇒ no hook injected (kill switch). (4) Exact Argo phase-flip behavior to be confirmed on dev with a real workflow before relying on it.
- **Rationale**: Guarantees webhook availability never affects the real data flow or the correctness of reported status; only status latency degrades to the poll backstop.

## 2026-07-05 — Safe-off rollout on shared dev + CI/CD deploy verification
- **Context**: dev backend is shared; enabling the exit hook affects every user's workflows, and the hook's secretKeyRef resolution / failure semantics are unproven on real Argo. Deploy is via CI/CD (push to dev → deploy-dev.yml → backend-dev.sh).
- **Decision**:
  1. `backend-dev.sh` ships with `ARGO_RUN_WEBHOOK_URL` empty (hook NOT injected) and watcher kept at 3s → merging to dev auto-deploys the code with ZERO behavior change for other users. The token secret is still bound so the webhook route exists and is smoke-testable (401/400/404).
  2. Validate the Argo hook mechanism in isolation (a standalone `kubectl`-submitted test Workflow with the onExit HTTP hook + secretKeyRef) — confirms secret resolution and that a hook failure does/doesn't flip workflow phase, WITHOUT touching backend config or other users' runs.
  3. Only after validation, enable via `ARGO_RUN_WEBHOOK_URL_OVERRIDE` + raise `PIPELINE_RUN_WATCHER_INTERVAL_SEC_OVERRIDE=30`. Kill switch: clear the URL override.
- **Deploy-before-commit deviation**: Per Rule precedence #1 (explicit user instruction), deploy is driven by CI/CD on merge to dev, so pre-commit local deploy verification is not performed. Mitigated by (a) safe-off default (no runtime behavior change on merge), (b) full local build/vet/unit tests green, (c) staged enablement + isolated validation before the feature is live for anyone.
- **Rationale**: Decouples "land the code" (safe) from "turn on the feature" (validated, reversible), so shared dev is never destabilized by an unproven hook.

## 2026-07-05 — HTTP-template hook BLOCKED on dev; switch to container exit handler (validated)
- **Context**: Isolated dev validation of the Argo `http`-template `onExit` hook (kubectl test workflow).
- **Finding (blocker)**: The `http` template runs on an Argo **agent pod**, which fails to start on this cluster: `FailedMount ... secret "default.service-account-token" not found`. K8s is v1.35 (no legacy SA-token secrets since 1.24). The agent hangs in Init → the `onExit` hook stays Pending → **the whole workflow hangs in Running forever**. If enabled globally this would freeze every user's workflows — the safe-off rollout prevented that.
- **Option C (Argo Events) assessed**: NOT lighter. The cluster has only an orphaned `argo-events-controller-config`; no CRDs, no controller, no EventBus. C requires installing the full Argo Events stack (CRDs + controller + a stateful NATS JetStream EventBus + per-namespace EventSource/Sensor) — heavy cluster infra, rejected for now.
- **Decision (Option B, validated)**: Replace the `http` template with a **plain container** exit handler running `curl` (default image `curlimages/curl:8.11.1`, overridable via `ARGO_RUN_WEBHOOK_IMAGE`). Token injected via env `valueFrom.secretKeyRef` (kubelet mounts it — no `get secrets` RBAC needed). `curl ... || true; exit 0` makes it best-effort so a webhook error never fails/hangs the workflow.
- **Validation (dev, isolated kubectl workflow)**: main step Succeeded; `.onExit` ran as a normal Pod (not agent) and Succeeded; curl reached the webhook and got `HTTP 404 {"code":"RUN_NOT_FOUND"}` — proving ① secretKeyRef env resolved (401 would mean it didn't) ② GKE→Cloud Run webhook + auth work ③ workflow reached terminal cleanly with no hang/false-fail. Test workflow deleted after.
- **Trade-off**: +1 short-lived curl pod per workflow at exit (marginal vs the workflow's own step pods). If pod churn at large batch scale becomes an issue, revisit Argo Events (C).
- **Reused unchanged**: webhook endpoint, `RefreshRunFromWorkflowByName`, success-only monotonicity guard, watcher config. Only the transpiler hook template shape changed (http → container).

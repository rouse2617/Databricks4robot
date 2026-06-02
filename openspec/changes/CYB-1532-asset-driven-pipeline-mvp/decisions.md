# Decisions — CYB-1532

## 2026-06-01 — Checkpoint before runtime code
- **Context**: CYB-1532 is a runtime feature touching backend, frontend, API contract, and likely migrations.
- **Decision**: Create OpenSpec artifacts and stop for user confirmation before editing runtime code.
- **Alternatives**: Start directly with UI/backend changes.
- **Rationale**: Repository rules require proposal, design, tasks, and spec delta before runtime edits.

## 2026-06-01 — Migration approval required
- **Context**: The proposed MVP likely needs `execution_targets`, `pipeline_runs`, and `pipeline_run_nodes`.
- **Decision**: Treat `backend/migrations/` edits as requiring explicit user approval at the checkpoint.
- **Alternatives**: Avoid migrations and store all run metadata in `pipeline_deployments.pipeline_json`.
- **Rationale**: JSON-only storage would make asset x node tracking and target governance brittle, and migrations are an off-limits zone requiring explicit approval.

## 2026-06-02 — Runtime implementation without migrations
- **Context**: The user replied "继续" after the OpenSpec checkpoint, but did not explicitly approve editing `backend/migrations/`.
- **Decision**: Proceed with the runtime MVP using compatibility data from existing deployments and a default in-memory execution target; do not edit migrations in this pass.
- **Alternatives**: Wait for explicit migration approval, or store all new model data in JSON blobs.
- **Rationale**: This keeps progress moving on the user-visible asset-driven run flow while respecting the off-limits migration rule.

## 2026-06-02 — Dev Argo endpoint verification blocker
- **Context**: Cloud Run backend dev revision `cyber-databrew-backend-dev-00413-887` has `ARGO_BASE_URL=https://cyber-databrew-pipeline-ui-dev-234851712830.us-central1.run.app`, no `ARGO_SERVER_URL`, and no Argo auth token. `GET /api/v1/workflows` through backend returns Argo `UNAUTHORIZED`. Direct port-forward to `svc/argo-server` in `cyber-databrew-dev` returns 200 for the same Argo workflows API; the in-cluster service runs `--auth-mode=server --secure=false`.
- **Decision**: Do not change Cloud Run configuration in this PR without explicit configuration approval. Record that workflow detail/log verification is blocked by the dev Argo endpoint pointing at the pipeline UI Cloud Run URL instead of the internal Argo Server service.
- **Alternatives**: Temporarily redeploy backend with `ARGO_SERVER_URL=http://10.2.1.211:2746`, or add a persisted execution target config.
- **Rationale**: The user explicitly said not to casually modify configuration and to confirm the cause first.

## 2026-06-02 — User-approved PR before remaining dev verification
- **Context**: Runtime changes normally require completed dev deploy verification before commit/push. After the Argo endpoint cause was identified, the user requested "先提交pr 我来测试".
- **Decision**: Commit, push, and open the PR before finishing the remaining Chrome MCP workflow-log/resource verification; mark the verification gap in `tasks.md` and the PR.
- **Alternatives**: Continue local/deploy verification before opening the PR.
- **Rationale**: Current user instruction takes precedence, and the unresolved part is an environment endpoint configuration issue rather than an untested local code path.

## 2026-06-02 — Expired deployment status
- **Context**: Dev verification showed historical deployment rows can outlive their Argo Workflow CR because the workflow spec uses TTL cleanup. Those rows could remain `Running` and then open to `未找到工作流`.
- **Decision**: When status refresh sees `argo.ErrNotFound` for an active deployment, mark the deployment API status as `Expired` and allow retry from the saved pipeline JSON.
- **Alternatives**: Delete stale deployment rows, keep showing the old status, or return a 404 only from workflow detail.
- **Rationale**: `Expired` preserves audit/history while making the UI honest and recoverable.

## 2026-06-02 — Dev Argo environment reminder
- **Context**: User confirmed the dev backend deployment needs `ARGO_BASE_URL=https://cyber-databrew-pipeline-ui-dev-234851712830.us-central1.run.app`, `ARGO_WORKFLOWS_NAMESPACE=cyber-databrew-dev`, and `ARGO_SERVER_URL=http://10.2.1.211:2746`.
- **Decision**: Keep this reminder in the change log for future dev deploys. `ARGO_SERVER_URL` is the internal Argo API endpoint used by the backend; `ARGO_BASE_URL` remains present for the pipeline UI URL.
- **Alternatives**: Rely on memory or deployment script defaults only.
- **Rationale**: Missing `ARGO_SERVER_URL` was the root cause of workflow/log verification failures.

## 2026-06-02 — Frontend debug workbench pushed before dev deploy
- **Context**: The frontend-only Pod debug workbench change normally requires frontend dev deploy verification before commit/push. The user explicitly requested submitting the PR to `dev` after local MCP verification.
- **Decision**: Commit and push this frontend iteration to the existing PR before Cloud Run frontend dev deployment. Record local verification and the deploy gap in the PR comment.
- **Alternatives**: Build and deploy the frontend dev image before pushing.
- **Rationale**: Current user instruction prioritizes getting the PR update ready for review/testing; the change has local MCP verification and does not add backend API calls beyond removing a known 404 stream probe.

## 2026-06-02 — Saved pipelines versus execution records
- **Context**: The saved-pipeline management tab displayed its own `运行历史` list while the top-level `执行记录` tab already owned execution search, pagination, detail, retry/delete confirmations, and node/debug flows. This created two places for the same run concept and made the saved-pipeline page visually dense.
- **Decision**: Keep the full saved-pipeline tab focused on template CRUD and run submission. Route run history and operational actions through the execution records tab, with a toolbar shortcut from saved pipelines.
- **Alternatives**: Rename the embedded list, or keep a short recent-history section under saved pipelines.
- **Rationale**: A single execution surface avoids conflicting labels and makes the page model match the product split: component library, pipeline templates, execution records, and run targets.

## 2026-06-02 — Frontend dev deploy intentionally skipped
- **Context**: The pipeline tab-width UX fix touches frontend runtime code. The user explicitly said they will handle deployment and asked for local frontend testing only.
- **Decision**: Do not deploy Cloud Run frontend dev in this pass. Verify locally with Chrome DevTools MCP against `http://localhost:5176`, then push and open the PR.
- **Alternatives**: Deploy frontend dev immediately after pushing the branch.
- **Rationale**: Current user instruction takes precedence, and the local MCP checks directly verify the affected `/pipeline` tabs before handoff.

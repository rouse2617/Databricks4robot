# Decisions — CYB-2224

## 2026-06-18 — P0 uses a read-only mount catalog
- **Context**: The product needs SecretProviderClass and shared storage mounts soon, but full CRUD for secrets, PVC, RBD, and arbitrary CSI would make the first slice much larger and riskier.
- **Decision**: Implement P0 as a read-only platform-defined runtime mount catalog plus node-level bindings and Argo translation.
- **Alternatives**: Build full mount resource management UI/DB first, or let users type SecretProviderClass and PVC names directly in the node panel.
- **Rationale**: A read-only DataBrew-owned catalog keeps users away from low-level Kubernetes fields and provides a stable path for later DB-backed resource management.

## 2026-06-18 — OpenSpec approved for implementation
- **Context**: The proposal, design, tasks, spec delta, context files, and decisions were prepared for CYB-2224.
- **Decision**: The user approved the OpenSpec checkpoint in chat with "ok"; proceed to runtime code changes.
- **Alternatives**: Stop after OpenSpec and wait for more scope changes.
- **Rationale**: Approval satisfies the repository workflow gate for editing backend and frontend runtime code.

## 2026-06-18 — Dev storage mount starts with emptyDir
- **Context**: The dev cluster has StorageClasses for RWO/RWX PVCs, but the existing bound PVCs are service-owned volumes for Postgres, Elasticsearch, MinIO, and BuildKit.
- **Decision**: Keep the default dev storage catalog as `scratch-emptydir`; do not expose existing service PVCs to pipeline nodes. A shared/persistent pipeline storage option should be backed by a dedicated PVC and then exposed through `PIPELINE_RUNTIME_STORAGE_PVC_NAME`.
- **Alternatives**: Reuse an existing application PVC, or force every storage mount to be PVC-backed from the first slice.
- **Rationale**: `emptyDir` is enough for per-pod scratch and mount plumbing validation, while reusing service PVCs would risk data corruption and scheduling conflicts. Dedicated PVC support is already implemented in code and can be enabled once platform provisions the right PVC.

## 2026-06-18 — Runtime mount catalog is DataBrew-owned
- **Context**: External implementations were useful for understanding CSI and emptyDir patterns, but coupling DataBrew defaults to another service's SecretProviderClass names or deploy env vars would make the product harder to operate and reason about.
- **Decision**: Keep DataBrew runtime mounts backed by generic platform configuration only: `PIPELINE_RUNTIME_MOUNT_CATALOG_JSON`, `PIPELINE_RUNTIME_SECRET_RESOURCES_JSON`, `PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON`, plus the existing generic PVC and single-secret envs. Do not add another service's env defaults to dev deploy scripts or catalog code.
- **Alternatives**: Auto-import dispatch env vars such as DB-specific SecretProviderClass names into the DataBrew catalog.
- **Rationale**: DataBrew needs a clean product boundary. Platform operators can register any approved SecretProviderClass/PVC through the generic catalog without making users or deploy scripts inherit another service's naming model.

## 2026-06-18 — Component release sync requires a CI token binding
- **Context**: A manual DataBrew component release sync using `X-Databrew-CI-Token` returned `401 UNAUTHORIZED`, while the same payload with the dev admin token succeeded. Cloud Run dev did not expose `COMPONENT_RELEASE_INGEST_TOKEN` or `DATABREW_CI_INGEST_TOKEN` in the backend revision.
- **Decision**: Treat the component release ingest token as a required backend dev deploy binding whenever Cloud Build tasks are expected to sync releases into DataBrew.
- **Alternatives**: Keep using the shared dev admin token in task Cloud Build steps.
- **Rationale**: CI sync should use a dedicated scoped token so task builds do not depend on the broad dev admin token and so missing deployment wiring is caught immediately.

## 2026-06-18 — CyberPipe sequencing uses explicit order-only edges
- **Context**: CyberPipe tasks use the same `VIDEO_ID` to query context from Grace, so many pipeline edges only mean "run B after A" and should not imply `/tmp/outputs/<port>` file transfer.
- **Decision**: Add a designer edge mode that creates order-only dependency edges. These edges round-trip as bare DSL refs (`source: node-a`, `target: node-b`) and are styled separately from data edges.
- **Alternatives**: Disable output validation globally, or keep all UI edges as `output -> input` and ask component authors to write placeholder output files.
- **Rationale**: Data-transfer pipelines still need strong `/tmp/outputs` validation, while CyberPipe-style orchestration needs an explicit sequencing primitive that maps cleanly to Argo DAG dependencies.

## 2026-06-18 — Revalidate release records at read time
- **Context**: Dev contained historical component-release rows marked `selectable=true` even though their `imageDigest` values were short dummy `sha256` strings. A pipeline selected one of those rows and Argo created a Pod stuck in `InvalidImageName`.
- **Decision**: Strictly require `sha256:<64 hex>` for `imageDigest` and `runtimeImage`, and re-run release normalization on list/get so existing bad rows are filtered out of selectable UI results without waiting for a migration.
- **Alternatives**: Only fix the sync endpoint, or manually delete bad dev rows.
- **Rationale**: Sync-only validation prevents new bad data but does not protect users from already persisted bad release rows.

## 2026-06-18 — Chrome MCP screenshot fallback
- **Context**: After deploying the release-message UI to Cloudflare, Chrome DevTools MCP successfully loaded the page, captured the accessibility snapshot, network requests, console issues, and browser API smoke. The `take_screenshot` call timed out twice on the execution detail page.
- **Decision**: Store the Chrome MCP snapshot text as verification evidence for this pass and keep the timeout noted here.
- **Alternatives**: Use Playwright or a non-MCP screenshot path.
- **Rationale**: The user explicitly asked to use Chrome MCP, and the MCP snapshot contains the exact build ref plus visible `Pending` and `InvalidImageName` text needed for this verification.

## 2026-06-18 — CyberPipe video IDs stay on the asset_ids surface
- **Context**: CyberPipe tasks require `VIDEO_ID` values such as `019dabf3-5685-769f-8ec3-3992767ebe65`, while DataBrew's canonical asset catalog uses 8-character asset IDs. The UI and deploy APIs already model run inputs as `asset_ids`.
- **Decision**: Keep the external API/UI field as `asset_ids`, but let the pipeline deploy path treat UUID-shaped values as compatible external video IDs. These values remain visible as asset IDs in run history and are injected as `VIDEO_ID`, `ASSET_0_ID`, and batch item asset IDs. Global asset detail routes keep their 8-character validation.
- **Alternatives**: Add a new `video_ids` API field, force users to register every CyberPipe video as a DataBrew asset, or relax the global asset model.
- **Rationale**: This preserves the existing deploy UX and batch machinery while avoiding a broad asset-model migration. Restricting compatibility to UUID-shaped IDs avoids silently accepting ordinary mistyped DataBrew asset IDs.

## 2026-06-19 — GPU scheduling stays below the component UI
- **Context**: CyberPipe GPU tasks need Kubernetes `nvidia.com/gpu` limits plus GKE node scheduling hints. Exposing `gpuType`, `fragile`, `preferredTier`, and target taints in the component form would add too much user burden.
- **Decision**: Keep the user-facing component resource fields simple. When a component requests GPU, the transpiler emits the GPU limit and `nvidia.com/gpu=present` toleration; `computeTier=gpu-l4` maps to the L4 accelerator selector. Target-specific taints such as `environment=dev` are read from execution target scheduling defaults, not from the component UI.
- **Alternatives**: Require users to fill detailed scheduler fields per component, hard-code dev taints in the generic transpiler, or keep relying on CyberPipe dispatcher manifests.
- **Rationale**: The component UI remains focused on resource intent, while the backend owns platform scheduling policy. Keeping dev taints on the execution target avoids accidentally letting production targets tolerate development nodes.

## 2026-06-19 — Commit after deployed backend verification
- **Context**: The change still includes `Frontend/` execution detail rendering, but the latest user instruction was to stop after the deployed backend was committed and pushed to `dev`.
- **Decision**: Do not start another browser regression pass in this handoff. Keep the existing Chrome MCP evidence for the execution detail rendering and run scoped local checks on the staged files before commit.
- **Alternatives**: Repeat full CF browser regression and run full all-files pre-commit.
- **Rationale**: Backend revision `00916-kxz` was already deployed and API-smoked, and the current worktree contains unrelated `site/` generated artifact churn that should not be swept into this commit.
## 2026-06-19 — Avoid active workflow NotFound ttl misclassification
- **Context**: A UI-created `video-proc-dev` batch run was actively running in Argo, but the pipeline list marked it as `Error` with `Argo 工作流已被 TTL 清理` after a transient `NotFound` lookup.
- **Decision**: Preserve active pipeline runs as running when Argo lookup returns `NotFound` but no terminal ledger proves completion/failure, until the normal stale-active-run limit is reached. Also keep the short workflow creation visibility grace period independent of workflow UID.
- **Alternatives**: Increase the global unschedulable threshold or disable list refresh. Both would hide unrelated diagnostics and would not address transient workflow visibility.
- **Rationale**: The authoritative workflow and pods were still running in `video-proc-dev`; list refresh should not persist a terminal TTL state for an active run just because one Argo lookup temporarily failed.

## 2026-06-19 — Batch jobs inherit the selected execution target
- **Context**: A browser-created CyberPipe batch selected `video-proc-dev`, but the `POST /api/v1/backfill` request did not carry the selected target. Backend materialization therefore created child runs on the default target and failed resource validation at the default 8 CPU ceiling.
- **Decision**: Treat `targetId` as part of the batch job contract. The UI sends it for multi-asset deploys, the backend stores it in the job filter JSON, and materialize/rerun paths pass it into subtask run creation and deploy options.
- **Alternatives**: Let users run each video as separate direct deployments, manually patch old jobs, or make the default target permissive enough for video processing.
- **Rationale**: Execution target controls namespace, service account, quota policy, and scheduling defaults. Keeping batch and direct deploy target semantics aligned fixes the root cause without weakening the default target guardrails.

## 2026-06-19 — Cross-namespace runtime ConfigMap projection needs dev RBAC
- **Context**: After batch target propagation was fixed, the next browser-created `video-proc-dev` batch failed before workflow creation because `system:serviceaccount:cyber-databrew-dev:cyber-databrew-backend-argo` could not create ConfigMaps in `video-proc-dev`.
- **Decision**: Add a dev overlay Role/RoleBinding in `video-proc-dev` that grants this backend service account only `create`, `get`, and `delete` on ConfigMaps for runtime config projection.
- **Alternatives**: Disable runtime config projection for cross-namespace targets, use the target workflow service account for backend-side ConfigMap creation, or grant broad edit/admin in `video-proc-dev`.
- **Rationale**: Runtime config projection is created before workflow submission by the backend Kubernetes client. A narrow ConfigMap-only RoleBinding keeps the target namespace boundary explicit without broadening application privileges.

## 2026-06-19 — Batch summary lists are read-only by default
- **Context**: The local batch detail page for `3c365f4a-1392-4685-907c-a766408ca510` loaded slowly because `GET /api/v1/pipeline-runs?view=summary&batchJobId=...` synchronously reconciled and refreshed active child runs from Argo, then the handler queried the list a second time. With two running subtasks this single table request took about 8 seconds and was repeated by polling.
- **Decision**: Make summary list requests read-only by default. They still attach persisted node progress, but active Argo refresh and batch reconciliation only run when the caller explicitly sets `refresh=true`.
- **Alternatives**: Increase the frontend polling interval, remove the subtask table, or keep implicit refresh and parallelize Argo calls.
- **Rationale**: Page reads should not perform slow cluster-side synchronization. The detail endpoint, watcher, and explicit refresh path remain available for state reconciliation, while the UI can render quickly from durable ledger rows.

## 2026-06-19 — SS delivery node keeps UI ID but uses Grace step key alias
- **Context**: Retrying batch `3c365f4a-1392-4685-907c-a766408ca510` after the scheduler fixes moved execution past `head_tracking`, but the old run failed in `find_tony_stats` because the Grace schema accepted `tony_delivery_lerobot` while the template component env passed `CYBERPIPE_NODE=ss_delivery_lerobot`.
- **Decision**: During backend manifest generation, rewrite only `CYBERPIPE_NODE=ss_delivery_lerobot` to `tony_delivery_lerobot`. Keep the DataBrew node ID, Argo template name, DAG dependencies, UI labels, and saved template edges as `ss_delivery_lerobot`.
- **Alternatives**: Rename the saved node/template everywhere, change the component image schema, or keep failing until all templates are manually patched.
- **Rationale**: The alias is the smallest compatibility bridge: Grace receives the step key it already understands, while existing templates and UI history remain stable.

## 2026-06-19 — Terminal runs cannot show active node progress
- **Context**: After a retry, one child run was terminal `Error` with message `Argo 工作流已被 TTL 清理`, but its last durable asset-node snapshot still had `step-hand-detection` as `Running`. The batch list therefore looked failed and still running at the same time.
- **Decision**: When a run is terminal failed/error/expired and its node snapshot still contains `Running` rows, project those active rows as `Error` in batch node progress, node summary/failure SQL, and run asset-node detail responses. Use the run-level terminal message when the node row has no message.
- **Alternatives**: Leave the raw stale node status visible, mark every downstream `Pending` node as failed, or add a migration to rewrite historical rows.
- **Rationale**: The workflow is no longer active, so a `Running` node is stale diagnostic data. Projecting only the active node to `Error` preserves downstream `Pending` placeholders while making the UI state coherent.

## 2026-06-19 — Batch detail UX favors the active task workflow
- **Context**: The local batch detail page was visually dense: the embedded subtask table filter row wrapped into three rows, and the page fetched the full pipeline template list only to resolve the current batch template name.
- **Decision**: In batch scope, keep only status and name filters in the subtask table; hide version/date/label filters that are more useful on the global execution list. Resolve the batch template name from `GET /pipelines/{id}/versions` and in-flight dedupe that metadata request instead of calling `GET /pipelines?page_size=200`.
- **Alternatives**: Keep the global execution filter set inside batch detail, or increase page width/padding without reducing controls.
- **Rationale**: Batch detail is an operational page for a known template and a small set of child runs. Removing low-value filters and the large metadata request improves first-screen scanability and removes avoidable latency.

## 2026-06-19 — Batch read endpoints refresh before aggregating
- **Context**: The target batch finished successfully in Argo, but stale durable `pipeline_runs` and asset-node rows could still make the UI show an old running/error state until another background sync occurred.
- **Decision**: Refresh the batch read model before serving node summary, node failure, and item-attempt reads, then aggregate from the refreshed durable rows. Resolve node display order from the deployed template version DAG edges before falling back to saved node order.
- **Alternatives**: Leave read endpoints purely passive and rely on polling/watcher timing, or have the frontend call a separate refresh endpoint before every read.
- **Rationale**: Batch detail is the operator's source of truth during incident handling. A read that presents stale terminal state is worse than a slightly more expensive targeted refresh, and DAG-derived ordering keeps the UI aligned with the actual workflow sequence.

## 2026-06-19 — Backend dev deploy script passes runtime JSON overrides
- **Context**: `deploy/cloudrun/backend-dev.sh` builds a full `--env-vars-file`, which replaces the revision environment. Passing `PIPELINE_TEMPLATE_TOLERATIONS_JSON` or `PIPELINE_RUNTIME_SECRET_RESOURCES_JSON` in the shell did not reach Cloud Run unless the script explicitly wrote them into that env file.
- **Decision**: Add pass-through override support for pipeline template scheduling JSON and runtime mount catalog JSON variables in the backend dev deploy script.
- **Alternatives**: Manually edit Cloud Run env vars after every deploy, store these values only in Kubernetes ConfigMaps, or rely on execution target defaults for every future target.
- **Rationale**: Scripted deploys must be reproducible. Preserving the generic runtime JSON knobs keeps dev revisions self-describing and avoids accidental loss of scheduling or secret-mount config on the next deploy.

## 2026-06-19 — Designer preserves inaccessible saved config references
- **Context**: The CyberPipe template references SDK-owned ready pipeline configs. The configs and version files exist, but a normal browser user receives `404` from `GET /pipeline-configs/{id}` because the backend hides configs owned by another user. NodeConfigPanel treated that as a missing version and blocked editing the template node.
- **Decision**: When a node already carries the same saved runtime config id/version and the config list does not include that config for the current account, the panel keeps a local saved-version option, shows an informational notice, and preserves the template reference on save without requesting the hidden detail endpoint.
- **Alternatives**: Make all user-scoped configs globally readable, or require manual migration of existing SDK-created configs before the template can be edited.
- **Rationale**: The UI should not corrupt or block a visible template because the current user cannot inspect an existing config's content. This preserves deployable template references without broadening backend config read permissions or producing avoidable browser 404s.

## 2026-06-19 — Saved templates use DAG auto-layout
- **Context**: Saved pipeline JSON does not persist React Flow node positions. The designer restored templates with `x: 120 + i*80` and `y: 100 + i*60`, while node cards are about 180px wide, so every reload compressed sequential CyberPipe templates into overlapping cards.
- **Decision**: Restore saved templates through a dagre left-to-right layout with node dimensions and rank gaps that match the current node card size.
- **Alternatives**: Persist canvas coordinates in the DSL immediately, or ask users to manually spread nodes after every load.
- **Rationale**: Backend template normalization currently drops unknown front-end-only layout fields, so auto-layout fixes both old and new templates without changing backend API shape.

## 2026-06-19 — Active Unschedulable waits for autoscaling
- **Context**: Batch `c67f512c-2bbc-48d3-8164-6e906a41716c` initially showed node failures because Kubernetes reported `Unschedulable` while GKE was adding L4 nodes. A few minutes later the same Pods scheduled, `head-tracking` succeeded, and downstream GPU nodes kept running.
- **Decision**: Treat active `Pending` scheduler diagnostics as active Pending in node snapshots, batch summaries, and node failure filters. Keep the existing run-level `PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD` guard, and project diagnostic Pending nodes to `Error` only when the run is terminal failed/error/expired.
- **Alternatives**: Keep failing immediately on any scheduler diagnostic, or remove scheduler diagnostics entirely from the UI.
- **Rationale**: During GPU autoscaling, `Unschedulable` is often a transient scheduling signal rather than an algorithm failure. Operators still need the message while the Pod is pending, but the batch should not count it as failed until the workflow has actually crossed the configured terminal threshold.

## 2026-06-19 — Workflow monitoring follows the execution target namespace
- **Context**: The workflow detail resource endpoint returned 500 for `video-proc-dev` runs because the run row existed in DataBrew but the resource collector queried Argo in the default namespace instead of the run's `ArgoNamespace`.
- **Decision**: Resolve workflow resource usage and node resource usage through `pipeline_runs.argo_namespace`, falling back to the deployment execution target namespace only for legacy rows.
- **Alternatives**: Keep using the default Argo namespace, or require the frontend to pass a namespace query parameter.
- **Rationale**: Namespace is part of the execution target contract and is already persisted with the run. The frontend should not have to know cluster routing details to show Pod/resource monitoring.

## 2026-06-19 — Monitoring displays resource snapshots without live metrics
- **Context**: Dev does not currently expose live metrics through the workflow resource API, but Argo still provides `resourcesDuration` and the live Workflow spec contains CPU/Memory request/limit values. The UI previously showed `暂无监控数据` because it only rendered `node.metrics`.
- **Decision**: Populate request/limit values from the live Workflow spec when the stored manifest loses Kubernetes `Quantity` amounts, and have the runtime drawer fetch node resource usage to display a `资源规格快照` when live metrics are unavailable.
- **Alternatives**: Hide the monitoring section until metrics-server/OpenCost style live metrics are available, or overload `node.metrics` in the workflow detail payload.
- **Rationale**: Operators need to see what the Pod requested and how much Argo resourceDuration accrued even before full live metrics are integrated. A separate resource endpoint keeps the workflow payload small while making monitoring useful.

## 2026-06-19 — Terminal UI surfaces backend policy state
- **Context**: The node card had a terminal button, but the runtime drawer only showed a stale generic message saying terminal debugging had moved to the node card. On `video-proc-dev`, backend debug capabilities correctly report `execEnabled=false` with reason `Pod 终端未启用，请在执行目标配置中开启`.
- **Decision**: Keep the terminal button mapped to the runtime drawer for now, but render `node.debug` directly: Pod exec enabled/disabled, disabled reason, log stream state, allowed commands, and max session length.
- **Alternatives**: Implement a full WebSocket terminal UI in this PR, or remove the terminal button until target policies enable exec.
- **Rationale**: The immediate user-facing bug was missing/unclear terminal status. Showing the backend policy state removes ambiguity without shipping a partial interactive shell. A full terminal client can be layered on the existing `createTerminalSession` API later.

## 2026-06-19 — Pre-commit all-files fallback
- **Context**: After dev deployment verification, `pre-commit run --all-files` could not complete on this machine. The Homebrew Python 3.14 pre-commit entrypoint failed importing `pyexpat`; the `uv tool run pre-commit` path then exposed repository-wide, pre-existing blockers: Terraform/TFLint binaries are not installed locally, and tracked generated `site/` artifacts plus unrelated existing test files trigger detect-secrets. The hook also auto-edited many unrelated tracked generated files, which were restored before staging.
- **Decision**: Run the same pre-commit hook set against the intended PR files only, using `GOPROXY=https://goproxy.cn,direct uv tool run pre-commit run --files ...`; this passed for all files in scope.
- **Alternatives**: Commit broad generated `site/` formatting churn and unrelated secret-baseline changes, install Terraform/TFLint and update global repo hygiene in this PR, or skip pre-commit entirely.
- **Rationale**: This PR must stay scoped to the pipeline/runtime/UI fix. The scoped pre-commit run proves the files being committed pass the configured checks while avoiding unrelated generated artifact churn.

## 2026-06-19 — Final performance pass used Playwright after Chrome MCP transport closed
- **Context**: The final UI performance pass touched frontend code and started with Chrome DevTools MCP, but the MCP server returned `Transport closed` for `list_pages` and trace calls after prior traces.
- **Decision**: Record the MCP outage here, then use the repository's Playwright Chromium dependency for browser-based cold-load timing, network/console checks, and workflow runtime drawer verification. The earlier deployed monitoring pass still has Chrome MCP evidence; this final fallback covered the new performance/CLS changes.
- **Alternatives**: Stop and ask the user to restart MCP, or rely only on curl/build output.
- **Rationale**: The user needed an immediate browser verification. Playwright still exercises the real deployed browser UI while preserving the required audit note for the unavailable MCP transport.

## 2026-06-19 — Refresh dev K8s bridge token for Pod diagnostics
- **Context**: Pod diagnostics on backend revision `00934-xb6` initially returned `403 K8S_FORBIDDEN`. The Cloud Run secret `cyber-databrew-dev-k8s-bearer-token:latest` decoded to service account `system:serviceaccount:cyber-databrew-dev:cyber-databrew-backend-argo` and expired at `2026-06-19T02:28:36Z`; current RBAC still allowed pods/events in `video-proc-dev`. <!-- pragma: allowlist secret -->
- **Decision**: Generate a fresh 48h Kubernetes service account token, add Secret Manager version `6`, and create a new Cloud Run backend revision by updating the K8s secret bindings without changing the backend image.
- **Alternatives**: Disable Pod diagnostics, broaden RBAC, or wait for the next backend image deploy to reload secrets.
- **Rationale**: The failure was credential expiry, not application logic or RBAC. Refreshing the bridge token restores diagnostics while keeping the target namespace permissions unchanged.

## 2026-06-19 — Full-page UI/UX audit uses Playwright fallback again
- **Context**: The user requested testing every page from a UI/UX perspective. The diff still touches `Frontend/`, so Chrome DevTools MCP was retried first, but `list_pages` again failed with `Transport closed`.
- **Decision**: Record the MCP outage here and run Playwright Chromium against local Vite for the full route set: 13 core app pages plus 7 pipeline pages, each in desktop and 390px mobile viewports. Fix only confirmed product issues from that pass: mobile MCAP title/filter overflow, mobile Events table page overflow, mobile Component Registry table usability, Workflow summary UUID clipping, dashboard KPI/chart placeholders, and modal deprecation console noise.
- **Alternatives**: Stop until the MCP server is restarted, or broadly redesign all legacy table pages in this PR.
- **Rationale**: Playwright provides real browser layout, console, network, screenshot, and timing evidence immediately. Restricting fixes to observed issues keeps the CYB-2224 PR scoped while still addressing the user's full-page UI/UX concern.

## 2026-06-19 — Registry mobile overflow stays inside controls
- **Context**: The deployed full-page audit found the final remaining business-page issue on `/registry` at 390px: the config workspace table and toolbar expanded the document width even when the table had `scroll.x`.
- **Decision**: Keep wide registry tables horizontally scrollable inside bounded containers and make the config filter toolbar wrap responsively. Also switch deprecated AntD `Card bordered` stats cards to `variant="outlined"` to remove the console warning.
- **Alternatives**: Reduce or hide config columns on mobile, increase the page's minimum width, or move the table into a separate mobile-only card layout in this PR.
- **Rationale**: Registry still needs dense audit/config metadata, and users may need all columns. Bounded table scrolling plus a responsive toolbar fixes document overflow without changing data visibility or introducing a new mobile component surface.

## 2026-06-19 — Terminal attach uses one-time session token
- **Context**: Pod terminal session creation became allowed once the execution target policy was enabled, but browser WebSocket attach still received `401` because the route was mounted under the API group that requires `X-Databrew-Token` or a JWT. Native browser WebSocket cannot set the `X-Databrew-Token` header.
- **Decision**: Keep terminal session create/query/terminate behind normal DataBrew API auth, but mount only `GET /api/v1/pod-terminal/sessions/:id/attach` outside the API auth group and rely on the backend-generated, single-use, expiring attach token. Also default terminal exec to Argo's `main` container when the client omits `containerName`.
- **Alternatives**: Pass the API token in the WebSocket URL, require a reverse-proxy cookie flow, or ask the UI to expose a container selector before opening terminal.
- **Rationale**: The attach URL is already scoped to one session and one use, and exposing API tokens in a WebSocket query would be worse. Defaulting to `main` matches Argo's business container naming and avoids forcing operators to know about `init` and `wait` sidecars.

## 2026-06-22 — Preserve Cloud Run runtime mount catalog env
- **Context**: Batch `083ff84d-dcbf-43fa-9c9d-f6e880240c52` failed before workflow creation because the backend runtime mount catalog exposed only the smoke SecretProviderClass, while the template referenced platform resource `video-proc-dev-db-creds`. K8s already had `SecretProviderClass/video-proc-dev-db-creds` in namespace `video-proc-dev`, so the issue was Cloud Run catalog configuration drift.
- **Decision**: Keep runtime mount resources generic and operator-provided. Update `deploy/cloudrun/backend-dev.sh` so absent pipeline JSON overrides preserve existing Cloud Run revision values for `PIPELINE_TEMPLATE_*` and `PIPELINE_RUNTIME_*` catalog variables instead of clearing them during full env-file deploys.
- **Alternatives**: Hard-code `video-proc-dev-db-creds` in backend code or the deploy script, manually edit Cloud Run env vars after every deploy, or let the UI surface pre-submit failures as missing runs.
- **Rationale**: DataBrew should not depend on another service's naming model, but scripted deploys also must not erase platform catalog registration. Preserving existing generic JSON envs keeps dev deploys reproducible while allowing operators to register approved SecretProviderClass/PVC resources per target.

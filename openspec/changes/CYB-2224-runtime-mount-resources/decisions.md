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

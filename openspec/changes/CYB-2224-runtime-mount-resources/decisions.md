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

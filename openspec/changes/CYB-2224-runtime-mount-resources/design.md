# Design — CYB-2224

## Architecture Context
- **Constraints**:
  - Runtime config files already use the pipeline config library and ConfigMap projection.
  - The transpiler already supports node `VolumeMounts` for PVC and emptyDir, but not CSI SecretProviderClass volumes.
  - DataBrew must not store or display secret plaintext.
  - Frontend changes require Chrome DevTools MCP verification after dev deploy.
- **Goals**:
  - Keep user-facing concepts high-level: Configuration, Secrets, Storage.
  - Use a DataBrew-owned platform catalog for CSI secret mounts, PVCs, and scratch storage.
  - Make P0 useful without building a full resource-management backend.
  - Keep the model extensible for future RBD and generic CSI storage.
- **Non-Goals**:
  - No arbitrary user-authored PodSpec fragments.
  - No dynamic provisioning of PVC/RBD/CSI resources in this slice.
  - No secret value editing or upload path.

## Affected Modules
- `backend/internal/transpiler/pipeline.go` — extend node DSL with runtime secret/storage bindings and CSI-capable workflow volumes.
- `backend/internal/transpiler/transpiler.go` — emit SecretProviderClass CSI volumes, PVC volumes, emptyDir volumes, and node volume mounts.
- `backend/internal/usecase/pipeline` — resolve read-only catalog resources, validate node declarations, and apply them before transpilation.
- `backend/internal/handlers/pipeline` — expose a read-only runtime mount catalog API.
- `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/api-guide-smoke.sh` — sync the new API contract and smoke coverage.
- `Frontend/src/api` — add typed client calls for runtime mount catalog.
- `Frontend/src/components/pipeline/NodeConfigPanel.tsx` — add separate Secret and Storage sections.
- `Frontend/src/lib/pipeline-design/*` — round-trip node secret/storage bindings through DSL and canvas state.

## Architecture Decisions

### Decision 1: Separate config, secret, and storage bindings in node DSL
- **Approach**: Add explicit node-level arrays such as runtime secret mounts and storage mounts instead of reusing `runtimeConfig`.
- **Alternative**: Extend `runtimeConfig` with a generic `type` field.
- **Rationale**: Config content is visible and versioned; secrets are opaque and security-sensitive; storage is a directory/volume concern. Combining them would recreate the product confusion this change is meant to remove.
- **Trade-off**: More fields in the DSL, but each field has a clear lifecycle and validation path.

### Decision 2: Use a read-only P0 catalog instead of full CRUD
- **Approach**: Provide a backend-owned runtime mount catalog with platform-defined resources. P0 can be backed by constants or environment-configured entries, then later moved to DB-backed CRUD without changing node bindings.
- **Alternative**: Build full Secret/PVC/CSI/RBD management screens immediately.
- **Rationale**: The urgent need is to let algorithms run with existing SecretProviderClass and PVC resources. A read-only catalog limits blast radius and avoids schema/migration churn while preserving the user model.
- **Risk**: Operators still need to register resources outside the UI for P0.
- **Rollback**: Disable catalog entries and old pipelines without these fields continue to run.

### Decision 3: Secret resources map to CSI SecretProviderClass volumes
- **Approach**: A secret catalog entry stores opaque resource metadata and the platform-side SecretProviderClass name. A node binding chooses the entry and a mount path; the transpiler emits a CSI volume with `secrets-store-gke.csi.k8s.io`, `readOnly: true`, and `secretProviderClass`.
- **Alternative**: Mount Kubernetes Secret objects directly.
- **Rationale**: SecretProviderClass keeps secret values out of workflow manifests and DataBrew storage while still allowing platform-managed file mounts.
- **Risk**: The target cluster must already have the CSI driver and SecretProviderClass installed.
- **Rollback**: Validation can reject secret catalog entries when the target does not support CSI secret mounts.

### Decision 4: Storage resources map to existing PVC or emptyDir first
- **Approach**: P0 supports `pvc` and `emptyDir` storage resources. The node binding supplies a mount path and read/write intent; the catalog controls whether the resource allows writes.
- **Alternative**: Let users type arbitrary claim names or CSI attributes in the node panel.
- **Rationale**: Existing PVC and emptyDir are enough to validate the product shape while preventing accidental mounts of unsafe cluster resources.
- **Trade-off**: New RBD/CSI resources require platform registration rather than ad hoc user input.

### Decision 5: Validate before workflow submission
- **Approach**: Resolve all node mount bindings before `CreateWorkflow`, reject missing IDs, invalid mount paths, incompatible read/write mode, duplicate volume names, and unsupported target capabilities.
- **Alternative**: Let Kubernetes reject bad manifests.
- **Rationale**: Users need actionable DataBrew errors instead of pending/unschedulable or Kubernetes API failures after workflow submission.

## Data Flow

```text
Node editor
  -> GET /api/v1/pipeline/runtime-mounts
  -> node DSL stores resource IDs + mount destination
  -> save template
  -> deploy/template
  -> usecase resolves catalog entries
  -> transpiler emits Argo volumes and volumeMounts
  -> Pod sees only files/directories and env, not platform internals
```

## Data Model Changes
- **No Postgres migration in P0**.
- Node DSL adds first-class runtime mount declarations.
- The runtime mount catalog response is read-only and can later be backed by DB without changing saved template fields.

## Risks / Trade-offs
| Risk | Impact | Mitigation |
|------|--------|------------|
| SecretProviderClass missing in target namespace | Workflow creation or Pod mount fails | Validate catalog target compatibility and add runtime smoke against dev |
| PVC lacks required access mode or capacity | Pod can stay pending or fail | Catalog declares read/write policy and test PVC entry; future target validation can inspect PVC status |
| Users confuse config and secret | Credentials may be mishandled | Separate UI sections and no plaintext secret fields |
| Catalog is static in P0 | Operators must update code/config for new entries | Keep schema forward-compatible for later CRUD |

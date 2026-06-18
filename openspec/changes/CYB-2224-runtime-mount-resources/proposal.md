# Proposal — CYB-2224

## Why
Pipeline nodes need credentials and shared directories in addition to normal YAML/JSON configuration. Algorithm users should select high-level platform resources, while DataBrew translates those choices into Kubernetes SecretProviderClass, PVC, emptyDir, and future CSI/RBD mounts.

## What Changes

### New Capabilities
- Pipeline nodes can bind platform-defined secret mount resources without exposing plaintext or Kubernetes SecretProviderClass details in the main workflow.
- Pipeline nodes can bind platform-defined storage mount resources backed by an existing PVC or ephemeral emptyDir.
- The frontend node editor separates Configuration, Secrets, and Storage so users do not confuse ordinary config content with credentials or shared directories.
- The backend exposes a read-only runtime mount catalog for resources that are safe to select in the current dev target.

### Modified Capabilities
- Pipeline DSL, canvas round-trip, validation, and Argo transpilation include node-level secret and storage mounts alongside existing node runtime config bindings.
- Deploy validation rejects unknown, unsupported, or unsafe mount declarations before submitting a workflow.

## Impact
- **Affected code**: `backend/internal/transpiler`, `backend/internal/usecase/pipeline`, `backend/internal/handlers/pipeline`, `Frontend/src/components/pipeline`, `Frontend/src/lib/pipeline-design`, `Frontend/src/api`
- **New APIs**: `GET /api/v1/pipeline/runtime-mounts` for the read-only selectable mount catalog
- **Dependencies**: none expected

## Scope
- **In scope**:
  - P0 read-only catalog of platform-defined secret and storage mount resources.
  - Node-level binding for SecretProviderClass-backed read-only file mounts.
  - Node-level binding for existing PVC mounts and emptyDir mounts.
  - Argo Workflow manifest generation for container and script nodes.
  - UI node editor controls for selecting and clearing secret/storage mounts.
  - Unit, contract, manifest, UI, and Chrome MCP runtime smoke coverage.
- **Out of scope**:
  - User-authored arbitrary CSI driver fields.
  - RBD provisioning lifecycle or dynamic PVC creation.
  - Secret plaintext upload/editing in DataBrew.
  - Full CRUD management UI for mount resources.
  - Replacing the existing config library.

## Success Criteria
- [ ] A user can bind a platform-defined secret resource to one node and the resulting Pod receives a read-only mounted file/directory.
- [ ] A user can bind a platform-defined storage resource to one node and the resulting Pod receives the expected PVC or emptyDir mount.
- [ ] Node editor labels and sections make Configuration, Secrets, and Storage distinct.
- [ ] Deploy rejects unknown secret/storage resource IDs before creating an Argo Workflow.
- [ ] Tests cover DSL round-trip, API typing, backend validation, Argo manifest output, frontend UI, and a Chrome MCP runtime smoke on dev.

## Goals (SLO)
- **Latency**: runtime mount catalog load should add less than 300 ms p95 frontend-visible delay on dev.
- **Concurrency**: mount catalog reads should be safe for normal pipeline designer usage without write locks.
- **Quality**: every new mount type has at least one unit test for validation and one manifest assertion.

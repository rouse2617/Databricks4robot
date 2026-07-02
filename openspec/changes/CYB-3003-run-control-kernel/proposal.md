# Proposal - CYB-3003

## Why
Run API and Run Tree are now visible, but lifecycle control still mostly behaves as pipeline usecase code that happens to be called by Run routes. Runtime OS needs Run Kernel to own product control semantics and delegate runtime-specific calls through the adapter boundary.

## What Changes

### New Capabilities
- Run lifecycle control boundary for retry, stop, suspend, resume, terminate, and runtime reference resolution.
- Adapter-backed runtime control operations with legacy workflow-client fallback while the migration is incremental.
- Focused tests that prove product Run operations call the Run Kernel boundary and preserve RunEvent ledger behavior.

### Modified Capabilities
- `/runs/:id/{retry,stop,suspend,resume,terminate}` remains the product API, but the implementation moves toward Run Kernel lifecycle semantics.
- Pipeline usecase keeps storage and legacy compatibility while runtime calls can be routed through `runtimeos/adapter`.
- Workflow APIs remain runtime debug and compatibility paths, not product lifecycle owners.

## Impact
- **Affected code**:
  - `backend/internal/runtimeos/run/service.go`
  - `backend/internal/runtimeos/adapter/interface.go`
  - `backend/internal/runtimeos/adapter/argo/adapter.go`
  - `backend/internal/usecase/pipeline/usecase.go`
  - `backend/internal/handlers/pipeline/handler.go`
  - `backend/cmd/server/core.go`
  - `api/openapi.yaml`
  - `docs/review/api-guide.md`
  - `scripts/smoke-runs-dev.sh`
- **New APIs**: none.
- **Changed APIs**: no response shape change planned.
- **Dependencies**: none.

## Scope
- **In scope**:
  - Add an explicit Run lifecycle-control path in Run Kernel.
  - Add or wire adapter-backed control methods for Argo runtime operations.
  - Preserve event ledger entries for requested, succeeded, and failed operations.
  - Preserve old `/pipeline-runs` and `/workflows` compatibility.
  - Add tests for adapter delegation, fallback, and event semantics.
- **Out of scope**:
  - Rewriting Run submit/transpiler flow.
  - Removing `argo.WorkflowClient`.
  - Removing legacy deployment or workflow endpoints.
  - Adding new database tables or migrations.
  - Frontend navigation changes beyond any needed type/doc alignment.

## Success Criteria
- [ ] Run product lifecycle operations can be executed through the Run Kernel boundary without handlers knowing Argo details.
- [ ] Adapter-backed retry/stop/suspend/resume/terminate uses Run runtime refs, namespace fallback, and consistent errors when runtime refs are missing.
- [ ] Legacy pipeline-run operations remain compatible.
- [ ] RunEvent ledger behavior remains covered for requested, succeeded, and failed runtime controls.
- [ ] Backend targeted tests cover the new Run Kernel lifecycle path and Argo adapter delegation.

## Goals (SLO)
- **Latency**: No extra live runtime lookup is added before each lifecycle operation beyond the current stored Run fetch.
- **Concurrency**: Lifecycle operations remain one runtime API call per user action.
- **Quality**: Tests cover success, missing runtime reference, missing runtime adapter/client, and runtime operation failure.

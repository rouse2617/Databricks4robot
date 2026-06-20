# Design - CYB-3003

## Architecture Context
- **Constraints**: Run storage remains `pipeline_runs`; `pipeline.Usecase` still owns persistence, event append, and legacy routes.
- **Goals**: Move product lifecycle control semantics behind the Run Kernel boundary while keeping Argo replaceable through `runtimeos/adapter`.
- **Non-Goals**: Full Run schema migration, submit pipeline rewrite, or removing legacy workflow/debug endpoints.

## Affected Modules
- `backend/internal/runtimeos/run/` - extend the facade from thin delegation toward lifecycle-control orchestration.
- `backend/internal/runtimeos/adapter/` - keep the adapter-neutral runtime control contract and ensure Argo operation coverage is complete.
- `backend/internal/usecase/pipeline/` - expose a narrow runtime-control dependency or helper that Run Kernel can use without duplicating ledger code.
- `backend/internal/handlers/pipeline/` - keep product Run handlers routed through `run.Service`.
- `backend/cmd/server/` - wire the Argo runtime adapter for Run Kernel where applicable.
- `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/smoke-runs-dev.sh` - document and verify unchanged product operation semantics.

## Architecture Decisions

### Decision 1: Lift lifecycle control first, leave submit flow for a later slice
- **Approach**: Start with retry, stop, suspend, resume, and terminate because these already have a clear runtime reference on stored Runs.
- **Alternative**: Move Run submit and workflow creation to RuntimeAdapter in the same slice.
- **Rationale**: Submit currently includes manifest construction, config projection, resource guards, and runtime config owner handling. Mixing that with lifecycle control would create a broad risky diff.
- **Trade-off**: Submit remains Argo-client based until a later RuntimeAdapter submit slice.
- **Rollback**: Revert RunService lifecycle wiring and keep the existing pipeline usecase calls.

### Decision 2: Preserve pipeline usecase ledger helpers during the transition
- **Approach**: Keep RunEvent append and source Run lookup in pipeline usecase helpers, but make runtime calls adapter-backed where possible.
- **Alternative**: Duplicate event ledger code inside `runtimeos/run`.
- **Rationale**: The ledger implementation is already tested and repository-backed there. Duplicating it before storage migration would increase drift.
- **Trade-off**: `pipeline.Usecase` remains part of the Run Kernel transition boundary for this slice.
- **Rollback**: Remove the adapter injection and use the previous `wfClient` operation helpers.

### Decision 3: Use explicit fallback for compatibility
- **Approach**: If a runtime adapter is not configured, keep using the existing `argo.WorkflowClient` path so tests and local setups remain compatible.
- **Alternative**: Require adapter wiring everywhere immediately.
- **Rationale**: This repository has many unit tests constructing `pipeline.Usecase` directly; a hard adapter requirement would create noisy test setup churn.
- **Trade-off**: Two runtime call paths exist briefly, but the adapter path is the preferred production wiring.
- **Rollback**: Disable adapter injection and continue using `wfClient`.

## Data Flow

```text
POST /api/v1/runs/:id/stop
  -> pipeline Handler
  -> runtimeos/run.Service
  -> pipeline runtime control helper
  -> stored Run runtime ref (workflowName, namespace, uid)
  -> RuntimeAdapter.Stop(ref) when configured
  -> fallback argo.WorkflowClient.StopWorkflow(...)
  -> RunEvent requested/succeeded/failed appended
```

## Data Model Changes
- No database schema changes.
- No public API response shape changes.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Adapter and workflow client paths diverge | Runtime controls may behave differently in tests versus dev | Add shared tests for namespace/ref mapping and failure event behavior |
| RunService remains partially delegated | Architecture is not fully pure Run Kernel yet | Keep this slice explicit and track submit/event-service migration separately |
| Missing runtime refs are surfaced earlier | Some malformed historical Runs cannot be controlled | Return existing invalid argument semantics and preserve debug metadata for inspection |

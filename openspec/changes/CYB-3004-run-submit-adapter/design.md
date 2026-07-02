# Design - CYB-3004

## Architecture Context
- **Constraints**: `pipeline.Usecase` still builds Argo Workflow manifests and owns persistence during this phase.
- **Goals**: Make RuntimeAdapter the preferred runtime submission boundary without changing public Run APIs.
- **Non-Goals**: Rewriting the transpiler, changing storage schema, or removing Argo-specific debug APIs.

## Affected Modules
- `backend/internal/usecase/pipeline/` - replace direct submit call with adapter-preferred helper while preserving fallback.
- `backend/internal/runtimeos/adapter/` - keep `Submit` contract adequate for returning runtime references.
- `backend/internal/runtimeos/adapter/argo/` - use existing Argo adapter submit implementation and test namespace/ref behavior.
- `backend/cmd/server/` - already wires ArgoRuntimeAdapter; verify submit path uses it.
- `docs/review/api-guide.md` - clarify Run creation uses RuntimeAdapter for runtime submit.

## Architecture Decisions

### Decision 1: Adapter submit is preferred, workflow client remains fallback
- **Approach**: Add a submit helper in `pipeline.Usecase` that calls `runtimeAdapter.Submit` when configured and falls back to `wfClient.CreateWorkflow`.
- **Alternative**: Require adapter for all submissions immediately.
- **Rationale**: Many tests and local constructors still provide only `wfClient`; preserving fallback keeps the migration low-risk.
- **Trade-off**: There are two submit paths for one more phase.
- **Rollback**: Remove adapter branch and restore direct `wfClient.CreateWorkflow`.

### Decision 2: Keep post-submit detail lookup for status and owner data
- **Approach**: After adapter submit, continue using the existing workflow detail/status logic to resolve UID, status, and runtime config owner data.
- **Alternative**: Trust only the adapter submit return object.
- **Rationale**: Runtime config projection needs a reliable owner reference, and existing Argo server behavior sometimes requires a follow-up detail lookup.
- **Trade-off**: The adapter submit return is not the sole source of status yet.
- **Rollback**: Revert to the previous direct workflow-client lookup sequence.

### Decision 3: Keep API contracts stable
- **Approach**: Treat this as internal runtime-boundary work. Update docs but not OpenAPI schemas or route shapes.
- **Alternative**: Add new submit endpoint fields exposing adapter details.
- **Rationale**: Run API already exposes runtime refs through `/runs/:id/runtime`; creation response shape does not need to change.
- **Trade-off**: Adapter internals remain invisible to clients.

## Data Flow

```text
POST /api/v1/runs/template/:id
  -> pipeline handler
  -> RunService / pipeline usecase
  -> resource guard + runtime config + transpiler
  -> RuntimeAdapter.Submit(runRef, runtimeSpec)
  -> fallback wfClient.CreateWorkflow when adapter missing
  -> existing detail/status lookup
  -> deployment + pipeline_run persistence
  -> RunEvent ledger append
```

## Data Model Changes
- No database schema changes.
- No public API response shape changes.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Adapter submit and fallback error wrapping diverge | Clients may see inconsistent error messages | Preserve current `ErrWorkflowUnavailable` wrapping for adapter errors that indicate missing runtime backend |
| Runtime config owner lookup still uses Argo client | Submit is not fully adapter-neutral yet | Keep this explicitly scoped; future runtime config owner abstraction can follow |
| Existing tests assume direct CreateWorkflow calls | Test failures may be noisy | Keep fallback path and add adapter-specific tests rather than changing all old tests |

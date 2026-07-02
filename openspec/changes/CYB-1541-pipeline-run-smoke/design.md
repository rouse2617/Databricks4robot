# Design — CYB-1541 Pipeline Run Smoke Fixes

## Architecture Context
- **Constraints**: Existing API request/response shapes must remain unchanged. The Postgres `pipeline_runs.asset_ids` column is not nullable. The frontend runs as a Vite app against the local Go API.
- **Goals**: Fix two regressions with minimal surface area and preserve the first-class pipeline run model introduced by earlier CYB work.
- **Non-Goals**: Add new API endpoints, change schema constraints, or solve component output-file policy.

## Affected Modules
- `backend/internal/postgres` — ensure empty slices are serialized as an empty JSON/array value instead of `NULL`.
- `backend/internal/usecase/pipeline` — preserve empty asset lists from requests through run creation.
- `Frontend/src/components/pipeline` — make pipeline and run-dialog inputs reliably controlled and reset modal-local state.

## Architecture Decisions

### Decision 1: Normalize empty asset IDs at the persistence boundary
- **Approach**: Treat nil and empty asset ID slices as an explicit empty collection before writing `pipeline_runs`.
- **Alternative**: Relax the database constraint to allow `NULL`.
- **Rationale**: The domain distinguishes "no assets selected" as a valid no-asset run. Persisting `[]` keeps query semantics stable and avoids a migration.
- **Rollback**: Revert repository normalization if a later migration intentionally changes the column contract.

### Decision 2: Keep frontend inputs controlled by React state
- **Approach**: Ensure form inputs update their bound state directly and reset modal-local search state when the run dialog closes/opens for a pipeline.
- **Alternative**: Work around only automation by using imperative DOM clearing.
- **Rationale**: The user-visible issue is stale text accumulation; controlled state is the correct source of truth.

## Data Flow

```text
Run dialog submit
  -> pipelineApi.createPipelineRunFromTemplate({ asset_ids: [] | [id], target_id })
  -> pipeline usecase creates PipelineRun
  -> postgres repo writes asset_ids as [] or [id], never NULL
  -> Argo workflow creation / run detail refresh
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Existing nil asset lists may now read back as empty lists | Low; this matches API intent | Add/update unit coverage around empty asset IDs |
| Frontend reset may clear a user's previous search when reopening | Intended behavior for modal isolation | Keep selected assets only within an open dialog |

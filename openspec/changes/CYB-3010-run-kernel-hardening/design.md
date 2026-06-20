# Design - CYB-3010 Run Kernel Hardening

## Architecture Context

- **Constraints**: `pipeline_runs` remains the current Run storage table; legacy BatchJob and Deployment paths still exist; dev deploy requires migrations before backend image rollout.
- **Goals**: Turn Run Tree relationships and Run inputs from transient projections into durable facts while preserving existing API envelopes and historical fallback.
- **Non-Goals**: This slice does not introduce RunOutput/Artifact persistence or move all submit/start ownership into RunService.

## Affected Modules

- `backend/migrations/` - add durable Run fact tables.
- `backend/internal/models/pipeline.go` - add persistent Run input/relation models and aggregate health fields.
- `backend/internal/repository/pipeline_repository.go` - add repository contracts for Run relations and Run inputs.
- `backend/internal/postgres/pipeline_repo.go` - implement idempotent upsert/list operations.
- `backend/internal/usecase/pipeline/usecase.go` - write parent Runs, relations, and inputs during Run materialization.
- `backend/internal/runtimeos/state/state_machine.go` - prefer durable facts when aggregating and retain legacy fallback.
- `Frontend/src/api/runApi.ts` and `Frontend/src/pages/BatchJobDetailPage.tsx` - display aggregate health fields.

## Architecture Decisions

### Decision 1: Store durable Run facts in separate tables
- **Approach**: Add `run_relations` and `run_inputs` tables instead of embedding more facts in `PipelineJSON`.
- **Alternative**: Continue using `_run_config_inputs` and RunEvent payload projection.
- **Rationale**: Relations and inputs are product facts that must survive runtime TTL, config edits, and UI projection changes.
- **Trade-off**: Requires a dev migration and repository updates.
- **Rollback**: Code can fall back to old projection paths if the new tables are empty; migration is additive.

### Decision 2: Use idempotent natural uniqueness
- **Approach**: Deduplicate relations by parent, child, and type; deduplicate inputs by run, input type, node, reference, and mount target.
- **Alternative**: Allow append-only duplicate rows and deduplicate at read time.
- **Rationale**: Batch and retry flows can be retried by workers or operators, so storage must tolerate repeated materialization without inflating UI counts.
- **Trade-off**: Some input sources need stable keys derived from existing metadata.
- **Rollback**: Uniqueness constraints can remain while old projection fallback still handles missing rows.

### Decision 3: Parent Batch Run is the product root
- **Approach**: Batch creation/upsert creates a Run whose id matches the BatchJob id when possible, and child Runs persist `batch_child` relations to that parent.
- **Alternative**: Keep BatchJob as a separate root and resolve children only by `batch_job_id`.
- **Rationale**: Run Kernel requires Batch to be a parent Run so `/runs/{id}` and `/runs/{id}/children` use one product identity.
- **Trade-off**: Parent Runs may not have a runtime workflow; Run Inspector must treat them as aggregate Runs.
- **Rollback**: Existing `batch_job_id` fallback still resolves historical children.

### Decision 4: Keep projection fallback for historical data
- **Approach**: Read durable rows first. If no rows exist, derive inputs from current `PipelineJSON` and relations from `batch_job_id` / event payloads.
- **Alternative**: Backfill all historical data immediately and remove projections.
- **Rationale**: Immediate backfill would add operational risk and is unnecessary for read compatibility.
- **Trade-off**: The read path remains dual-source during the migration period.
- **Rollback**: Durable readers can be disabled without breaking historical UI.

## Data Flow

```text
Create BatchJob
  -> Upsert parent PipelineRun(type=batch/root by id)
  -> For each child Run
       -> Save PipelineRun
       -> Upsert run_relation(batch_child)
       -> Upsert run_inputs(asset/config/runtime_target/parameter)

Rerun / Resubmit / Retry full-run
  -> Save new PipelineRun
  -> Upsert run_relation(parent=source, child=new, relation_type)
  -> Upsert run_inputs for the child Run

GET /runs/:id/children
  -> list durable run_relations
  -> hydrate child runs and aggregate health
  -> if empty, use legacy batch/event projection fallback

GET /runs/:id/inputs
  -> list durable run_inputs
  -> if empty, use legacy PipelineJSON projection fallback
```

## Data Model Changes

- **Table**: `run_relations`
- **Change**: Add durable parent/child Run relation facts with relation type, optional asset id, source, and timestamps.
- **Migration**: `backend/migrations/056_run_kernel_facts.sql`

- **Table**: `run_inputs`
- **Change**: Add durable Run input facts with type, node, reference, version, mount target, content hash, source, snapshot, and timestamps.
- **Migration**: `backend/migrations/056_run_kernel_facts.sql`

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| New tables are empty for historical Runs | Old data could appear to lose relationships or inputs | Keep existing projection fallbacks |
| Batch parent Run has no runtime workflow | Inspector could show misleading runtime data | Treat missing runtime as aggregate/not-submitted metadata |
| Duplicate materialization on retries | UI counts could inflate | Use idempotent upsert constraints |
| Migration applied after code deploy | Runtime 500s on new table access | Apply migration before backend deploy per repo deploy rules |

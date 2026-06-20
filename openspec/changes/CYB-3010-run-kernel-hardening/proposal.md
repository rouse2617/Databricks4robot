# Proposal - CYB-3010 Run Kernel Hardening

## Why

Run Kernel currently looks Run-centric, but key facts are still projected from legacy metadata. Batch parent identity, child relationships, and Run inputs must become durable facts so users can reliably answer "what ran, why, with which inputs, and how is it related?".

## What Changes

### New Capabilities
- Persist Run relationships such as batch children, rerun, resubmit, and retry into a durable relation ledger.
- Persist Run inputs such as assets, configs, runtime target, and parameters at Run creation time.
- Create or upsert a parent Run for every BatchJob so batch detail and `/runs/{id}/children` have a stable product root.

### Modified Capabilities
- `/runs/{id}/children` prefers durable relations and keeps legacy fallback for historical batch data.
- `/runs/{id}/inputs` prefers durable inputs and keeps legacy projection fallback for older Runs.
- Batch aggregate summaries expose degraded health when a running batch already has failed or blocked children.
- Failed, Error, and Expired Runs receive a generic diagnostic reason even when runtime messages are empty.

## Impact
- **Affected code**:
  - `backend/migrations/`
  - `backend/internal/models/pipeline.go`
  - `backend/internal/repository/pipeline_repository.go`
  - `backend/internal/postgres/pipeline_repo.go`
  - `backend/internal/usecase/pipeline/usecase.go`
  - `backend/internal/runtimeos/state/state_machine.go`
  - `Frontend/src/api/runApi.ts`
  - `Frontend/src/pages/BatchJobDetailPage.tsx`
- **New APIs**: none.
- **Changed APIs**: existing Run child summaries may include aggregate health fields; existing Run inputs/children keep their envelope shape.
- **Dependencies**: Postgres migration applied before backend deploy.

## Scope
- **In scope**:
  - Add `run_relations` table and repository methods.
  - Add `run_inputs` table and repository methods.
  - Persist `batch_child`, `rerun_of`, `resubmit_of`, and retry-created child relations.
  - Materialize asset, config, runtime target, and parameter Run inputs for new Runs.
  - Ensure BatchJob creation/upsert creates a parent Run.
  - Preserve legacy projection fallback for existing data.
  - Add backend and frontend tests for durable relations, inputs, and aggregate health.
- **Out of scope**:
  - Persistent RunOutput/Artifact tables.
  - Full RunService ownership of submit/start.
  - Removing legacy Deployment or Workflow routes.
  - Databricks or multi-runtime expansion.
  - Full frontend renaming from WorkflowDetailPage to RunInspectorPage.

## Success Criteria
- [x] Every newly created BatchJob has a parent Run root.
- [x] Newly created child Runs write durable `batch_child` relations.
- [x] Rerun/resubmit/retry-created child Runs write durable relations instead of relying only on event projection.
- [x] Newly created Runs persist asset, config, runtime target, and parameter inputs.
- [x] `/runs/{batchJobId}/children` works for new parent Runs and historical batches without a parent Run.
- [x] `/runs/{id}/inputs` returns durable inputs for new Runs and legacy projection for old Runs.
- [x] Batch summaries can show running-with-failures or blocked health without hiding child failures.
- [x] Backend tests and local frontend validation cover the hardening behavior.

## Goals (SLO)
- **Latency**: `/runs/{id}/children` and `/runs/{id}/inputs` stay bounded by indexed lookups and do not require scanning all runs.
- **Concurrency**: Batch creation persists relations and inputs idempotently so repeated materialization does not create duplicates.
- **Quality**: Schema changes are additive, migration is idempotent, and legacy fallback keeps old Runs readable.

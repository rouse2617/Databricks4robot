# Tasks

## Context Files

- `Frontend/src/pages/ComponentManager.tsx` — component create/edit resource fields.
- `Frontend/src/components/pipeline/NodeConfigPanel.tsx` — pipeline node override resource fields.
- `Frontend/src/lib/pipelineValidation.ts` — save/run validation helper.
- `Frontend/src/pages/ComponentManager.test.tsx` — component form tests.
- `Frontend/src/pages/PipelinePage.test.tsx` — pipeline designer tests.
- `backend/internal/models/pipeline_component.go` — component resource model.
- `backend/internal/handlers/pipeline_component` / `backend/internal/usecase/pipeline_component` — component CRUD validation path.
- `backend/internal/transpiler` / pipeline run creation path — final manifest/resource validation path.

## OpenSpec

- [x] Create CYB-1685 proposal, tasks, context files, and spec delta.

## Implementation

- [x] [Frontend] Add shared resource quantity validation helpers for CPU, memory, disk, and GPU.
- [x] [Frontend] Apply validation to component create/edit resource fields.
- [x] [Frontend] Apply validation to pipeline node override resource fields and save/run validation.
- [x] [backend] Add server-side validation for component resources.
- [x] [backend] Add server-side validation before pipeline templates/runs are accepted or transpiled.
- [x] [tests] Add frontend tests proving `memory=1` / `disk=1` fail and `512Mi` / `1Gi` pass.
- [x] [tests] Add backend tests proving unsafe API payloads are rejected.

## API Contract Sync

- [x] No new HTTP endpoint.
- [x] Existing component/template/run endpoints keep their response shape and return existing validation/error envelopes for invalid resource values.
- [x] Updated OpenAPI, API guide, and API guide smoke for resource quantity validation semantics.

## Verification

- [x] Frontend targeted tests for component manager and pipeline designer validation.
- [x] Backend targeted tests for component validation and pipeline submission/transpiler validation.
- [x] `cd Frontend && npm run build`.
- [x] Backend targeted `go test` for touched packages.
- [x] Browser verification: create component with `memory=1` fails; `memory=512Mi` succeeds.

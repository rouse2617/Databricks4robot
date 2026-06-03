# Tasks — CYB-1619 Complex Pipeline Authoring

## Context Files

- `Frontend/src/pages/ComponentManager.tsx`
- `Frontend/src/pages/PipelinePage.tsx`
- `Frontend/src/components/pipeline/*`
- `Frontend/src/lib/pipelineContract.ts`
- `Frontend/src/lib/pipelineValidation.ts`
- `backend/internal/handlers/pipeline_component/*`
- `backend/internal/usecase/pipeline_component/*`
- `backend/internal/transpiler/*`
- `api/openapi.yaml`
- `docs/review/api-guide.md`

## Checkpoint

- [x] Create Linear issue CYB-1619.
- [x] Create branch/worktree from latest `origin/dev`.
- [x] Write OpenSpec proposal, design, tasks, and spec delta.
- [x] Runtime edit checkpoint acknowledged by user request: “先编码”.

## Implementation

- [x] [Frontend] Add component input/output port editing UI.
- [x] [Frontend] Add output contract helper text for `/tmp/outputs/<port>` near output/script fields.
- [x] [Frontend] Ensure component port edits persist through existing component API or document required API sync.
- [x] [Frontend] Improve pipeline designer feedback for duplicate target input connections.
- [x] [Frontend] Make join-style fan-in composition understandable through port labels/selection/editing.
- [x] [Frontend] Add standard complex pipeline examples/templates:
  - [x] sequential chain
  - [x] fan-out/fan-in join
  - [x] 10s+ observation workflow
- [x] [Backend/API] If current component/template APIs cannot persist the needed port fields, update API contract and backend in the same PR.
- [x] [Tests] Add focused unit tests for port editing, validation, and sample generation/loading.

## Verification

- [x] `cd Frontend && npm run test -- --run <related tests>`
- [x] `cd Frontend && npm run lint`
- [x] `cd Frontend && npm run build`
- [x] `cd backend && go test ./internal/transpiler ./internal/usecase/pipeline_component ./internal/handlers/pipeline_component` if backend touched
- [ ] Browser smoke: create/edit component ports.
- [ ] Browser smoke: build and save valid fan-out/fan-in pipeline.
- [ ] Browser smoke: load examples and run at least one pipeline.

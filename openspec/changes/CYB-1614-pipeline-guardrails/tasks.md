# Tasks — CYB-1614 Pipeline Guardrails

## Context Files
- `backend/internal/transpiler/transpiler.go`
- `backend/internal/transpiler/transpiler_test.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/pipeline/usecase_*_test.go`
- `Frontend/src/pages/ComponentManager.tsx`
- `Frontend/src/pages/PipelinePage.tsx`
- `Frontend/src/pages/WorkflowDetailPage.tsx`
- `Frontend/src/components/pipeline/*`

## Checkpoint
- [x] Write OpenSpec proposal, tasks, and spec delta.
- [x] Stop for confirmation before runtime edits.

## Implementation
- [x] [backend] Add validation for duplicate edge target input parameters before workflow creation.
- [x] [backend] Return a clear validation error instead of letting Argo reject duplicate fan-in inputs.
- [x] [backend] Normalize shell command/args handling for `["sh", "-c"]` and `["sh"], ["-c", script]` component shapes.
- [x] [backend] Preserve output-file behavior for consumed outputs while avoiding unexpected failure for unconsumed/default outputs.
- [x] [Frontend] Add duplicate fan-in target feedback in the pipeline designer before save/run.
- [x] [Frontend] Normalize component form values so `sh -c` stores one script argument, not `sh`, `-c`, and script as separate args.
- [x] [Frontend] Add output-port helper text/warning for `/tmp/outputs/<port>` when outputs are declared.
- [x] [Frontend] Improve cost-summary syncing/partial-data empty state in list/detail views.
- [x] [Frontend] Normalize workflow log rendering so Argo-prefixed log output displays as readable lines.

## Verification
- [x] `cd backend && go test ./internal/transpiler ./internal/usecase/pipeline`
- [x] `cd backend && go test ./...`
- [x] `cd Frontend && npm run test -- --run src/lib/pipelineValidation.test.ts src/pages/workflowLogView.test.ts src/pages/PipelinePage.test.tsx`
- [x] `cd Frontend && npm run lint`
- [x] `cd Frontend && npm run build`
- [ ] Local browser smoke: create/run sequential and fan-in pipelines; verify invalid fan-in is blocked and valid distinct-input fan-in succeeds.
- [ ] Local browser smoke: open workflow detail logs/cost view and verify readable logs plus cost syncing state.

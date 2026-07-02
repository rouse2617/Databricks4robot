# Tasks - CYB-3008

## Context files

- `Frontend/src/pages/useWorkflowDetail.ts`
- `Frontend/src/pages/useWorkflowDetail.test.tsx`
- `Frontend/src/pages/WorkflowDetailPage.tsx`
- `Frontend/src/pages/WorkflowDetailPage.test.tsx`
- `Frontend/src/api/runApi.ts`

## OpenSpec

- [x] [openspec] Create proposal, design, tasks, decisions, context files, and runtime-os spec delta.
- [x] [openspec] Continue without stopping for another checkpoint per user instruction.

## Implementation

- [x] [Frontend] Load Run inputs, outputs, and runtime metadata in `useWorkflowDetail`.
- [x] [Frontend] Keep metadata failures local and non-fatal.
- [x] [Frontend] Render Run Inspector metadata section for DataBrew Runs.
- [x] [Frontend] Add focused hook/page tests.

## Verification

- [x] [Frontend] `cd Frontend && npx biome check src/pages/useWorkflowDetail.ts src/pages/useWorkflowDetail.test.tsx src/pages/WorkflowDetailPage.tsx src/pages/WorkflowDetailPage.test.tsx`.
- [x] [Frontend] `cd Frontend && npm run test -- src/pages/useWorkflowDetail.test.tsx src/pages/WorkflowDetailPage.test.tsx --run`.
- [x] [Frontend] Include this page in the final full frontend build/test pass: `cd Frontend && npm run test -- --run` and `cd Frontend && npm run build`.
- [ ] [repo] Non-Terraform pre-commit checks before final commit when the full runtime goal is ready.

## Deploy Verification

- [ ] Deploy frontend dev when the full runtime batch is ready for deploy gate.
- [ ] Use Chrome DevTools MCP to verify Run Inspector renders metadata on a deployed Run detail page.

## Push Strategy

- [ ] Do not open or push a PR for this slice alone.
- [ ] Keep work on local `dev`; push once only after the full Run as Kernel goal is complete and verified.

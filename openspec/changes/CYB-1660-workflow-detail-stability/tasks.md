# Tasks — CYB-1660

## Implementation

- [x] Add frontend pipeline-run lookup by `runId`.
- [x] Resolve workflow detail candidates from `runId`, workflow name, normalized pipeline name, and route name.
- [x] Preserve `runId` in execution-detail navigation from pipeline run results and asset lineage.
- [x] Add failed-node and status-sync warnings on the execution detail page.
- [x] Show failed asset-node log actions even when a direct `logRef` is unavailable.
- [x] Make unavailable row operations visibly disabled instead of opening an empty menu.

## Verification

- [x] Add or update focused frontend tests for runId lookup, status drift warning, and failed-node log entry.
- [x] Run `cd Frontend && npm run test -- --run src/pages/useWorkflowDetail.test.tsx src/pages/WorkflowDetailPage.test.tsx src/pages/WorkflowExecutionList.test.tsx`.
- [x] Run targeted `npx biome check` on changed frontend files.
- [x] Run `cd Frontend && npm run build`.
- [x] Run browser fallback spot check on the current local frontend.
- [ ] Open PR to `dev` after final browser spot check.

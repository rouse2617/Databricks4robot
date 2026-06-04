# CYB-1643 Tasks

## Context files

- `Frontend/src/pages/WorkflowDetailPage.tsx` — workflow detail cost and asset-node panel rendering
- `Frontend/src/pages/WorkflowDetailPage.test.tsx` — workflow detail UI assertions
- `Frontend/src/pages/useWorkflowDetail.ts` — cost summary and asset-node state loading

## 1. Implementation

- [x] [Frontend] Classify cost empty states by workflow/node status before showing warning copy.
- [x] [Frontend] Replace generic unavailable copy for Pending/Running nodes with "waiting for resource snapshot" copy.
- [x] [Frontend] Keep completed missing-cost rows quiet and reserve config wording for true `not_available` summaries.
- [x] [Frontend] Add or update workflow detail tests for pending/running and completed missing-cost states.

## 2. Verification

- [x] Run targeted frontend tests for workflow detail.
- [x] Run frontend build or equivalent Tier L check before PR.
- [x] Verify with Chrome DevTools MCP on a workflow detail page after frontend dev deployment.
- [x] Open PR to `dev` with Linear and OpenSpec links.

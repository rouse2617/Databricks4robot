# Tasks — CYB-1703

## Implementation
- [x] Build a node display-name resolver for workflow detail node table rows.
- [x] Prefer component/business labels from live workflow/pipeline snapshot data over Argo technical step ids.
- [x] Preserve technical ids as secondary diagnostic metadata or tooltip.
- [x] Keep node action handlers using the original row identity, not the display label.
- [x] Keep workflow detail in a DataBrew syncing state while run context and asset-node details are still loading.

## Verification
- [x] Frontend targeted test for workflow detail node table names.
- [x] `npm run test -- --run src/pages/WorkflowDetailPage.test.tsx`
- [x] `npm run lint`
- [x] `npm run build`
- [x] Deploy frontend dev Worker.
- [x] Chrome DevTools MCP verification on `/pipeline/executions/test-77705a?runId=4ec0af22-d1eb-4880-9a62-088a8a5951b0`.

## Documentation / Tracking
- [ ] Record PR link and verification evidence on Linear CYB-1703.
- [ ] PR body references CYB-1703 and this OpenSpec change.

## Deploy Record
- **Frontend Worker**: `cyber-databrew-dev`
- **Worker Version ID**: `c0395d42-ce7f-412e-b37f-30e1de45cfdb`
- **Dev URL**: `https://cyber-databrew-dev.cyberorigin.ai/`

# Tasks — CYB-1522

## Context files
- `Frontend/src/pages/PipelinePage.tsx` — designer layout, toolbar, import/deploy modals, pipeline/components tab structure
- `Frontend/src/components/pipeline/DeployPanel.tsx` — saved pipeline list, run history, full management view
- `Frontend/src/pages/ComponentManager.tsx` — component CRUD management tab
- `Frontend/src/components/pipeline/PipelineEmptyState.tsx` — empty canvas presentation
- `Frontend/src/styles/pipeline.css` — desktop layout proportions and component styling
- `Frontend/src/pages/PipelinePage.test.tsx` and `Frontend/src/components/pipeline/DeployPanel.test.tsx` — regression coverage

## Implementation
- [x] [Frontend] Rebalance desktop designer columns so the palette and saved-pipeline panel are readable while the canvas remains flexible.
- [x] [Frontend] Improve the top toolbar hierarchy: primary save/deploy, secondary import/export, clearly separated destructive clear.
- [x] [Frontend] Add a deploy-disabled explanation for empty/unsaved/invalid deploy conditions.
- [x] [Frontend] Strengthen empty canvas guidance and hide/minimize nonessential canvas controls when no nodes exist.
- [x] [Frontend] Refactor the saved sidebar so saved pipelines use readable row structure and node configuration does not permanently squeeze the list when no node is selected.
- [x] [Frontend] Improve the expanded saved/run-history modal organization and dense list readability.
- [x] [Frontend] Update focused tests for deploy-disabled messaging, saved list rendering, and modal organization.
- [x] [Frontend] Move saved pipeline CRUD out of the designer sidebar into a dedicated pipeline management tab.
- [x] [Frontend] Keep the designer right panel focused on current-node configuration and an empty node-selection state.
- [x] [Frontend] Align save/deploy follow-up actions with the new pipeline management tab.
- [x] [Frontend] Update tests for the dedicated pipeline management tab and removed saved sidebar.

## Verification
- [x] Run `cd Frontend && npm run lint`.
- [x] Run focused frontend tests for pipeline page and deploy panel.
- [x] Run `cd Frontend && npm run build`.
- [x] Run local frontend preview and inspect `/pipeline` with Chrome DevTools MCP on desktop.
- [x] Verify console has no new errors and targeted layout screenshots show the improved desktop states.

## Deploy verification
- [ ] Deploy frontend dev.
- [ ] Open dev `/pipeline` with Chrome DevTools MCP on desktop.
- [ ] Verify design tab empty-canvas state, toolbar disabled deploy explanation, pipeline management tab readability, and console.

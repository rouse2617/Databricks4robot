# Proposal — CYB-1617

## Why
Assets are already the business object users start from, while Pipeline is where processing happens. The current product has partial hooks, but the asset list still sends users to a disabled "algorithm" action and the run/detail surfaces do not consistently explain which assets are being processed.

## What Changes

### New Capabilities
- Asset discovery supports launching a Pipeline run from selected assets, including multi-select batches.
- Pipeline run dialogs preserve the asset context that came from asset pages and make the selected asset set visible before submission.
- Pipeline execution detail links asset-bound runs back to the input assets and keeps future product/lineage affordances in business language.

### Modified Capabilities
- The asset "trigger algorithm" bulk action becomes a Pipeline processing action instead of a disabled placeholder.
- Asset detail's "create pipeline" action becomes "run pipeline for this asset" so users do not confuse template authoring with execution.
- Pipeline empty/cost/event placeholders use product-facing copy and avoid backend-oriented "待接入" language.

## Impact
- **Affected code**: `Frontend/src/pages/AssetsPage.tsx`, `Frontend/src/components/assets/BulkActionBar.tsx`, `Frontend/src/pages/AssetDetailPage.tsx`, `Frontend/src/pages/PipelinePage.tsx`, `Frontend/src/pages/WorkflowDetailPage.tsx`, `Frontend/src/components/pipeline`
- **New APIs**: none planned; use existing asset search, pipeline template, pipeline run, run events, asset-node, and cost summary APIs
- **Dependencies**: no new third-party dependency planned

## Scope
- **In scope**: frontend asset-to-pipeline entry points, selected asset context propagation, run confirmation UX, execution detail asset links, and targeted tests for these flows.
- **Out of scope**: new database tables, migration files, new run/product/lineage APIs, OpenLineage integration, actual billing reconciliation, and event-triggered automatic pipeline runs.

## Success Criteria
- [ ] A user can select one or more assets on the asset discovery page and open Pipeline with those assets preselected for a run.
- [ ] A user can open an asset detail page and run an existing Pipeline for that single asset without losing context.
- [ ] Pipeline run confirmation clearly shows selected asset count and IDs before submission.
- [ ] Execution detail for an asset-bound run shows the bound asset IDs with links back to asset detail.
- [ ] No-asset runs remain available and are clearly labeled as debug/no-input runs.

## Goals (SLO)
- **Latency**: opening Pipeline from selected assets should render the run surface within 1 second after frontend navigation, excluding API latency.
- **Concurrency**: the UI remains usable when a selected batch contains 100 asset IDs.
- **Quality**: frontend tests cover asset list launch, asset detail launch, preselected run modal state, and execution detail asset links.

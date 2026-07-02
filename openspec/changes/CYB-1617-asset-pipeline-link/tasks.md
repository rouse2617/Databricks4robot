# Tasks — CYB-1617

## Context files
- `Frontend/src/pages/AssetsPage.tsx` — asset discovery selection and bulk action wiring.
- `Frontend/src/components/assets/BulkActionBar.tsx` — selected-assets toolbar actions.
- `Frontend/src/pages/AssetDetailPage.tsx` — single-asset action entry point.
- `Frontend/src/pages/PipelinePage.tsx` — `asset_ids` query parsing, run modal, deploy submission.
- `Frontend/src/components/pipeline/AssetPicker.tsx` — reusable asset selector in the run modal.
- `Frontend/src/components/pipeline/DeployPanel.tsx` — saved-template run modal.
- `Frontend/src/pages/WorkflowDetailPage.tsx` — execution detail summary, asset-node, event, and cost sections.
- `Frontend/src/api/pipelineApi.ts` — existing typed Pipeline run API client.
- `openspec/changes/CYB-1532-asset-driven-pipeline-mvp/*` — prior asset-driven run scope.
- `openspec/changes/CYB-1565-pipeline-observability/*` — prior run observability scope.

## Discovery
- [x] Confirm branch is based on latest `origin/dev`.
- [x] Identify existing single-asset entry in asset detail.
- [x] Identify disabled selected-assets algorithm action in asset discovery.
- [x] Confirm existing Pipeline run API accepts `asset_ids`.
- [x] Confirm existing run detail already has asset-node/event/cost hooks.

## OpenSpec Checkpoint
- [x] Write `proposal.md`.
- [x] Write `design.md`.
- [x] Write `specs/asset-management/spec.md`.
- [x] Write `specs/pipeline/spec.md`.
- [x] Write `context-files.md`.
- [x] Stop for user confirmation before runtime code edits.

## Implementation
- [x] [Frontend] Enable the asset discovery selected-assets action as "运行 Pipeline" and navigate to Pipeline with selected asset IDs.
- [x] [Frontend] Update asset detail action copy and navigation so it reads as running a Pipeline for the current asset.
- [x] [Frontend] Preserve incoming asset IDs in Pipeline until the user clears or changes the run selection.
- [x] [Frontend] Add a concise selected-asset summary in the run modal with visible IDs and clear/no-asset affordance.
- [x] [Frontend] Ensure saved-template run modal and canvas run modal both use consistent asset/no-asset labels.
- [x] [Frontend] Show linked input asset chips in execution detail when run/workflow metadata includes asset bindings.
- [x] [Frontend] Replace backend-oriented unavailable text with product-facing empty states for legacy or external workflows.
- [x] [Frontend] Add or update tests for asset list launch, asset detail launch, preselected run modal state, no-asset clearing, and execution detail asset links.

## API Contract Sync
- [x] No HTTP API changes planned; if implementation changes API behavior, update `api/openapi.yaml`, `docs/review/api-guide.md`, SDK, smoke scripts, and frontend API types in the same PR.

## Verification
- [x] `cd Frontend && npm run lint`
- [x] `cd Frontend && npm run test -- --run src/components/assets/BulkActionBar.test.tsx src/components/pipeline/DeployPanel.test.tsx src/pages/PipelinePage.test.tsx src/pages/WorkflowDetailPage.test.tsx`
- [x] `cd Frontend && npm run build`
- [x] Local browser/MCP: open `/pipeline?asset_ids=asset-a,asset-b`, add a component, open deploy modal, verify selected asset IDs are prefilled.
- [x] Local browser/MCP: clear selected assets in the deploy modal and verify no-asset run copy/button.
- [x] Local/unit verification: execution detail renders linked input asset chips and product-facing event unavailable copy.

## PR
- [ ] Fill PR template with Linear ID `CYB-1617` and OpenSpec change ID `CYB-1617-asset-pipeline-link`.
- [ ] Mention that schema/API additions are intentionally deferred because existing run APIs already accept asset IDs.

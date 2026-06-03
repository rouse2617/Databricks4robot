# Design — CYB-1617

## Architecture Context
- **Constraints**: Use latest `origin/dev`; do not add migrations; reuse existing Pipeline run APIs and asset search APIs; keep Argo execution unchanged.
- **Goals**: Make assets the natural starting point for Pipeline processing, keep run submission explicit, and preserve business context across asset, pipeline, and execution detail pages.
- **Non-Goals**: Do not create a new scheduler, do not add new persistence, do not implement automatic event-triggered runs, and do not implement product registration or exact billing.

## Affected Modules
- `Frontend/src/pages/AssetsPage.tsx` — route selected assets into Pipeline processing instead of the disabled algorithm placeholder.
- `Frontend/src/components/assets/BulkActionBar.tsx` — rename and enable the selected-assets processing action.
- `Frontend/src/pages/AssetDetailPage.tsx` — make the single-asset Pipeline action execution-oriented.
- `Frontend/src/pages/PipelinePage.tsx` — preserve `asset_ids` query context and surface it in the run dialog.
- `Frontend/src/pages/WorkflowDetailPage.tsx` — display asset bindings as linked business objects.
- `Frontend/src/components/pipeline` — reuse or lightly extend existing asset picker/empty states for this flow.

## Architecture Decisions

### Decision 1: Use URL asset context instead of new backend state
- **Approach**: Navigate to `/pipeline?asset_ids=...` from asset list/detail, then let the existing Pipeline page parse and submit those IDs through `deployTemplate`.
- **Alternative**: Create a server-side draft run or selection session.
- **Rationale**: The current frontend already parses `asset_ids`, and the backend run API already accepts asset IDs. URL context is enough for the MVP and is easy to verify.
- **Trade-off**: Very large selections make URLs long, so the UI should keep the MVP batch size conservative and guide users toward explicit selection.
- **Rollback**: Remove the new asset actions and return to the existing asset picker-only Pipeline flow.

### Decision 2: Keep submission explicit
- **Approach**: Asset pages navigate to Pipeline with assets preselected; users still choose a saved/current Pipeline and confirm the run.
- **Alternative**: Clicking "run pipeline" immediately submits the most recent/default Pipeline.
- **Rationale**: Immediate execution risks running the wrong template or target. The explicit confirm step keeps asset, template, target, and no-asset semantics visible.
- **Rollback**: Keep only the Pipeline-side asset picker and remove asset-page launch buttons.

### Decision 3: Treat lineage/product registration as displayed affordance, not new runtime scope
- **Approach**: Execution detail shows input asset links and user-friendly empty states for products/events when unavailable; product registration remains existing/future backend work.
- **Alternative**: Add new output product APIs in this change.
- **Rationale**: CYB-1532 and CYB-1565 already cover run/event/asset-node foundations. CYB-1617 should close navigation and product wording gaps first.
- **Rollback**: Revert UI copy and linked asset chips without affecting run execution.

## Data Flow

```text
Asset list/detail
  -> /pipeline?asset_ids=a,b,c
  -> Pipeline page parses asset_ids
  -> user selects template/target and confirms
  -> existing POST /pipeline-runs/template/{id}
  -> execution detail shows run labels / run data / asset-node data
  -> linked asset chips navigate back to asset detail / lineage
```

## Data Model Changes
- None.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| URL grows too long for very large asset selections | Navigation may fail or become hard to share | Keep bulk launch focused on explicit selected rows and preserve asset picker for larger/manual selection |
| Existing run detail may only have workflow labels for legacy runs | Some old/external runs still show no asset binding | Show product-facing "无资产" or "历史/外部工作流" copy instead of backend placeholders |
| Users may expect asset launch to start immediately | Extra confirm step may feel slower | Use clear button labels and preselected asset summary so the confirmation is obviously useful |

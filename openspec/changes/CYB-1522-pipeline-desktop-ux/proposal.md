# Proposal — CYB-1522

## Why
The desktop pipeline area mixes three separate jobs in one editing surface: designing the current graph, managing reusable components, and managing saved pipelines. This makes the canvas feel cramped and makes saved pipeline CRUD feel like a side panel instead of a first-class resource workflow.

## What Changes

### New Capabilities
- The pipeline designer explains why deploy is unavailable when the current canvas cannot be deployed.
- The empty canvas gives stronger first-step guidance and reduces nonessential visual noise.
- Saved pipelines are managed in a dedicated tab where users can open/edit, run, delete, and review history without competing with the canvas.
- Components remain managed in their own tab while the design palette stays focused on drag-to-use behavior.

### Modified Capabilities
- The desktop designer layout gives the component palette, canvas, and current node configuration clearly separated responsibilities.
- Saved pipeline management is easier to scan and less error-prone as a standalone management surface.
- Primary, secondary, and destructive toolbar actions have clearer visual hierarchy.
- The saved/run-history management view presents dense data in a more focused structure.

## Impact
- **Affected code**: `Frontend/src/pages/PipelinePage.tsx`, `Frontend/src/components/pipeline/DeployPanel.tsx`, `Frontend/src/components/pipeline/PipelineEmptyState.tsx`, `Frontend/src/pages/ComponentManager.tsx`, `Frontend/src/styles/pipeline.css`, related frontend tests
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**: desktop pipeline designer layout, empty canvas, dedicated saved-pipeline management tab, component management tab alignment, deploy disabled explanation, save feedback, saved/run-history readability
- **Out of scope**: mobile-specific redesign, backend API changes, workflow execution semantics, persisted data model changes

## Success Criteria
- [ ] Desktop `/pipeline` keeps the design canvas focused on component dragging, graph editing, and current node configuration.
- [ ] Saved pipeline CRUD and run-history management live in a dedicated tab rather than in the designer side panel.
- [ ] Component CRUD remains in a dedicated tab, while the design palette only exposes drag-to-use components.
- [ ] Empty canvas prominently guides users to add components and does not show the mini map until nodes exist.
- [ ] Disabled deploy action explains the unmet condition.
- [ ] Saving a pipeline gives clear user feedback and points users to the pipeline management tab.
- [ ] Saved pipeline rows display long names, node count, time metadata, and actions without visual clutter.
- [ ] The saved/run-history view is organized so saved pipelines and run history are not visually mixed into one dense wall.

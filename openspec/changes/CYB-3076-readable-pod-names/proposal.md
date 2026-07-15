# Proposal — CYB-3076

## Why

Pipeline pod names look like `grace-sync-…-step-node-09800b58-0282-4f4b-…-2967435060` — the node UUID is opaque, so you can't tell which business step (transcode, hand-tracking, …) a pod is when looking at raw pods (kubectl / GKE / logs). Argo embeds the **template name** in the pod name, and the template name is currently `step-<nodeID>`.

## What Changes

### Modified Capabilities
- **runtime-os**: Argo template names (and therefore pod names) for pipeline steps include the step's human component name, so a pod is identifiable by step from its name — e.g. `…-step-head-track-pycuvslam-8242f006-<hash>`.

## Impact
- **Affected code**: `backend/internal/transpiler` (templateName + callers + validation + the output-param wiring at :620), `backend/internal/usecase/backfill` (node-order matcher), `backend/internal/handlers/workflow` (runtime-info lookup), `Frontend/src/lib/workflowNodeDisplay.ts` (label variant), a shared slug helper.
- **New APIs**: none. **Migration**: none (old runs keep their stored format; matching is dual-format).
- **Dependencies**: none new (reuse `sanitizePodNamePart`).

## Scope
- **In scope**: readable template/pod names via a shared `stepKey(component, nodeID)`; dual-format (new + legacy) matching in the node-summary sort, workflow runtime-info lookup, and frontend label lookup so both historical and new runs work.
- **Out of scope**: renaming pipeline node IDs; changing DAG execution semantics; any data migration.

## Success Criteria
- [ ] A new run's step pod name contains the component name (e.g. `step-head-track-pycuvslam-<uuid8>`), identifiable at a glance.
- [ ] node-summary ordering, workflow runtime info, and frontend step labels work for BOTH new-format and legacy (`step-node-<uuid>`) runs.
- [ ] Two nodes with the same component name still transpile (uuid8 uniqueness; collision fallback), and a genuinely-colliding pipeline is not falsely rejected.
- [ ] Pod name stays within K8s limits (slug truncated).

## Goals (SLO)
- **Readability**: step identifiable from the raw pod name without cross-referencing the pipeline definition.
- **No regression**: historical runs' summary/labels/runtime-info unchanged (dual-format).

# Proposal: Pipeline Algorithm Iteration UX

**Change ID:** CYB-1618-pipeline-version-history
**Issue:** [CYB-1618](https://linear.app/cyberorigin/issue/CYB-1618)
**Type:** Feature
**Status:** Draft

## Problem

Algorithm engineers iterating on pipeline configurations have no way to:

1. See what versions exist and how they evolved
2. Filter executions by version to compare performance
3. Know what parameters/costs/metrics each run produced
4. Compare multiple runs side-by-side
5. Roll back to a known-good version when something goes wrong

## Solution — Four Capabilities

### P0: Version History + Execution Version Filter
Pipeline cards show clickable version badge → drawer with version timeline. Execution list gains version filter dropdown.

### P0: Run Parameter & Metric Recording
Auto-capture per-run: parameters, cost, CPU/memory duration, status, template version. Store in `PipelineDeployment` (already has the skeleton). Expose via workflow detail API.

### P1: Multi-Run Comparison View
Select 2-3 runs in execution list → side-by-side comparison panel showing: version, params, metrics, cost, duration, status.

### P1: One-Click Rollback
From version history drawer or execution detail → "Rollback to this version" button → sets active template version → next run uses rolled-back config.

## Scope

- Backend: extend `PipelineDeployment` model + add version history endpoint + add rollback endpoint
- Frontend: version history drawer, version filter, run comparison panel, rollback button
- No DB migration needed for initial phase (use existing columns)

## Out of scope

- A/B / Canary deployment (separate issue)
- Version diff visualization (separate issue)
- Quality gates (separate issue)
- Description/changelog field (follow-up PR)

# Proposal — CYB-1272 Asset Env Passthrough

## Why
Pipeline containers can be launched with selected asset IDs, but they still need a reliable asset context contract to know which DataBrew assets and storage paths to process.

## What Changes

### Modified Capabilities
- Asset-based pipeline runs inject selected asset context into every Argo node container through stable environment variables.
- The injected context includes aggregate asset IDs/count, per-asset identity, storage, timing, locator, revision, and metadata summary where available.
- Dry-run manifests expose the same environment contract as real submitted runs so operators and tests can validate the runtime context before submission.
- Pipeline run tests assert the rendered Argo manifest contains the expected asset environment variables.

## Impact
- **Affected code**: `backend/internal/usecase/pipeline`, `backend/internal/transpiler`, `backend/internal/repository`, `backend/internal/postgres`
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**: Complete the existing asset env passthrough behavior for selected pipeline assets, preserve current run API request/response shapes, and add backend regression coverage around manifest env injection.
- **In scope**: Keep compatibility variables already used by CyberPipe-style tasks, including `ASSET_IDS`, `ASSET_COUNT`, `VIDEO_ID`, and `REQUEST_ID`.
- **Out of scope**: New pipeline endpoints, frontend redesign, database schema changes, output asset registration, retry/recovery controls, and large-batch query optimization.
- **Out of scope**: Defining customer-specific or pipeline-specific asset health gates.

## Success Criteria
- [ ] A pipeline run with selected assets renders each node container with aggregate env vars `ASSET_IDS` and `ASSET_COUNT`.
- [ ] The first selected asset remains available through compatibility env `VIDEO_ID`.
- [ ] Each selected asset has indexed env vars for ID, storage URI, MCAP file ID, time range, segment locator, asset type, logical asset ID, revision, and metadata JSON when available.
- [ ] Dry-run and real-run manifest generation use the same asset env assembly path.
- [ ] Missing assets are still rejected before workflow creation.
- [ ] No-asset runs preserve current behavior and do not inject per-asset env vars.

## Goals (SLO)
- **Latency**: Keep the first implementation acceptable for interactive single/small-batch runs; large-batch `BatchGet` optimization is deferred.
- **Concurrency**: No new concurrent worker behavior.
- **Quality**: Backend tests cover selected-asset, missing-asset, and no-asset manifest behavior.

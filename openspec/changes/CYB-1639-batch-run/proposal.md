# CYB-1639 — Batch pipeline run (fan-out)

## Problem

Selecting N assets and deploying one template currently creates **one** Argo workflow with all asset IDs in environment variables. CyberPipe batch-run expects **N workflows** (one asset each) for isolation, retries, and per-asset execution records.

## Solution

- `POST /api/v1/pipeline-runs/template/:id/batch` with `{ asset_ids, target_id, version }`
- Backend loops `CreateRunByTemplateID` with a single asset per call
- All runs share `batch_run_id` (migration 052)
- Frontend uses batch endpoint when `asset_ids.length > 1`

## Non-goals

- Parallel submission throttling (future)
- Batch cancel/retry API (future)
- Replacing single-workflow multi-asset mode for N=1 or explicit opt-in later

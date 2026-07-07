# Proposal — CYB-3118

## Why
GKE cost allocation propagates pod labels into the resource-level BigQuery billing export (`k8s-label/cyber-databrew/*`). We want real GCP cost attributable to a **single run / single video**, joinable to `video_durations` (keyed by asset_id). The pod name leads with the run id (CYB-3076) but billing carries labels, not pod names — so labels are required.

## What Changes
### Modified Capabilities
- **runtime-os**: step-pod cost labels become `owner` + `run-id` + (single-asset) `asset-id`. The previous `batch-job-id` and `template-id` labels are removed — both are derivable from the run id via the app DB, and run-id + asset-id are the minimal keys needed for the full picture.

## Impact
- **Code**: `buildCostTrackingLabels` (+ Deploy call site) in `backend/internal/usecase/pipeline`.
- **APIs/Migration**: none. Future runs only (no backfill).
- **Trade-off**: batch/template-level BQ rollups now require a run-id → Postgres join instead of a direct label GROUP BY (accepted).

## Success Criteria
- [ ] A new single-asset run's pods carry `cyber-databrew/run-id` + `cyber-databrew/asset-id` (+ owner).
- [ ] No-asset / multi-asset runs omit `asset-id`; run-id always present.
- [ ] `batch-job-id` / `template-id` labels no longer emitted.

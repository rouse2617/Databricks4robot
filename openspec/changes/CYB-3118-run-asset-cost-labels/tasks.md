# Tasks — CYB-3118 (Tier S)
- [x] [backend] `buildCostTrackingLabels(owner, runID, assetID)` → emits `owner` / `run-id` / `asset-id` (skip empty); dropped `batch-job-id` + `template-id`.
- [x] [backend] Deploy call site passes `costOwner`, `depID`, and single `assetID` (only when exactly one asset, not "no-asset").
- [x] [backend] Unit tests updated.
- [ ] Verify on dev: new single-asset run pod shows run-id + asset-id labels (kubectl); ~a day later BQ has the asset-id label.

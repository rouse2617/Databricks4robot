# Tasks — CYB-3705

## Context files

- `backend/config/gcp_pricing.yaml` — pricing table (structure change)
- `backend/internal/usecase/pipeline/cost.go` — `resourcesDurationToCost`, `PricingConfig`, `ComputeRunCost`
- `backend/internal/usecase/pipeline/cost_test.go` — unit tests
- `backend/internal/usecase/pipeline/usecase_cost_snapshot_test.go` — snapshot tests
- `backend/internal/usecase/pipeline/usecase.go:2288` — per-node cost derivation call site (`enrichResourcesDurationWithNode` + `resourcesDurationToCost`)

## Implementation

- [x] [backend] Rewrite `gcp_pricing.yaml` to `units[resource][provisioning]` shape (cpu / nvidia.com/gpu / memory; standard + spot)
- [x] [backend] Add `Units map[string]map[string]float64` to `PricingConfig`; keep `CalibrationFactor`; drop dead `Prices`/instance-fallback path
- [x] [backend] Rewrite `resourcesDurationToCost`: sum `Σ_r resourcesDuration[r] × unit_rate[r] / 3600`; provisioning from `rd["provisioning"]`, default `standard`; GPU only when `nvidia.com/gpu > 0`; NaN/Inf guard; return non-nil (possibly 0) cost when any positive resource duration present
- [x] [backend] Remove now-dead `lookupHourlyRate` / `maxResourceDurationSeconds`; keep `toFloat64`, `isLeafPodNode`, `ComputeRunCost` (leaf-pods-only)
- [x] [backend] Leave `enrichResourcesDurationWithNode` (still supplies `provisioning` when resolver works); no longer load-bearing for correctness

## API contract sync

No new/changed HTTP API — response shape unchanged (`estimatedCostUsd` / `totalEstimatedCostUsd`). Skipped OpenAPI/api-guide/SDK per AI-RULES (value-only change).

## Local verification (Tier M)

- [x] [backend] `gofmt` clean on changed files
- [x] [backend] `go vet ./internal/usecase/pipeline/...`
- [x] [backend] `go build ./...` (removed struct field has no residual refs)
- [x] [backend] Rewrote `cost_test.go`: cpu-only priced by vCPU rate (regression guard); gpu step = cpu + gpu; spot < standard; provisioning defaults standard; unknown resource → 0; calibration; nil/empty
- [x] [backend] Added `TestShippedPricingYAMLPricesDeliveryNodeSanely` — loads real YAML, prices the sample run's real resourcesDuration → ~$0.26
- [x] [backend] Updated `usecase_cost_snapshot_test.go` (`Prices` → `Units`)
- [x] [backend] `go test ./internal/usecase/pipeline/...` + `./internal/handlers/pipeline/...` green

## Real-data verification (offline, non-disruptive)

- [x] Pulled sample run `35961808-…` real `resourcesDuration` from live dev API: `{cpu:20270, memory:814970}`, no gpu, host `gke-delivery-clust-t2d-pool-…`
- [x] New model prices it to **$0.2595** vs old **$6.7567** (26× reduction) — locked into shipped-YAML test

## Deploy verification

- [ ] **Deferred to post-merge** (user-approved in chat; see decisions.md). Sample run is a frozen snapshot; live confirmation needs a fresh delivery-clust run after dev auto-deploy on merge.

## PR

- [ ] PR template filled; Linear CYB-3705 linked; before/after cost numbers in body

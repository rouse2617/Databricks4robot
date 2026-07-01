# Tasks — CYB-1543 Cost Estimation

## Backend

- [x] **B1: Pricing YAML** — copy czh's `gcp_pricing.yaml` to `backend/config/gcp_pricing.yaml`
  Adjust for cyber-databrew's GCP regions and machine types

- [x] **B2: Load pricing at startup** — add pricing loading to config package
  - Read YAML on startup, cache in memory
  - Fall back to empty pricing (no crash) if file missing

- [x] **B3: Cost compute helper** — `backend/internal/usecase/pipeline/cost.go`
  - `ComputeCost(resourcesDuration map[string]interface{}, pricing PricingConfig) *float64`
  - Lookup: resourcesDuration keys ("cpu", "nvidia.com/gpu") → pricing → hourly rate → cost

- [x] **B4: Migration** — `backend/migrations/046_pipeline_run_nodes_cost.sql`
  ```sql
  ALTER TABLE pipeline_run_nodes ADD COLUMN estimated_cost_usd DECIMAL(12,4);
  ```

- [x] **B5: Wire into Argo status refresh** — `backend/internal/usecase/pipeline/usecase.go`
  - When refreshing run node status, call `ComputeCost` with the node's `resourcesDuration`
  - Update `pipeline_run_nodes.estimated_cost_usd`

- [x] **B6: API return** — `GET /api/v1/pipeline-runs/:id`
  - Sum `estimated_cost_usd` across nodes
  - Return as `totalEstimatedCost` in response

## Frontend

- [ ] **F1: Run detail cost display** — `Frontend/src/pages/WorkflowDetailPage.tsx`
  - Show `💰 预估花费: $X.XX` in run summary section
  - Hide when cost is nil (no data yet)

- [ ] **F2: History list cost column (optional)** — `Frontend/src/pages/WorkflowListPage.tsx`
  - Show per-run cost in execution list

## Verification

- [x] Pricing YAML loads and caches correctly
- [x] `ComputeCost` returns expected values for known inputs
- [x] After a real pipeline run, `pipeline_run_nodes.estimated_cost_usd` is populated
- [ ] Run detail page shows cost

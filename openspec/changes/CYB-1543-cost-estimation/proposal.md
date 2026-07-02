# Proposal — CYB-1543 Cost Estimation

## Problem

Pipeline runs consume GCP resources (CPU, GPU, memory) but there is no cost visibility. Users cannot answer "how much did this run cost?" or compare costs across runs.

## Approach

Inherit CyberPipe's approach: a static GCP pricing YAML + Argo's `resourcesDuration` → per-node cost estimate.

### How it works

1. **Pricing YAML** — `backend/config/gcp_pricing.yaml` maps machine types → hourly USD rates, with a `calibration_factor` to adjust for CUD/SUD/contract discounts
2. **Resources from Argo** — each pipeline node's `resourcesDuration` records CPU/GPU seconds consumed
3. **Compute** — `resourcesDuration × hourly rate = estimated_cost_usd` per node
4. **Persist** — `pipeline_run_nodes.estimated_cost_usd` column
5. **Aggregate** — `SUM(estimated_cost_usd)` per run, per node, across all runs

### Pricing data source

P0 uses a static YAML file (same as CyberPipe). P1 can upgrade to BigQuery billing export for dynamic calibration.

```yaml
region: us-central1
calibration_factor: 1.00
prices:
  g2-standard-16:
    nvidia-l4:
      standard: 1.20
  c3d-standard-8-lssd:
    none:
      standard: 0.45
      spot: 0.14
```

### Data model

```sql
ALTER TABLE pipeline_run_nodes ADD COLUMN estimated_cost_usd DECIMAL(12,4);
```

### API changes

- `GET /api/v1/pipeline-runs/:id` returns `totalEstimatedCost` (sum of node costs)
- History/list view optionally returns aggregate cost

### Frontend

- Run detail page footer shows `💰 预估花费: $X.XX`
- History list shows per-run cost column (optional)

## Goals

- Per-node cost estimate written during Argo status refresh
- Per-run cost visible in run detail page
- Per-run aggregate in history view

## Non-Goals

- Network/storage/idle Node costs (Pod runtime only)
- External billing gateway integration
- Per-team budget and alerting (P2)

## Acceptance Criteria

- `backend/config/gcp_pricing.yaml` exists with baseline GCP prices
- `pipeline_run_nodes` has `estimated_cost_usd` column
- After a pipeline run completes, each node has a cost value
- `GET /api/v1/pipeline-runs/:id` returns `totalEstimatedCost`
- Frontend run detail shows the cost

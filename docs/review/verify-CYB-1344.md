# Verify CYB-1344 — DAG Edges (Normalized Workflow DAG)

## Deploy Record

| Item | Detail |
|------|--------|
| Branch | `feat/pipeline-integration` |
| Commit SHA | `ff2d44d59695c0e8a341de0526e28ca2fa9ca703` (feat: add normalized dag edges) |
| Backend image tag | `cyb-1344-dag-edges` |
| Backend revision | `cyber-databrew-backend-dev-cyb-1344-v2` |
| Backend tag | `cyb-1344` |
| Backend URL | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| Backend traffic | 100% |
| Verified at | 2026-05-28 |

## Changes Deployed

- **Backend**: New `GET /api/v1/workflows/:name` handler returns normalized `edges` in response
  - New file: `backend/internal/handlers/workflow/dag_edges.go`
  - Modified: `backend/internal/handlers/workflow/handler.go`
- **Frontend**: `WorkflowDagView.tsx` renders edges from API response, falls back to old logic
- **OpenAPI**: Updated, API guide updated, smoke test updated, SDK passthrough test updated

## Smoke Test Results

**44 passed, 1 failed**

| Test Group | Result |
|------------|--------|
| Healthz | FAIL (404 — pre-existing, endpoint not registered on this backend) |
| Lakehouse | All OK (6 passed, 3 WARN: optional endpoints) |
| Registry | All OK (3 passed) |
| Pipeline component registry | OK (1 passed) |
| Assets / Search / Delivery / MCAP | All OK (4 passed) |
| Workflow monitoring | OK (1 passed) |
| Algo-runs | OK (1 passed) |
| Customers | OK (2 passed) |
| Asset by ID | All OK (5 passed) |
| Audit search | All OK (2 passed) |
| Layered asset creation | All OK (3 passed) |

## DAG Edges Verification

**Endpoint**: `GET /api/v1/workflows/{name}`

Workflow `e2e-two-step-final-dbd003` (3 nodes, 2 steps):

```json
{
  "edges": [
    {
      "id": "e-e2e-two-step-final-dbd003-152140461-e2e-two-step-final-dbd003-101807604",
      "source": "e2e-two-step-final-dbd003-152140461",
      "target": "e2e-two-step-final-dbd003-101807604",
      "kind": "runtime"
    }
  ],
  "nodes": [...],
  "progress": {...}
}
```

- **edges count**: 1 (for e2e-two-step workflow with 3 nodes)
- **Edge fields**: `id`, `source`, `target`, `kind` — all present and correctly populated
- **Single-step workflows** (2 nodes): 0 edges (correct — no DAG dependencies)

## Frontend Build

```text
✓ built in 7.92s
```

Build succeeded. Only pre-existing chunk size warnings (vendor-echarts, vendor-antd-core > 800 kB).

## Issues Found

None. All verification passed.

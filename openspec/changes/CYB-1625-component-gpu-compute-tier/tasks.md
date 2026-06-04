# Tasks — CYB-1625

## Implementation
- [x] Extend pipeline resource types with `gpu` and `computeTier`.
- [x] Add GPU → `nvidia.com/gpu` manifest generation for container and script templates.
- [x] Add component manager form/detail fields for GPU and compute tier.
- [x] Preserve GPU and compute tier through component API → palette → canvas → pipeline JSON.
- [x] Update OpenAPI and API guide resource examples.
- [x] Ensure Cloud Run dev loads the pricing config so GPU runs can produce estimated cost snapshots.
- [x] Backfill completed workflow node snapshots and costs when a run finishes before the watcher sees it.
- [x] Fix execution detail node counts and unavailable-cost copy for complex GPU DAGs.

## Verification
- [x] Backend transpiler tests cover GPU manifest output.
- [x] Frontend pipeline contract tests cover GPU and compute tier round-trip.
- [x] Component manager tests cover editing and saving resource fields.
- [x] Workflow detail tests cover workflow node count and unavailable-cost copy.
- [x] Run targeted backend/frontend tests.

## Deploy verification
- [x] If deploying dev, create a component with GPU/compute tier and verify the generated manifest includes `nvidia.com/gpu`.
- [x] Re-deploy dev after the cost snapshot fix and verify a complex GPU DAG shows correct node count and cost state.

Verified on dev backend revision `cyber-databrew-backend-dev-00560-md7` with workflow
`cyb1625-complex-031650-2edcb9`: the detail page shows 5 business nodes, estimated
total cost `~$0.0090`, 5 asset-node rows, and all workflow/event/cost/detail API
requests return 200.

### Deploy record — CYB-1625
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:b439c52-cyb1625-costfix7` | `cyber-databrew-backend-dev-00560-md7` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| frontend-dev | `cyber-databrew-frontend:b439c52-cyb1625-frontend2` | `cyber-databrew-frontend-dev-00314-92p` | `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` |

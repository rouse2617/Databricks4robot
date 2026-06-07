# Proposal — CYB-1783

## Why
Dev regression found that several pipeline and asset workflows still fail after the workflow detail node-name fix: short template links open an empty designer, the asset Action timeline calls a documented path the backend does not serve, and the run modal cannot select an existing asset when search indexing misses its direct asset id.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- Pipeline template detail SHALL resolve user-facing short template ids when they uniquely match a saved template id.
- Asset action annotations SHALL be readable through the documented `/action-annotations` route while keeping existing `/actions` routes compatible.
- Pipeline run asset selection SHALL fall back to direct asset lookup when search returns no matches for an exact asset id query.

## Impact
- **Affected code**: `backend/internal/usecase/pipeline`, `backend/internal/handlers/pipeline`, `backend/routes`, `Frontend/src/components/pipeline/AssetPicker.tsx`, related tests.
- **New APIs**: None. The `/api/v1/assets/{id}/action-annotations` route already exists in OpenAPI and frontend clients; backend will serve the existing contract.
- **Dependencies**: None.

## Scope
- **In scope**: Fix short-id template detail links used by `/pipeline?templateId=<short-id>`.
- **In scope**: Add backend aliases for asset action annotation routes that match the frontend/OpenAPI contract.
- **In scope**: Add frontend direct asset lookup fallback for exact asset ids when `/search/assets` returns no rows.
- **Out of scope**: Kubernetes Pod diagnostics 403 on dev, because current evidence points to Cloud Run/GKE RBAC or credential configuration.
- **Out of scope**: Historical failed-node logs showing empty output, because the API returns 200 and this needs separate log-retention or Argo artifact investigation.

## Success Criteria
- [ ] `/api/v1/pipelines/c00f1136` resolves the same template as the full UUID `c00f1136-1f86-4a32-a25d-91bf81557fc8` when the prefix is unique.
- [ ] Opening `/pipeline?templateId=c00f1136` rehydrates the saved `test` pipeline instead of an empty new-pipeline canvas.
- [ ] `/api/v1/assets/CYB10A01/action-annotations?limit=200` returns 200 with the existing action list envelope.
- [ ] In the pipeline run modal, searching `CYB10A01` can select the existing asset even when `/search/assets?q=CYB10A01` returns zero rows.
- [ ] Existing full UUID template links, `/assets/{id}/actions`, and ordinary asset search results continue to work.

## Goals (SLO)
- **Latency**: Short-id resolution and direct-asset fallback should add at most one bounded lookup only on fallback paths.
- **Concurrency**: No change.
- **Quality**: Targeted backend and frontend regression tests cover each fixed symptom.

# Proposal — CYB-1111

## Why
Lakehouse audit analytics need a Silver asset-events export path so downstream queries do not depend directly on duplicate-prone Bronze ingestion rows.

## What Changes

### New Capabilities
- Build or refresh a Silver asset-events table from Bronze asset events.
- Surface Silver table availability through existing lakehouse operational views and smoke/docs where applicable.

### Modified Capabilities
- Lakehouse table reporting recognizes the Silver asset-events table when it exists.

## Impact
- **Affected code**: `deploy/cloudrun/bronze-incremental/`, `deploy/k8s/jobs/`, `backend/internal/handlers/lakehouse/`, `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/api-guide-smoke.sh`
- **New APIs**: None
- **Dependencies**: Existing PyIceberg/BigQuery lakehouse stack

## Scope
- **In scope**: Silver export path for asset events, idempotent rebuild/incremental notes, operational docs, table visibility, focused tests.
- **Out of scope**: Gold data marts beyond existing examples, new customer-facing endpoints, historical data backfill execution in prod.

## Success Criteria
- [ ] Silver asset-events export can be run repeatedly without corrupting existing Bronze data.
- [ ] Existing lakehouse table listing reports `silver_asset_events_current` when the table exists.
- [ ] API guide and smoke checks mention Silver availability and tolerate disabled lakehouse environments.
- [ ] Tests cover lakehouse table response shaping.

## Goals (SLO)
- **Latency**: Silver export is batch-oriented and follows Bronze ingestion cadence.
- **Quality**: Downstream analytics use deduped/current Silver semantics rather than raw duplicate Bronze rows.

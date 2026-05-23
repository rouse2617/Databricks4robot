# Proposal — CYB-1124

## Why
Follow-up review of CYB-1123 found residual consistency bugs in delivery operations and algo-runs pagination. These bugs can leave asset delivery indexes inconsistent, over-count draft items, misreport normalized pagination, or allow duplicate delivery side effects under concurrent idempotent requests.

## What Changes

### New Capabilities
- None. This is a bugfix follow-up for existing algo-runs and delivery behavior.

### Modified Capabilities
- Algo-runs list responses return normalized pagination metadata that matches the effective query.
- Delivery C2 item additions persist counts based on actual unique delivery items.
- Delivery cancellation updates delivery state and affected asset delivery indexes atomically.
- Delivery commit idempotency prevents same-key concurrent requests from creating duplicate side effects or overwriting mismatched payload records.

## Impact
- **Affected code**: `backend/internal/handlers/algorun`, `backend/internal/usecase/algorun`, `backend/internal/postgres`, `backend/internal/handlers/delivery`, `backend/internal/repository`, `scripts/api-guide-smoke.sh`
- **New APIs**: None
- **Changed APIs**: Existing response semantics for normalized pagination and delivery operation consistency
- **Dependencies**: None

## Scope
- **In scope**: Fix the four confirmed review findings, add focused backend tests, update API guide/smoke where behavior is documented or verified.
- **Out of scope**: New delivery lifecycle states, frontend redesign, schema migrations, SDK parity work.

## Success Criteria
- [ ] `GET /api/v1/algo-runs` returns effective normalized `page/page_size` in the response envelope.
- [ ] Adding duplicate assets to a pending delivery does not inflate persisted or returned item counts.
- [ ] Cancelling a committed delivery cannot leave delivery status and asset delivery indexes inconsistent if index refresh fails.
- [ ] Concurrent delivery commits with the same idempotency key do not create duplicate deliveries or overwrite a mismatched idempotency record.
- [ ] Focused tests and smoke coverage prove the corrected behavior.

## Goals (SLO)
- **Latency**: Delivery operations remain bounded by the number of affected assets; no additional unbounded scans beyond existing delivery item lookup.
- **Concurrency**: Same idempotency key concurrent commits are serialized or rejected deterministically.
- **Quality**: Focused tests cover all confirmed regressions before runtime code is considered complete.

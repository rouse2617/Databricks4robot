# Context files — CYB-1124

## Algo-runs
- `backend/internal/handlers/algorun/handler.go` — list endpoint parses query and builds response envelope.
- `backend/internal/usecase/algorun/usecase.go` — normalizes pagination for repository calls.
- `backend/internal/postgres/algo_runs.go` — applies query pagination and total count.

## Delivery
- `backend/internal/handlers/delivery/handler.go` — commit, C2 add-items, cancel, retry, and ack handlers.
- `backend/internal/postgres/repos.go` — delivery item insertion, asset index refresh, and idempotency persistence.
- `backend/internal/repository/common.go` — delivery and idempotency interfaces.
- `backend/internal/handlers/delivery/handler_test.go` — handler regression tests.
- `backend/internal/postgres/repos_test.go` — repository SQL/unit tests.

## Verification
- `docs/agents/deploy-before-commit.md` — required backend deploy gate before commit.
- `docs/agents/deploy-verification.md` — backend dev smoke verification.
- `scripts/api-guide-smoke.sh` — API contract smoke coverage.

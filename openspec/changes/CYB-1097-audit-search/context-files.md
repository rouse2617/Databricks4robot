# Context files — CYB-1097

backend/internal/handlers/audit/handler.go # Existing audit search and lineage handler skeleton.
backend/routes/routes.go # Current `/api/v1/audit/search` route registration.
backend/internal/postgres/repos.go # `asset_events` query patterns and row scan shapes.
backend/internal/repository/common.go # Existing event list option semantics.
schemas/pg-phase0.sql # `asset_events` columns and indexes.
api/openapi.yaml # Public API contract target.
docs/review/api-guide.md # Human curl guide target.
scripts/api-guide-smoke.sh # Dev contract smoke target.

# Context files — CYB-1098

backend/internal/handlers/audit/handler.go # Existing audit lineage handler skeleton and recursive CTE target.
backend/routes/routes.go # Current `/api/v1/audit/lineage-search` route registration.
schemas/pg-phase0.sql # `asset_relations` table, relation columns, and child index.
api/openapi.yaml # Public API contract target.
docs/review/api-guide.md # Human curl guide target.
scripts/api-guide-smoke.sh # Dev contract smoke target.
openspec/changes/CYB-1097-audit-search/ # Local OpenSpec style reference only; do not modify.

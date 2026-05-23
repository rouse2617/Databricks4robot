# Context files — CYB-1100

backend/routes/routes.go # Existing `/api/v1/logical-assets/:id/ratings-history` route registration.
backend/internal/handlers/asset/handler.go # Current handler skeleton, error mapping patterns, response shaping target.
backend/internal/usecase/asset/versioning.go # Logical asset current semantics and logical-asset not-found behavior.
backend/internal/usecase/asset/provenance.go # Revision list and version-history ordering conventions.
backend/internal/postgres/repos.go # `ListByLogicalAssetID` plus projection repository patterns.
backend/internal/postgres/eval_repos.go # AssetMetric row model and metric read patterns.
backend/internal/models/asset.go # Asset revision fields and JSON shape.
docs/review/unified-asset-catalog/archive/prd-rev11-historic.md # `rating.*` metric semantics and ratings-history intent.
docs/review/eval-metrics-design.md # Asset metrics source-of-truth and separation from algorithm state.
docs/review/api-guide.md # Human curl docs target.
api/openapi.yaml # Public API contract target.
scripts/api-guide-smoke.sh # Dev smoke target.

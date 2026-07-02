# Context files — AlgoRun MVP Phase 1

# Design docs
docs/review/algorn-enhancement.md  # §9 Phase 1 MVP scope
docs/review/asset-model-expansion.md  # Phase 2 ml_model types
docs/review/batch-two-roadmap.md  # Phase 1 plan

# Existing code
backend/internal/handlers/algorun/handler.go  # Existing algo-runs handlers
backend/internal/usecase/asset/usecase.go  # Asset creation + version logic
backend/internal/models/asset_type_schema.go  # ml_model/dataset schema registry
backend/internal/repository/algorun.go  # AlgoRun repo interface
backend/internal/postgres/algo_run_repo.go  # AlgoRun PG implementation
backend/internal/deliveryrules/asset_validator.go  # L1-L7 validation
backend/routes/routes.go  # Route registration

# Existing patterns
backend/internal/handlers/asset/handler.go  # CreateAsset pattern reference
backend/internal/models/asset.go  # Asset model struct
backend/internal/models/logical_asset.go  # Logical asset model

# Migration reference
backend/migrations/045_asset_model_p2.sql  # Latest migration (Phase 0)

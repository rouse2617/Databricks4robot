# Context files — 数据产线 MVP

# Design docs
docs/review/data-production-line.md  # §4 domain model, §7 template CRUD, §8 Argo renderer, §12 MVP
docs/review/batch-two-roadmap.md  # Phase 2 plan

# Existing pipeline code
backend/internal/handlers/pipeline/handler.go  # Pipeline handler reference
backend/internal/usecase/pipeline/usecase.go  # Pipeline usecase
backend/internal/transpiler/  # Argo template transpiler
backend/internal/models/pipeline.go  # Pipeline models
backend/routes/routes.go  # Route registration

# Asset registration
backend/internal/handlers/pipeline_asset/handler.go  # pipeline-assets registration
backend/internal/usecase/asset/usecase.go  # Asset version creation

# Existing patterns
backend/internal/deliveryrules/  # Validation pattern
backend/internal/postgres/  # Repo implementation pattern
backend/internal/models/asset_type_schema.go  # Schema registry
backend/migrations/  # Migration pattern

# Frontend reference
Frontend/src/components/pipeline/  # Current pipeline UI

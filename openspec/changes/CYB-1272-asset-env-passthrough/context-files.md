# Context files — CYB-1272

backend/internal/usecase/pipeline/usecase.go # asset validation, deployment, run creation, retry paths
backend/internal/transpiler/transpiler.go # global env injection into Argo templates
backend/internal/repository/asset_repository.go # asset read contract
backend/internal/models/asset.go # available asset fields for env summaries
backend/internal/usecase/pipeline/usecase_test.go # existing asset validation tests and mocks
backend/internal/usecase/pipeline/usecase_crud_test.go # deployment and output registration tests
docs/review/pipeline-dev-regression-log-2026-06-07.md # dev pipeline regression context
docs/agents/deploy-before-commit.md # backend deploy gate
docs/agents/deploy-verification.md # backend dev verification

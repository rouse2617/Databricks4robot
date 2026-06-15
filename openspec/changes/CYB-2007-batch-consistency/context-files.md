# Context files — CYB-2007

backend/internal/usecase/backfill/usecase.go # Batch creation, scheduling, reconciliation, progress, node summary
backend/internal/usecase/backfill/usecase_test.go # Existing backfill unit tests and mocks
backend/internal/postgres/backfill_repo.go # Backfill item aggregation and node coverage queries
backend/internal/postgres/pipeline_repo.go # Batch child execution list filtering and totals
backend/internal/usecase/pipeline/batch_subtask.go # Batch child run upsert behavior
Frontend/src/pages/BatchJobDetailPage.tsx # Deployed UI surface used for regression verification
Frontend/src/pages/WorkflowExecutionList.tsx # Batch detail child execution list UI and filters
docs/agents/deploy-verification.md # Dev deploy verification requirements

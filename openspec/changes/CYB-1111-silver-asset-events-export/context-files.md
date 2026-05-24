# Context files — CYB-1111

deploy/k8s/jobs/biglake-silver-gold-build-once.yaml # Existing Silver/Gold one-shot job.
deploy/cloudrun/bronze-incremental/main.py # Bronze ingest and related Silver quality path.
backend/internal/handlers/lakehouse/handler.go # Lakehouse table listing.
backend/internal/handlers/lakehouse/handler_test.go # Handler tests.
docs/review/lakehouse-incremental-ingestion.md # Bronze duplicate/cursor semantics.
docs/review/api-guide.md # Lakehouse API docs.
scripts/api-guide-smoke.sh # Dev smoke coverage.

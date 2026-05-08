.PHONY: all dev-up dev-down all-up all-down all-reset-volumes all-logs local-migrate local-dev-seed cdc-up cdc-down cdc-loadtest cdc-reconcile-report cdc-go-no-go pg-generate-scale pg-generate-rich iceberg-up iceberg-down iceberg-logs iceberg-mvp iceberg-mvp-host test test-full smoke smoke-local frontend-test build clean seed-rich verify-rich-seed test-e2e-rich

# ── Local infra ──────────────────────────────────────────
dev-up:
	cd deploy/local && docker compose up -d
	@echo "PostgreSQL:       localhost:5432"
	@echo "PgBouncer:        localhost:6432"
	@echo "Bigtable emulator: localhost:8086  |  Pubsub emulator: localhost:8085"

dev-down:
	cd deploy/local && docker compose --profile full --profile lakehouse --profile cdc down || true
	cd deploy/local && docker compose down

# CDC bus (Redpanda + Debezium Connect). Requires base Postgres (`make dev-up`) first.
cdc-up:
	cd deploy/local && docker compose --profile cdc up -d
	@echo "Kafka:          localhost:19092"
	@echo "Connect REST:   http://localhost:8084"
	@echo "Connector:      postgres-unified-cdc (auto-registered)"

cdc-down:
	cd deploy/local && docker compose --profile cdc down

cdc-loadtest:
	bash backend/scripts/cdc_loadtest.sh

cdc-reconcile-report:
	bash backend/scripts/cdc_reconcile_report.sh

cdc-go-no-go: cdc-reconcile-report
	@echo "Go/No-Go quick check finished. Review cdc-reconcile-report output and runbook thresholds."

# BuildKit: cache mounts in Frontend/backend Dockerfiles need DOCKER_BUILDKIT=1 (default on Docker Desktop).
# Incremental rebuilds: `cd deploy/local && docker compose build frontend` (avoid `--no-cache` unless debugging).
# Optional: NPM_REGISTRY=https://registry.npmmirror.com GOPROXY=https://goproxy.cn,direct make all-up
all-up:
	cd deploy/local && DOCKER_BUILDKIT=1 docker compose --profile full up -d --build
	@echo "Frontend: http://localhost:5173"
	@echo "Backend:  http://localhost:8080"
	@echo "Trino:    http://localhost:8082"
	@echo "ES:       http://localhost:9200"
	@echo "Kafka:    localhost:19092"
	@echo "Connect:  http://localhost:8084"
	@echo "Prom:     http://localhost:9090"
	@echo "Grafana:  http://localhost:3000  (admin / admin)"

all-down:
	cd deploy/local && docker compose --profile full down

# Tear down **all** Compose services and remove named volumes (PostgreSQL data wiped).
# Next `make all-up` runs initdb + DDL migrations only (no demo rows).
# Optional demo data: `make local-dev-seed` after Postgres is up.
all-reset-volumes:
	cd deploy/local && docker compose --profile full --profile lakehouse --profile cdc down -v || true
	cd deploy/local && docker compose down -v || true
	@echo "Volumes removed. Run: make all-up"

# Apply SQL migrations missing from an existing Postgres data dir (e.g. after adding 016_add_actions.sql).
# Uses docker exec local-postgres-1 when that container is running; otherwise localhost psql.
local-migrate:
	bash backend/scripts/ensure_migrations.sh

# Insert optional dev/demo rows (formerly auto-run 002_seed.sql). Idempotent skip if aset0001 exists.
local-dev-seed:
	bash backend/scripts/apply_dev_seed.sh

all-logs:
	cd deploy/local && docker compose --profile full logs -f

pg-generate-scale:
	psql postgresql://postgres:postgres@localhost:5432/data4cyber \
		-v row_count=$${ROW_COUNT:-100000} \
		-v batch_id=$${BATCH_ID:-scale_100k} \
		-f backend/scripts/generate_mock_scale.sql

pg-generate-rich:
	BATCH_ID=$${BATCH_ID:-rich_10k_2k} \
	ASSET_COUNT=$${ASSET_COUNT:-10000} \
	MCAP_COUNT=$${MCAP_COUNT:-2000} \
	DELIVERY_COUNT=$${DELIVERY_COUNT:-600} \
	bash backend/scripts/load_rich_scale.sh

iceberg-up:
	cd deploy/local && docker compose --profile lakehouse up -d
	@echo "Spark Notebook: http://localhost:8889"
	@echo "Spark UI:       http://localhost:8083"
	@echo "MinIO Console:  http://localhost:9001  (admin / password)"
	@echo "Iceberg REST:   http://localhost:8183"
	@echo "Trino:          http://localhost:8082"

iceberg-down:
	cd deploy/local && docker compose --profile lakehouse down

iceberg-logs:
	cd deploy/local && docker compose --profile lakehouse logs -f

iceberg-mvp:
	cd deploy/local && docker compose --profile lakehouse exec -T spark-iceberg \
		python /home/iceberg/notebooks/notebooks/build_lakehouse_mvp.py

iceberg-mvp-host:
	cd deploy/local && docker compose --profile lakehouse exec -T \
		-e POSTGRES_URL=jdbc:postgresql://host.docker.internal:5432/data4cyber \
		spark-iceberg python /home/iceberg/notebooks/notebooks/build_lakehouse_mvp.py

trino-smoke:
	cd deploy/local && docker compose --profile lakehouse exec -T trino \
		trino --server http://localhost:8080 --catalog iceberg --schema robot \
		--execute "SHOW TABLES"

# ── Backend ───────────────────────────────────────────────
backend-deps:
	cd backend && make deps

backend-build:
	cd backend && make build

backend-test:
	cd backend && make test

backend-run:
	cd backend && make run-server

# ── SDK ───────────────────────────────────────────────────
sdk-install:
	cd sdk && uv sync --dev

sdk-test:
	cd sdk && uv run pytest tests/unit/

sdk-lint:
	cd sdk && uv run ruff check src/

# ── Frontend ──────────────────────────────────────────────
frontend-install:
	cd Frontend && npm install

frontend-dev:
	cd Frontend && npm run dev

frontend-build:
	cd Frontend && npm run build

# Backward-compatible aliases
ui-install: frontend-install
ui-dev: frontend-dev
ui-build: frontend-build

# ── Rich mock data (HTTP seed; requires `make all-up` backend) ─
# SEED_TOTAL / SEED_WORKERS / SEED_BASE / SEED_TOKEN optional
seed-rich:
	python3 backend/scripts/seed_rich_dataset.py \
		--base $${SEED_BASE:-http://127.0.0.1:8080} \
		--token $${SEED_TOKEN:-dev-token} \
		--total $${SEED_TOTAL:-100} \
		--workers $${SEED_WORKERS:-8}

verify-rich-seed:
	bash backend/scripts/verify_rich_seed.sh

# Seed DB + API smoke + full Go test suite (long)
test-e2e-rich: seed-rich verify-rich-seed backend-test

# ── Dagster ───────────────────────────────────────────────
dagster-dev:
	cd dagster && uv run dagster dev -f definitions.py

# ── Combined ──────────────────────────────────────────────
test: backend-test sdk-test

# Backend + Frontend unit tests (SDK optional via sdk-test if uv installed).
test-full: backend-test frontend-test
	@if command -v uv >/dev/null 2>&1; then $(MAKE) sdk-test; else echo "skip sdk-test (uv not installed)"; fi

frontend-test:
	cd Frontend && npm test

# Requires backend on :8080; optional SEG_ASSET_ID for GET .../actions smoke.
smoke-local:
	bash backend/scripts/ensure_migrations.sh
	GRACE_TOKEN=$${GRACE_TOKEN:-dev-token} SEG_ASSET_ID=$${SEG_ASSET_ID:-} bash backend/scripts/smoke_actions_api.sh

smoke:
	@echo "Use: make smoke-local (needs Postgres + backend). See backend/scripts/smoke_actions_api.sh"

# GCE VM: set GCP_VM_NAME, GCP_ZONE, then run deploy/gcp-vm/deploy.sh (see deploy/gcp-vm/README.md)
gcp-vm-deploy:
	@test -n "$$GCP_VM_NAME" -a -n "$$GCP_ZONE" || (echo "Set GCP_VM_NAME and GCP_ZONE" && false)
	./deploy/gcp-vm/deploy.sh

build: backend-build frontend-build

clean:
	cd backend && make clean
	cd Frontend && rm -rf dist node_modules

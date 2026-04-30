.PHONY: all dev-up dev-down all-up all-down all-logs pg-generate-scale iceberg-up iceberg-down iceberg-logs iceberg-mvp iceberg-mvp-host test smoke build clean

# ── Local infra ──────────────────────────────────────────
dev-up:
	cd deploy/local && docker compose up -d
	@echo "PostgreSQL:       localhost:5432"
	@echo "PgBouncer:        localhost:6432"
	@echo "Bigtable emulator: localhost:8086  |  Pubsub emulator: localhost:8085"

dev-down:
	cd deploy/local && docker compose down

all-up:
	cd deploy/local && docker compose -f docker-compose.all.yml up -d --build
	@echo "Frontend: http://localhost:5173"
	@echo "Backend:  http://localhost:8080"
	@echo "Trino:    http://localhost:8082"
	@echo "ES:       http://localhost:9200"

all-down:
	cd deploy/local && docker compose -f docker-compose.all.yml down

all-logs:
	cd deploy/local && docker compose -f docker-compose.all.yml logs -f

pg-generate-scale:
	psql postgresql://postgres:postgres@localhost:5432/data4cyber \
		-v row_count=$${ROW_COUNT:-100000} \
		-v batch_id=$${BATCH_ID:-scale_100k} \
		-f backend/scripts/generate_mock_scale.sql

iceberg-up:
	cd deploy/local && docker compose -f docker-compose.iceberg.yml up -d
	@echo "Spark Notebook: http://localhost:8888"
	@echo "MinIO Console:  http://localhost:9001  (admin / password)"
	@echo "Iceberg REST:   http://localhost:8181"
	@echo "Trino:          http://localhost:8082"

iceberg-down:
	cd deploy/local && docker compose -f docker-compose.iceberg.yml down

iceberg-logs:
	cd deploy/local && docker compose -f docker-compose.iceberg.yml logs -f

iceberg-mvp:
	docker compose -f deploy/local/docker-compose.iceberg.yml exec -T spark-iceberg \
		python /home/iceberg/notebooks/notebooks/build_lakehouse_mvp.py

iceberg-mvp-host:
	docker compose -f deploy/local/docker-compose.iceberg.yml exec -T \
		-e POSTGRES_URL=jdbc:postgresql://host.docker.internal:5432/data4cyber \
		spark-iceberg python /home/iceberg/notebooks/notebooks/build_lakehouse_mvp.py

trino-smoke:
	docker compose -f deploy/local/docker-compose.iceberg.yml exec -T trino \
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

# ── Dagster ───────────────────────────────────────────────
dagster-dev:
	cd dagster && uv run dagster dev -f definitions.py

# ── Combined ──────────────────────────────────────────────
test: backend-test sdk-test

smoke:
	@echo "TODO: wire e2e smoke script in scripts/e2e_smoke.sh"

build: backend-build frontend-build

clean:
	cd backend && make clean
	cd Frontend && rm -rf dist node_modules

.PHONY: all dev-up dev-down test smoke build clean

# ── Local infra ──────────────────────────────────────────
dev-up:
	cd deploy/local && docker compose up -d
	@echo "Bigtable emulator: localhost:8086  |  Pubsub emulator: localhost:8085"

dev-down:
	cd deploy/local && docker compose down

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

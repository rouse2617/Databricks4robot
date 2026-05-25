.PHONY: install test test-all lint mypy format build validate clean

install:
	uv sync --dev

test:
	uv run pytest tests/unit/ -v

test-all:
	uv run pytest tests/ -v

lint:
	uv run ruff check src/cyber_databrew_sdk/

mypy:
	uv run mypy src/cyber_databrew_sdk/ --ignore-missing-imports

format:
	uv run ruff format src/

build:
	uv build

# ──────────────────────────────────────────────────────────────────
# Pre-release validation gate — run before every sdk/v* tag.
# Order: lint → mypy → test → build → smoke-install
# ──────────────────────────────────────────────────────────────────
validate: lint mypy test build
	@echo "=== Smoke test: install built wheel ==="
	uv venv /tmp/sdk-smoke-venv --seed 2>&1 | tail -1
	. /tmp/sdk-smoke-venv/bin/activate && \
		pip install --quiet dist/*.whl && \
		python -c "from cyber_databrew_sdk import CyberDatabrewClient; print('Smoke OK:', CyberDatabrewClient())"
	rm -rf /tmp/sdk-smoke-venv
	@echo "=== All validation gates passed ==="

# ──────────────────────────────────────────────────────────────────
# Publish to Artifact Registry (us-central1)
# Uses gcloud oauth token — no static credentials needed.
# ──────────────────────────────────────────────────────────────────
.PHONY: publish
publish: validate
	@echo "=== Publishing to Artifact Registry ==="
	uv run twine upload \
		--repository-url https://us-central1-python.pkg.dev/green-valley-442103/python-packages/ \
		-u oauth2accesstoken \
		-p "$$(gcloud auth print-access-token)" \
		dist/*.whl dist/*.tar.gz
	@echo "=== Published ==="

clean:
	rm -rf dist/ build/ *.egg-info
	rm -rf /tmp/sdk-smoke-venv

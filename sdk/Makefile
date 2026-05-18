.PHONY: install test lint format

install:
	uv sync --dev

test:
	uv run pytest tests/unit/

test-integration:
	uv run pytest tests/integration/

lint:
	uv run ruff check src/

format:
	uv run ruff format src/

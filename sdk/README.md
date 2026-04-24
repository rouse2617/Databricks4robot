# SDK

Python SDK for consuming backend APIs (`assets`, `mcap`, `delivery`).

## What

- `GraceClient` as the main entrypoint
- Resource clients:
  - `AssetClient`
  - `McapClient`
  - `DeliveryClient`
- Pydantic models in `src/grace_sdk/types/`

## How to Run

```bash
uv sync --dev
uv run pytest tests/unit/
```

## Config

Environment variables used by default:

- `GRACE_BASE_URL` (default: `http://localhost:8080`)
- `GRACE_TOKEN`

Or pass them explicitly to `GraceClient(...)`.

## API / Interfaces

Minimal usage:

```python
from grace_sdk import GraceClient

with GraceClient(base_url="http://localhost:8080", token="dev-token") as client:
    asset = client.assets.get("asset-id")
```

## Directory Structure

- `src/grace_sdk/client.py`: top-level client
- `src/grace_sdk/assets.py`: asset APIs
- `src/grace_sdk/mcap.py`: finalize/message APIs
- `src/grace_sdk/delivery.py`: delivery APIs
- `tests/unit/`: unit tests (respx + httpx mock)

## Development Workflow

```bash
uv run pytest tests/unit/
uv run ruff check src/
uv run ruff format src/
```

## Known Limitations

- Integration tests are not fully wired yet
- Some APIs are placeholder responses until backend logic is expanded

## Next Milestones

- Add integration tests against local backend
- Generate typed models from OpenAPI
- Add pagination/filter helpers in client APIs


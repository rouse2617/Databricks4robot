# Dagster

Workflow orchestration skeleton for ingest/tag/report jobs.

## What

- `definitions.py`: Dagster entrypoint
- `jobs/`: job definitions
- `assets/`: software-defined assets
- `sensors/`: trigger logic (currently placeholder)

## How to Run

```bash
uv sync --dev
uv run dagster dev -f definitions.py
```

Dagster UI defaults to `http://localhost:3000`.

## Config

Expected to reuse backend-facing env when wired fully:

- `GRACE_BASE_URL`
- `GRACE_TOKEN`
- `PUBSUB_PROJECT`
- `TOPIC_MCAP_FINALIZED`

## API / Interfaces

- Jobs should invoke backend APIs through `grace-sdk`
- Sensor will consume finalized-upload events and trigger ingest jobs

## Directory Structure

- `definitions.py`: exported `Definitions`
- `jobs/__init__.py`: `ingest_mcap_job`
- `assets/__init__.py`: placeholder asset graph
- `sensors/__init__.py`: placeholder sensor

## Development Workflow

```bash
uv run dagster dev -f definitions.py
uv run pytest
```

## Known Limitations

- Current jobs/assets are placeholders
- No real Pub/Sub subscription wired yet
- No production resources config yet

## Next Milestones

- Implement real sensor for `gcs.mcap.finalized.v1`
- Add ingest job that updates `mcap_files` and creates assets
- Add environment-specific resource configs (dev/staging/prod)


# Bronze incremental ingestion — pulls new asset_events from Postgres and
# appends them to the BigLake-managed Iceberg table on every Cloud Scheduler
# tick (every 5 min in dev). Runs as a Cloud Run Job, exits when caught up
# or when the per-task deadline is reached.
FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# psycopg2-binary ships its own libpq, but we still want libssl present for TLS.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt ./
RUN pip install -r requirements.txt

COPY main.py ./

ENTRYPOINT ["python", "/app/main.py"]

"""
Dagster sensor: poll Postgres max(updated_at) from assets table.

When the max updated_at changes compared to the last observed value,
trigger the full lakehouse pipeline (postgres_to_bronze → silver → gold).
"""

import os

from dagster import (
    sensor,
    RunRequest,
    SensorEvaluationContext,
    SkipReason,
    DefaultSensorStatus,
)

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

PG_HOST = os.getenv("PG_HOST", "localhost")
PG_PORT = os.getenv("PG_PORT", "5432")
PG_USER = os.getenv("PG_USER", "postgres")
PG_PASSWORD = os.getenv("PG_PASSWORD", "postgres")
PG_DATABASE = os.getenv("PG_DATABASE", "data4cyber")


# ---------------------------------------------------------------------------
# Helper
# ---------------------------------------------------------------------------


def _get_pg_max_updated_at() -> str | None:
    """
    Query Postgres for the maximum updated_at from the assets table.
    Returns ISO timestamp string or None if table is empty / unreachable.
    """
    import psycopg2

    try:
        conn = psycopg2.connect(
            host=PG_HOST, port=int(PG_PORT),
            user=PG_USER, password=PG_PASSWORD,
            dbname=PG_DATABASE,
        )
        try:
            with conn.cursor() as cur:
                cur.execute("SELECT max(updated_at) FROM assets")
                row = cur.fetchone()
                if row and row[0]:
                    return row[0].isoformat()
        finally:
            conn.close()
    except Exception:
        # Postgres may be unreachable — skip this tick
        return None
    return None


# ---------------------------------------------------------------------------
# Sensor definition
# ---------------------------------------------------------------------------


@sensor(
    job_name="lakehouse_sync_job",
    minimum_interval_seconds=60,
    default_status=DefaultSensorStatus.STOPPED,
    description=(
        "Polls Postgres SELECT max(updated_at) FROM assets every 60s. "
        "When the value changes, triggers the lakehouse sync pipeline."
    ),
)
def pg_assets_change_sensor(context: SensorEvaluationContext):
    """
    Compare current max(updated_at) with the last observed cursor value.
    If changed, emit a RunRequest to trigger the lakehouse pipeline.
    """
    current_max = _get_pg_max_updated_at()

    if current_max is None:
        yield SkipReason("Could not read max(updated_at) from Postgres assets table")
        return

    last_observed = context.cursor

    if last_observed == current_max:
        yield SkipReason(
            f"No change detected (max updated_at = {current_max})"
        )
        return

    context.log.info(
        f"Change detected: {last_observed} → {current_max}. "
        f"Triggering lakehouse sync pipeline."
    )

    # Update cursor to the new max value
    context.update_cursor(current_max)

    yield RunRequest(
        run_key=f"pg-change-{current_max}",
        tags={"trigger": "pg_assets_change_sensor", "max_updated_at": current_max},
    )

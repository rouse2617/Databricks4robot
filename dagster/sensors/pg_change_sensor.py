"""
Dagster sensor: poll Postgres lakehouse change watermark.

The preferred trigger source is the event/outbox table (`asset_events`).
For the current MVP schema, where `asset_events` may not exist yet, the
sensor falls back to existing mutable tables so local sync still works.
"""

from __future__ import annotations

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


def _table_exists(cur, table_name: str) -> bool:
    """Return True when the given table exists in the current schema."""
    cur.execute("SELECT to_regclass(%s)", (table_name,))
    row = cur.fetchone()
    return bool(row and row[0])


def _get_pg_change_watermark() -> str | None:
    """
    Query Postgres for the latest change watermark.

    Priority:
    1. asset_events occurred_at/created_at — target event/outbox source.
    2. Existing MVP tables — compatibility until all writes emit events.

    Returns a stable string suitable for the Dagster sensor cursor.
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
                candidates: list[tuple[str, object]] = []

                if _table_exists(cur, "asset_events"):
                    cur.execute(
                        """
                        SELECT max(coalesce(occurred_at, created_at))
                        FROM asset_events
                        """
                    )
                    row = cur.fetchone()
                    if row and row[0]:
                        candidates.append(("asset_events", row[0]))

                # Compatibility path for the current MVP schema.
                fallback_queries = [
                    ("asset_algo_events", "SELECT max(created_at) FROM asset_algo_events"),
                    ("assets", "SELECT max(updated_at) FROM assets"),
                    ("mcap_files", "SELECT max(updated_at) FROM mcap_files"),
                    ("deliveries", "SELECT max(updated_at) FROM deliveries"),
                    ("delivery_items", "SELECT max(created_at) FROM delivery_items"),
                ]
                for source, query in fallback_queries:
                    if not _table_exists(cur, source):
                        continue
                    cur.execute(query)
                    row = cur.fetchone()
                    if row and row[0]:
                        candidates.append((source, row[0]))

                if not candidates:
                    return None

                source, watermark = max(candidates, key=lambda item: item[1])
                return f"{source}:{watermark.isoformat()}"
        finally:
            conn.close()
    except Exception:
        # Postgres may be unreachable — skip this tick
        return None
    return None


def _get_pg_max_updated_at() -> str | None:
    """Backward-compatible alias for older tests/imports."""
    return _get_pg_change_watermark()


# ---------------------------------------------------------------------------
# Sensor definition
# ---------------------------------------------------------------------------


@sensor(
    job_name="lakehouse_sync_job",
    minimum_interval_seconds=60,
    default_status=DefaultSensorStatus.STOPPED,
    description=(
        "Polls Postgres lakehouse change watermark every 60s. "
        "When the value changes, triggers the lakehouse sync pipeline."
    ),
)
def pg_assets_change_sensor(context: SensorEvaluationContext):
    """
    Compare current Postgres change watermark with the last observed cursor value.
    If changed, emit a RunRequest to trigger the lakehouse pipeline.
    """
    current_max = _get_pg_change_watermark()

    if current_max is None:
        yield SkipReason("Could not read lakehouse change watermark from Postgres")
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
        tags={"trigger": "pg_assets_change_sensor", "change_watermark": current_max},
    )

"""
Dagster job + schedule: Iceberg table maintenance.

Runs daily (every 24h) and performs three maintenance operations on all
Iceberg tables in the ``rest_catalog.default`` namespace:

1. **Expire Snapshots** — remove snapshots older than 7 days to reclaim
   storage.
2. **Rewrite Data Files (Compaction)** — merge small data files produced
   by incremental writes into larger files for better query performance.
   Only applied to Silver and Gold tables.
3. **Remove Orphan Files** — delete data files that are no longer
   referenced by any table metadata.

The job is registered in ``definitions.py`` and triggered by a ``@daily``
Dagster schedule.
"""

import os
from datetime import datetime, timedelta, timezone

from dagster import (
    job,
    op,
    schedule,
    OpExecutionContext,
    ScheduleDefinition,
)

# ---------------------------------------------------------------------------
# Configuration — same env vars as the pipeline assets
# ---------------------------------------------------------------------------

ICEBERG_REST_URI = os.getenv("ICEBERG_REST_URI", "http://localhost:8183")
S3_ENDPOINT = os.getenv("S3_ENDPOINT", "http://localhost:9000")
S3_ACCESS_KEY = os.getenv("S3_ACCESS_KEY", "admin")
S3_SECRET_KEY = os.getenv("S3_SECRET_KEY", "password")
S3_REGION = os.getenv("S3_REGION", "us-east-1")

# Iceberg catalog + namespace
CATALOG_NS = "rest_catalog.default"

# ---------------------------------------------------------------------------
# Table lists
# ---------------------------------------------------------------------------

BRONZE_TABLES = [
    "bronze_assets",
    "bronze_mcap_files",
    "bronze_deliveries",
    "bronze_delivery_items",
    "bronze_asset_events",
]

SILVER_TABLES = [
    "silver_assets_current",
    "silver_asset_tags",
    "silver_asset_algo_latest",
    "silver_mcap_files_current",
    "silver_deliveries_current",
]

GOLD_TABLES = [
    "gold_dataset_snapshot_items",
    "gold_asset_search_docs",
]

ALL_TABLES = BRONZE_TABLES + SILVER_TABLES + GOLD_TABLES
SILVER_GOLD_TABLES = SILVER_TABLES + GOLD_TABLES

# Snapshot retention period
SNAPSHOT_RETENTION_DAYS = 7


# ---------------------------------------------------------------------------
# Spark session helper
# ---------------------------------------------------------------------------


def _get_spark():
    """
    Create a SparkSession configured for Iceberg REST Catalog + MinIO.
    Reuses the same configuration as the pipeline assets.
    """
    from pyspark.sql import SparkSession

    spark = (
        SparkSession.builder
        .appName("iceberg_maintenance")
        .config("spark.sql.catalog.rest_catalog",
                "org.apache.iceberg.spark.SparkCatalog")
        .config("spark.sql.catalog.rest_catalog.type", "rest")
        .config("spark.sql.catalog.rest_catalog.uri", ICEBERG_REST_URI)
        .config("spark.sql.catalog.rest_catalog.io-impl",
                "org.apache.iceberg.aws.s3.S3FileIO")
        .config("spark.sql.catalog.rest_catalog.s3.endpoint", S3_ENDPOINT)
        .config("spark.sql.catalog.rest_catalog.s3.access-key-id",
                S3_ACCESS_KEY)
        .config("spark.sql.catalog.rest_catalog.s3.secret-access-key",
                S3_SECRET_KEY)
        .config("spark.sql.catalog.rest_catalog.s3.region", S3_REGION)
        .config("spark.sql.catalog.rest_catalog.s3.path-style-access", "true")
        .config("spark.sql.defaultCatalog", "rest_catalog")
        .config("spark.sql.extensions",
                "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions")
        .config("spark.jars.packages",
                "org.apache.iceberg:iceberg-spark-runtime-3.5_2.12:1.5.2,"
                "org.apache.iceberg:iceberg-aws-bundle:1.5.2")
        .getOrCreate()
    )
    return spark


# ---------------------------------------------------------------------------
# Ops
# ---------------------------------------------------------------------------


@op(description="Expire Iceberg snapshots older than 7 days for all tables.")
def expire_snapshots_op(context: OpExecutionContext) -> dict:
    """
    For every Iceberg table, call the Iceberg ``expire_snapshots`` stored
    procedure to remove snapshots older than the retention threshold.
    """
    spark = _get_spark()
    older_than = (
        datetime.now(timezone.utc) - timedelta(days=SNAPSHOT_RETENTION_DAYS)
    ).strftime("%Y-%m-%d %H:%M:%S")

    results: dict[str, str] = {}
    for table in ALL_TABLES:
        fqn = f"{CATALOG_NS}.{table}"
        try:
            spark.sql(
                f"CALL rest_catalog.system.expire_snapshots("
                f"table => '{fqn}', older_than => TIMESTAMP '{older_than}')"
            )
            context.log.info("expire_snapshots OK: %s", fqn)
            results[table] = "ok"
        except Exception as exc:
            context.log.warning("expire_snapshots FAILED for %s: %s", fqn, exc)
            results[table] = f"error: {exc}"

    spark.stop()
    context.log.info("Expire snapshots complete: %s", results)
    return results


@op(description="Rewrite (compact) data files for Silver and Gold tables.")
def rewrite_data_files_op(context: OpExecutionContext, start: dict = None) -> dict:
    """
    Merge small data files into larger ones for Silver and Gold tables.
    Bronze tables are excluded because they are append-heavy and compaction
    would conflict with the incremental write pattern.
    """
    spark = _get_spark()

    results: dict[str, str] = {}
    for table in SILVER_GOLD_TABLES:
        fqn = f"{CATALOG_NS}.{table}"
        try:
            spark.sql(
                f"CALL rest_catalog.system.rewrite_data_files("
                f"table => '{fqn}')"
            )
            context.log.info("rewrite_data_files OK: %s", fqn)
            results[table] = "ok"
        except Exception as exc:
            context.log.warning(
                "rewrite_data_files FAILED for %s: %s", fqn, exc
            )
            results[table] = f"error: {exc}"

    spark.stop()
    context.log.info("Rewrite data files complete: %s", results)
    return results


@op(description="Remove orphan files for all Iceberg tables.")
def remove_orphan_files_op(context: OpExecutionContext, start: dict = None) -> dict:
    """
    Clean up data files that are no longer referenced by any table metadata
    snapshot.
    """
    spark = _get_spark()

    results: dict[str, str] = {}
    for table in ALL_TABLES:
        fqn = f"{CATALOG_NS}.{table}"
        try:
            spark.sql(
                f"CALL rest_catalog.system.remove_orphan_files("
                f"table => '{fqn}')"
            )
            context.log.info("remove_orphan_files OK: %s", fqn)
            results[table] = "ok"
        except Exception as exc:
            context.log.warning(
                "remove_orphan_files FAILED for %s: %s", fqn, exc
            )
            results[table] = f"error: {exc}"

    spark.stop()
    context.log.info("Remove orphan files complete: %s", results)
    return results


# ---------------------------------------------------------------------------
# Job — chains the three maintenance ops
# ---------------------------------------------------------------------------


@job(
    name="iceberg_maintenance_job",
    description=(
        "Daily Iceberg maintenance: expire snapshots (7-day retention), "
        "compact Silver/Gold data files, remove orphan files."
    ),
)
def iceberg_maintenance_job():
    expire_result = expire_snapshots_op()
    compact_result = rewrite_data_files_op(start=expire_result)
    remove_orphan_files_op(start=compact_result)


# ---------------------------------------------------------------------------
# Schedule — @daily (every 24 h)
# ---------------------------------------------------------------------------

iceberg_maintenance_schedule = ScheduleDefinition(
    job=iceberg_maintenance_job,
    cron_schedule="0 3 * * *",  # daily at 03:00 UTC
    name="iceberg_maintenance_daily",
    description="Run Iceberg maintenance (expire snapshots, compaction, orphan cleanup) daily at 03:00 UTC.",
)

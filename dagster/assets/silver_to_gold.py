"""
Dagster assets: Silver → Gold Iceberg tables + data reconciliation.

Gold tables:
  - gold_dataset_snapshot_items  (hand_tracking ok + quality good/excellent)
  - gold_asset_search_docs       (wide table for OpenSearch sync)

Reconciliation:
  - Compare Postgres count vs Iceberg silver_assets_current count
  - Compare status distribution differences
  - Write results to sync_reconciliation Postgres table or JSON
  - Alert when difference > 0.1%
"""

import json
import os
from datetime import datetime, timezone

import dagster
from dagster import asset, Output, AssetExecutionContext

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

PG_HOST = os.getenv("PG_HOST", "localhost")
PG_PORT = os.getenv("PG_PORT", "5432")
PG_USER = os.getenv("PG_USER", "postgres")
PG_PASSWORD = os.getenv("PG_PASSWORD", "postgres")
PG_DATABASE = os.getenv("PG_DATABASE", "data4cyber")

ICEBERG_REST_URI = os.getenv("ICEBERG_REST_URI", "http://localhost:8183")
S3_ENDPOINT = os.getenv("S3_ENDPOINT", "http://localhost:9000")
S3_ACCESS_KEY = os.getenv("S3_ACCESS_KEY", "admin")
S3_SECRET_KEY = os.getenv("S3_SECRET_KEY", "password")
S3_REGION = os.getenv("S3_REGION", "us-east-1")

SOURCE_DB = "data4cyber"
CATALOG_NS = "rest_catalog.default"

# Reconciliation threshold — alert when difference exceeds this ratio
RECONCILIATION_THRESHOLD = 0.001  # 0.1%


# ---------------------------------------------------------------------------
# Spark session helper
# ---------------------------------------------------------------------------


def _get_spark():
    """
    Create or retrieve a SparkSession configured for Iceberg REST Catalog
    writing to MinIO (S3-compatible).
    """
    from pyspark.sql import SparkSession

    spark = (
        SparkSession.builder
        .appName("silver_to_gold")
        .config("spark.sql.catalog.rest_catalog", "org.apache.iceberg.spark.SparkCatalog")
        .config("spark.sql.catalog.rest_catalog.type", "rest")
        .config("spark.sql.catalog.rest_catalog.uri", ICEBERG_REST_URI)
        .config("spark.sql.catalog.rest_catalog.io-impl", "org.apache.iceberg.aws.s3.S3FileIO")
        .config("spark.sql.catalog.rest_catalog.s3.endpoint", S3_ENDPOINT)
        .config("spark.sql.catalog.rest_catalog.s3.access-key-id", S3_ACCESS_KEY)
        .config("spark.sql.catalog.rest_catalog.s3.secret-access-key", S3_SECRET_KEY)
        .config("spark.sql.catalog.rest_catalog.s3.region", S3_REGION)
        .config("spark.sql.catalog.rest_catalog.s3.path-style-access", "true")
        .config("spark.sql.defaultCatalog", "rest_catalog")
        .config("spark.sql.extensions",
                "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions")
        .config("spark.jars.packages",
                "org.apache.iceberg:iceberg-spark-runtime-3.5_2.12:1.5.2,"
                "org.postgresql:postgresql:42.7.3,"
                "org.apache.iceberg:iceberg-aws-bundle:1.5.2")
        .getOrCreate()
    )
    return spark


# ---------------------------------------------------------------------------
# Audit column helper
# ---------------------------------------------------------------------------


def _add_audit_columns(df, source_table: str, dagster_run_id: str):
    """Add audit columns required by all Gold tables."""
    from pyspark.sql.functions import lit, current_timestamp

    return (
        df
        .withColumn("_sync_timestamp", current_timestamp())
        .withColumn("_source_db", lit(SOURCE_DB))
        .withColumn("_source_table", lit(source_table))
        .withColumn("_dagster_run_id", lit(dagster_run_id))
    )


# ---------------------------------------------------------------------------
# Task 12.1 — gold_dataset_snapshot_items
# ---------------------------------------------------------------------------


@asset(
    group_name="gold",
    deps=["silver_assets_current", "silver_asset_tags", "silver_asset_algo_latest"],
    description=(
        "Gold dataset snapshot: assets where hand_tracking status=ok "
        "AND quality tag in (good, excellent). "
        "Produces gold_dataset_snapshot_items."
    ),
)
def gold_dataset_snapshot_items(context: AssetExecutionContext) -> Output[dict]:
    """
    Join silver_assets_current with silver_asset_algo_latest (hand_tracking ok)
    and silver_asset_tags (quality good/excellent) to produce a curated
    dataset snapshot for training pipelines.
    """
    spark = _get_spark()
    try:
        from pyspark.sql.functions import col

        # Read Silver tables
        assets = spark.read.table(f"{CATALOG_NS}.silver_assets_current")
        algo = spark.read.table(f"{CATALOG_NS}.silver_asset_algo_latest")
        tags = spark.read.table(f"{CATALOG_NS}.silver_asset_tags")

        # Filter: hand_tracking algo with status = ok
        ht_ok = (
            algo
            .filter(col("algo_name") == "hand_tracking")
            .filter(col("status") == "ok")
            .select("asset_id")
            .distinct()
        )

        # Filter: quality tag in (good, excellent)
        quality_ok = (
            tags
            .filter(col("tag_key") == "quality")
            .filter(col("tag_value").isin("good", "excellent"))
            .select("asset_id")
            .distinct()
        )

        # Inner join: assets that satisfy BOTH conditions
        snapshot = (
            assets
            .join(ht_ok, on="asset_id", how="inner")
            .join(quality_ok, on="asset_id", how="inner")
            # Drop Silver audit columns — Gold gets its own
            .drop("_sync_timestamp", "_source_db", "_source_table", "_dagster_run_id")
        )

        snapshot = _add_audit_columns(snapshot, "silver_assets_current", context.run_id)

        iceberg_path = f"{CATALOG_NS}.gold_dataset_snapshot_items"
        snapshot.writeTo(iceberg_path).using("iceberg").createOrReplace()

        row_count = snapshot.count()
        context.log.info(f"gold_dataset_snapshot_items: wrote {row_count} rows")

        return Output(
            {"table": "gold_dataset_snapshot_items", "rows": row_count},
            metadata={
                "rows": dagster.MetadataValue.int(row_count),
                "gold_table": dagster.MetadataValue.text("gold_dataset_snapshot_items"),
            },
        )
    finally:
        spark.stop()


# ---------------------------------------------------------------------------
# Task 12.2 — gold_asset_search_docs
# ---------------------------------------------------------------------------


@asset(
    group_name="gold",
    deps=["silver_assets_current", "silver_asset_tags", "silver_asset_algo_latest"],
    description=(
        "Gold wide table for OpenSearch sync: asset base fields + "
        "tags pivoted to columns + algo_summary. "
        "Produces gold_asset_search_docs."
    ),
)
def gold_asset_search_docs(context: AssetExecutionContext) -> Output[dict]:
    """
    Build a denormalized wide table joining:
      - silver_assets_current (base fields)
      - silver_asset_tags (pivoted: one column per tag key)
      - silver_asset_algo_latest (aggregated into algo_summary JSON)

    This table is the source for OpenSearch bulk indexing.
    """
    spark = _get_spark()
    try:
        from pyspark.sql.functions import (
            col, collect_list, struct, to_json,
            first, concat_ws, count, when, lit,
        )

        # Read Silver tables
        assets = spark.read.table(f"{CATALOG_NS}.silver_assets_current")
        tags = spark.read.table(f"{CATALOG_NS}.silver_asset_tags")
        algo = spark.read.table(f"{CATALOG_NS}.silver_asset_algo_latest")

        # --- Pivot tags: one column per tag_key ---
        # Collect distinct tag keys to pivot dynamically
        tag_keys = [row["tag_key"] for row in tags.select("tag_key").distinct().collect()]

        if tag_keys:
            tags_pivoted = (
                tags
                .groupBy("asset_id")
                .pivot("tag_key", tag_keys)
                .agg(first("tag_value"))
            )
            # Prefix tag columns with "tag_" to avoid name collisions
            for tk in tag_keys:
                tags_pivoted = tags_pivoted.withColumnRenamed(tk, f"tag_{tk}")
        else:
            # No tags — create empty DataFrame with just asset_id
            from pyspark.sql.types import StructType, StructField, StringType
            tags_pivoted = spark.createDataFrame(
                [], StructType([StructField("asset_id", StringType())])
            )

        # --- Aggregate algo results into algo_summary JSON ---
        # Produce a summary like {"ok": 2, "failed": 1, "running": 0}
        algo_summary = (
            algo
            .groupBy("asset_id")
            .agg(
                count(when(col("status") == "ok", True)).alias("algo_ok"),
                count(when(col("status") == "failed", True)).alias("algo_failed"),
                count(when(col("status") == "running", True)).alias("algo_running"),
                count(when(col("status") == "pending", True)).alias("algo_pending"),
                count(when(col("status") == "blocked", True)).alias("algo_blocked"),
            )
            .withColumn(
                "algo_summary",
                to_json(
                    struct("algo_ok", "algo_failed", "algo_running",
                           "algo_pending", "algo_blocked")
                ),
            )
            .select("asset_id", "algo_summary")
        )

        # --- Join everything ---
        # Drop Silver audit columns from base assets before join
        base = assets.drop(
            "_sync_timestamp", "_source_db", "_source_table", "_dagster_run_id"
        )

        gold = (
            base
            .join(tags_pivoted, on="asset_id", how="left")
            .join(algo_summary, on="asset_id", how="left")
        )

        gold = _add_audit_columns(gold, "silver_assets_current", context.run_id)

        iceberg_path = f"{CATALOG_NS}.gold_asset_search_docs"
        gold.writeTo(iceberg_path).using("iceberg").createOrReplace()

        row_count = gold.count()
        context.log.info(f"gold_asset_search_docs: wrote {row_count} rows")

        return Output(
            {"table": "gold_asset_search_docs", "rows": row_count},
            metadata={
                "rows": dagster.MetadataValue.int(row_count),
                "gold_table": dagster.MetadataValue.text("gold_asset_search_docs"),
            },
        )
    finally:
        spark.stop()


# ---------------------------------------------------------------------------
# Task 12.3 + 12.4 — Data reconciliation
# ---------------------------------------------------------------------------


def _get_pg_connection():
    """Create a psycopg2 connection to Postgres."""
    import psycopg2
    return psycopg2.connect(
        host=PG_HOST, port=int(PG_PORT),
        user=PG_USER, password=PG_PASSWORD,
        dbname=PG_DATABASE,
    )


def _ensure_reconciliation_table():
    """
    Create the sync_reconciliation table if it doesn't exist.
    Stores reconciliation results for each run.
    """
    conn = _get_pg_connection()
    try:
        with conn.cursor() as cur:
            cur.execute("""
                CREATE TABLE IF NOT EXISTS sync_reconciliation (
                    id              SERIAL PRIMARY KEY,
                    dagster_run_id  TEXT NOT NULL,
                    checked_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
                    pg_total_count  BIGINT NOT NULL,
                    iceberg_total_count BIGINT NOT NULL,
                    count_diff_pct  DOUBLE PRECISION NOT NULL,
                    pg_status_dist  JSONB NOT NULL DEFAULT '{}',
                    iceberg_status_dist JSONB NOT NULL DEFAULT '{}',
                    status_diff     JSONB NOT NULL DEFAULT '{}',
                    is_alert        BOOLEAN NOT NULL DEFAULT FALSE,
                    details         JSONB NOT NULL DEFAULT '{}'
                );
                CREATE INDEX IF NOT EXISTS idx_sync_recon_checked
                    ON sync_reconciliation (checked_at DESC);
            """)
        conn.commit()
    finally:
        conn.close()


def _get_pg_counts():
    """
    Query Postgres for total active asset count and status distribution.
    Returns (total_count, {status: count}).
    """
    conn = _get_pg_connection()
    try:
        with conn.cursor() as cur:
            # Total count of non-deleted assets
            cur.execute("SELECT count(*) FROM assets WHERE is_deleted = FALSE")
            total = cur.fetchone()[0]

            # Status distribution
            cur.execute("""
                SELECT status, count(*)
                FROM assets
                WHERE is_deleted = FALSE
                GROUP BY status
            """)
            status_dist = {row[0]: row[1] for row in cur.fetchall()}

        return total, status_dist
    finally:
        conn.close()


def _get_iceberg_counts(spark):
    """
    Query Iceberg silver_assets_current for total count and status distribution.
    Returns (total_count, {status: count}).
    """
    from pyspark.sql.functions import col, count as spark_count

    silver = spark.read.table(f"{CATALOG_NS}.silver_assets_current")
    total = silver.count()

    # Status distribution
    status_rows = (
        silver
        .groupBy("status")
        .agg(spark_count("*").alias("cnt"))
        .collect()
    )
    status_dist = {row["status"]: row["cnt"] for row in status_rows}

    return total, status_dist


def _compute_status_diff(pg_dist: dict, iceberg_dist: dict) -> dict:
    """
    Compute per-status count differences between Postgres and Iceberg.
    Returns {status: {"pg": N, "iceberg": M, "diff": N-M}}.
    """
    all_statuses = set(pg_dist.keys()) | set(iceberg_dist.keys())
    diff = {}
    for s in sorted(all_statuses):
        pg_val = pg_dist.get(s, 0)
        ice_val = iceberg_dist.get(s, 0)
        diff[s] = {"pg": pg_val, "iceberg": ice_val, "diff": pg_val - ice_val}
    return diff


def _write_reconciliation_result(
    dagster_run_id: str,
    pg_total: int,
    iceberg_total: int,
    count_diff_pct: float,
    pg_status_dist: dict,
    iceberg_status_dist: dict,
    status_diff: dict,
    is_alert: bool,
):
    """Write reconciliation result to sync_reconciliation Postgres table."""
    conn = _get_pg_connection()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO sync_reconciliation
                    (dagster_run_id, pg_total_count, iceberg_total_count,
                     count_diff_pct, pg_status_dist, iceberg_status_dist,
                     status_diff, is_alert, details)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    dagster_run_id,
                    pg_total,
                    iceberg_total,
                    count_diff_pct,
                    json.dumps(pg_status_dist),
                    json.dumps(iceberg_status_dist),
                    json.dumps(status_diff),
                    is_alert,
                    json.dumps({
                        "threshold": RECONCILIATION_THRESHOLD,
                        "checked_at": datetime.now(timezone.utc).isoformat(),
                    }),
                ),
            )
        conn.commit()
    finally:
        conn.close()


@asset(
    group_name="gold",
    deps=["gold_dataset_snapshot_items", "gold_asset_search_docs"],
    description=(
        "Data reconciliation: compare Postgres active asset count vs "
        "Iceberg silver_assets_current count and status distribution. "
        "Writes results to sync_reconciliation table. "
        "Alerts when difference > 0.1%."
    ),
)
def data_reconciliation(context: AssetExecutionContext) -> Output[dict]:
    """
    After Gold tables are written, run reconciliation:
    1. Query Postgres: SELECT count(*) FROM assets WHERE is_deleted=FALSE
    2. Query Iceberg: silver_assets_current count
    3. Compare totals and status distributions
    4. Write results to sync_reconciliation table
    5. Alert if difference > 0.1%
    """
    spark = _get_spark()
    try:
        # Ensure reconciliation table exists
        _ensure_reconciliation_table()

        # Get counts from both sources
        pg_total, pg_status_dist = _get_pg_counts()
        iceberg_total, iceberg_status_dist = _get_iceberg_counts(spark)

        context.log.info(
            f"Reconciliation — Postgres: {pg_total} rows, "
            f"Iceberg: {iceberg_total} rows"
        )

        # Compute difference percentage
        if pg_total == 0 and iceberg_total == 0:
            count_diff_pct = 0.0
        elif pg_total == 0:
            count_diff_pct = 1.0  # 100% difference
        else:
            count_diff_pct = abs(pg_total - iceberg_total) / pg_total

        # Compute status distribution diff
        status_diff = _compute_status_diff(pg_status_dist, iceberg_status_dist)

        # Determine if alert should fire
        is_alert = count_diff_pct > RECONCILIATION_THRESHOLD

        if is_alert:
            context.log.warning(
                f"RECONCILIATION ALERT: count difference {count_diff_pct:.4%} "
                f"exceeds threshold {RECONCILIATION_THRESHOLD:.4%}. "
                f"Postgres={pg_total}, Iceberg={iceberg_total}"
            )
        else:
            context.log.info(
                f"Reconciliation OK: difference {count_diff_pct:.4%} "
                f"within threshold {RECONCILIATION_THRESHOLD:.4%}"
            )

        # Write results to Postgres
        _write_reconciliation_result(
            dagster_run_id=context.run_id,
            pg_total=pg_total,
            iceberg_total=iceberg_total,
            count_diff_pct=count_diff_pct,
            pg_status_dist=pg_status_dist,
            iceberg_status_dist=iceberg_status_dist,
            status_diff=status_diff,
            is_alert=is_alert,
        )

        result = {
            "pg_total": pg_total,
            "iceberg_total": iceberg_total,
            "count_diff_pct": count_diff_pct,
            "is_alert": is_alert,
            "status_diff": status_diff,
        }

        return Output(
            result,
            metadata={
                "pg_total": dagster.MetadataValue.int(pg_total),
                "iceberg_total": dagster.MetadataValue.int(iceberg_total),
                "count_diff_pct": dagster.MetadataValue.float(count_diff_pct),
                "is_alert": dagster.MetadataValue.bool(is_alert),
            },
        )
    finally:
        spark.stop()

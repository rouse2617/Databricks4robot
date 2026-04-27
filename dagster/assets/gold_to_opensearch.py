"""
Dagster asset: gold_to_opensearch — incremental sync from
gold_asset_search_docs Iceberg table to OpenSearch bulk API.

Reads rows where _sync_timestamp > last watermark, converts to
OpenSearch bulk upsert (index action, doc_id = asset_id).
"""

import json
import os
from datetime import datetime, timezone

import dagster
from dagster import asset, Output, AssetExecutionContext

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

OPENSEARCH_URL = os.getenv("OPENSEARCH_URL", "http://localhost:9200")
OPENSEARCH_INDEX = os.getenv("OPENSEARCH_INDEX", "assets")
BULK_BATCH_SIZE = int(os.getenv("OS_BULK_BATCH_SIZE", "500"))

ICEBERG_REST_URI = os.getenv("ICEBERG_REST_URI", "http://localhost:8183")
S3_ENDPOINT = os.getenv("S3_ENDPOINT", "http://localhost:9000")
S3_ACCESS_KEY = os.getenv("S3_ACCESS_KEY", "admin")
S3_SECRET_KEY = os.getenv("S3_SECRET_KEY", "password")
S3_REGION = os.getenv("S3_REGION", "us-east-1")

PG_HOST = os.getenv("PG_HOST", "localhost")
PG_PORT = os.getenv("PG_PORT", "5432")
PG_USER = os.getenv("PG_USER", "postgres")
PG_PASSWORD = os.getenv("PG_PASSWORD", "postgres")
PG_DATABASE = os.getenv("PG_DATABASE", "data4cyber")

CATALOG_NS = "rest_catalog.default"
WATERMARK_KEY = "gold_to_opensearch"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _get_spark():
    """Create or retrieve a SparkSession for Iceberg REST Catalog."""
    from pyspark.sql import SparkSession

    return (
        SparkSession.builder
        .appName("gold_to_opensearch")
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


def _get_pg_connection():
    import psycopg2
    return psycopg2.connect(
        host=PG_HOST, port=int(PG_PORT),
        user=PG_USER, password=PG_PASSWORD,
        dbname=PG_DATABASE,
    )


def _read_watermark() -> str | None:
    """Read the last sync watermark from Postgres sync_watermarks table."""
    conn = _get_pg_connection()
    try:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT watermark_value FROM sync_watermarks WHERE watermark_key = %s",
                (WATERMARK_KEY,),
            )
            row = cur.fetchone()
            return row[0] if row else None
    finally:
        conn.close()


def _write_watermark(value: str) -> None:
    """Upsert the sync watermark."""
    conn = _get_pg_connection()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO sync_watermarks (watermark_key, watermark_value, updated_at)
                VALUES (%s, %s, now())
                ON CONFLICT (watermark_key)
                DO UPDATE SET watermark_value = EXCLUDED.watermark_value,
                              updated_at = now()
                """,
                (WATERMARK_KEY, value),
            )
        conn.commit()
    finally:
        conn.close()


def _bulk_upsert(docs: list[dict]) -> tuple[int, int]:
    """
    Send documents to OpenSearch via the _bulk API.
    Returns (success_count, error_count).
    """
    import urllib.request

    if not docs:
        return 0, 0

    lines: list[str] = []
    for doc in docs:
        doc_id = doc.get("asset_id", "")
        action = json.dumps({"index": {"_index": OPENSEARCH_INDEX, "_id": doc_id}})
        lines.append(action)
        lines.append(json.dumps(doc, default=str))

    body = "\n".join(lines) + "\n"

    req = urllib.request.Request(
        f"{OPENSEARCH_URL}/_bulk",
        data=body.encode("utf-8"),
        headers={"Content-Type": "application/x-ndjson"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        result = json.loads(resp.read())

    errors = 0
    if result.get("errors"):
        for item in result.get("items", []):
            idx = item.get("index", {})
            if idx.get("status", 200) >= 300:
                errors += 1

    return len(docs) - errors, errors


def _row_to_doc(row: dict) -> dict:
    """Convert a Spark Row (as dict) to an OpenSearch document."""
    doc: dict = {}
    skip_keys = {"_sync_timestamp", "_source_db", "_source_table", "_dagster_run_id"}
    for k, v in row.items():
        if k in skip_keys:
            continue
        if v is None:
            continue
        # Convert datetime objects to ISO strings
        if isinstance(v, datetime):
            doc[k] = v.isoformat()
        else:
            doc[k] = v
    return doc


# ---------------------------------------------------------------------------
# Dagster Asset
# ---------------------------------------------------------------------------


@asset(
    group_name="opensearch",
    deps=["gold_asset_search_docs"],
    description=(
        "Incremental sync from gold_asset_search_docs Iceberg table "
        "to OpenSearch 'assets' index via bulk API."
    ),
)
def gold_to_opensearch(context: AssetExecutionContext) -> Output[dict]:
    """
    1. Read watermark (last _sync_timestamp synced).
    2. Query gold_asset_search_docs WHERE _sync_timestamp > watermark.
    3. Batch bulk upsert to OpenSearch.
    4. Update watermark.
    """
    spark = _get_spark()
    try:
        from pyspark.sql.functions import col, max as spark_max

        table_path = f"{CATALOG_NS}.gold_asset_search_docs"
        gold = spark.read.table(table_path)

        watermark = _read_watermark()
        if watermark:
            context.log.info(f"Incremental sync from watermark: {watermark}")
            gold = gold.filter(col("_sync_timestamp") > watermark)
        else:
            context.log.info("Full sync (no watermark found)")

        total_rows = gold.count()
        if total_rows == 0:
            context.log.info("No new rows to sync.")
            return Output(
                {"synced": 0, "errors": 0},
                metadata={
                    "synced": dagster.MetadataValue.int(0),
                    "errors": dagster.MetadataValue.int(0),
                },
            )

        context.log.info(f"Syncing {total_rows} rows to OpenSearch ...")

        # Collect rows and batch-send
        rows = [row.asDict() for row in gold.collect()]
        total_ok = 0
        total_err = 0

        for i in range(0, len(rows), BULK_BATCH_SIZE):
            batch = rows[i : i + BULK_BATCH_SIZE]
            docs = [_row_to_doc(r) for r in batch]
            ok, err = _bulk_upsert(docs)
            total_ok += ok
            total_err += err
            context.log.info(
                f"  Batch {i // BULK_BATCH_SIZE + 1}: "
                f"{ok} indexed, {err} errors"
            )

        # Update watermark to max _sync_timestamp in this batch
        max_ts_row = (
            spark.read.table(table_path)
            .agg(spark_max("_sync_timestamp").alias("max_ts"))
            .collect()[0]
        )
        new_watermark = str(max_ts_row["max_ts"])
        _write_watermark(new_watermark)
        context.log.info(f"Watermark updated to {new_watermark}")

        return Output(
            {"synced": total_ok, "errors": total_err, "watermark": new_watermark},
            metadata={
                "synced": dagster.MetadataValue.int(total_ok),
                "errors": dagster.MetadataValue.int(total_err),
                "watermark": dagster.MetadataValue.text(new_watermark),
            },
        )
    finally:
        spark.stop()

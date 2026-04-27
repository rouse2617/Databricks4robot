"""
Dagster assets: Postgres → Bronze Iceberg tables.

Reads Postgres tables (assets, mcap_files, deliveries, delivery_items,
asset_algo_events) incrementally by `updated_at > watermark` and writes
to Bronze Iceberg tables via PySpark + Iceberg REST Catalog.

Bronze tables:
  - bronze_assets
  - bronze_mcap_files
  - bronze_deliveries
  - bronze_delivery_items
  - bronze_asset_events

All Bronze tables include audit columns:
  _sync_timestamp, _source_db, _source_table, _dagster_run_id

JSONB fields (cf_meta, cf_algo, cf_tag, cf_files) are preserved as STRING
columns — no transformation at the Bronze layer.
"""

import os
from datetime import datetime, timezone
from typing import Optional

import dagster
from dagster import asset, Output, AssetExecutionContext

# ---------------------------------------------------------------------------
# Configuration — read from environment with sensible local-dev defaults
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

# ---------------------------------------------------------------------------
# Table definitions — source table → Bronze table + column metadata
# ---------------------------------------------------------------------------

# Columns that contain JSONB in Postgres and must be kept as STRING in Bronze
JSONB_COLUMNS = {"cf_meta", "cf_algo", "cf_tag", "cf_files", "cf_process",
                 "request_summary", "response_json"}

# Tables that have an `updated_at` column for incremental sync
INCREMENTAL_TABLES = {
    "assets": "bronze_assets",
    "mcap_files": "bronze_mcap_files",
    "deliveries": "bronze_deliveries",
}

# Tables without `updated_at` — use `created_at` as watermark
CREATED_AT_TABLES = {
    "delivery_items": "bronze_delivery_items",
    "asset_algo_events": "bronze_asset_events",
}

ALL_TABLES = {**INCREMENTAL_TABLES, **CREATED_AT_TABLES}


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _pg_jdbc_url() -> str:
    """Build JDBC URL for Postgres."""
    return f"jdbc:postgresql://{PG_HOST}:{PG_PORT}/{PG_DATABASE}"


def _pg_jdbc_props() -> dict:
    """JDBC connection properties."""
    return {"user": PG_USER, "password": PG_PASSWORD, "driver": "org.postgresql.Driver"}


def _get_spark():
    """
    Create or retrieve a SparkSession configured for Iceberg REST Catalog
    writing to MinIO (S3-compatible).
    """
    from pyspark.sql import SparkSession

    spark = (
        SparkSession.builder
        .appName("postgres_to_bronze")
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
        .config("spark.sql.extensions", "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions")
        .config("spark.jars.packages",
                "org.apache.iceberg:iceberg-spark-runtime-3.5_2.12:1.5.2,"
                "org.postgresql:postgresql:42.7.3,"
                "org.apache.iceberg:iceberg-aws-bundle:1.5.2")
        .getOrCreate()
    )
    return spark


def _read_watermark(spark, table_name: str) -> Optional[str]:
    """
    Read the last watermark for a given source table from Postgres
    sync_watermarks table. Returns ISO timestamp string or None.
    """
    try:
        df = (
            spark.read.jdbc(
                _pg_jdbc_url(),
                f"(SELECT watermark FROM sync_watermarks WHERE table_name = '{table_name}') AS wm",
                properties=_pg_jdbc_props(),
            )
        )
        rows = df.collect()
        if rows:
            return str(rows[0]["watermark"])
    except Exception:
        # Table may not exist yet on first run
        pass
    return None


def _write_watermark(spark, table_name: str, watermark: str,
                     dagster_run_id: str, synced_rows: int):
    """
    Upsert watermark into Postgres sync_watermarks table.
    Uses a raw JDBC connection for the upsert.
    """
    import psycopg2

    conn = psycopg2.connect(
        host=PG_HOST, port=int(PG_PORT),
        user=PG_USER, password=PG_PASSWORD,
        dbname=PG_DATABASE,
    )
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO sync_watermarks (table_name, watermark, dagster_run_id, synced_rows, updated_at)
                VALUES (%s, %s, %s, %s, now())
                ON CONFLICT (table_name) DO UPDATE SET
                    watermark = EXCLUDED.watermark,
                    dagster_run_id = EXCLUDED.dagster_run_id,
                    synced_rows = EXCLUDED.synced_rows,
                    updated_at = now()
                """,
                (table_name, watermark, dagster_run_id, synced_rows),
            )
        conn.commit()
    finally:
        conn.close()


def _cast_jsonb_to_string(df):
    """
    Cast known JSONB columns to StringType so they are stored as STRING
    in Iceberg Bronze tables (no transformation).
    """
    from pyspark.sql.functions import col
    from pyspark.sql.types import StringType

    for c in df.columns:
        if c in JSONB_COLUMNS:
            df = df.withColumn(c, col(c).cast(StringType()))
    return df


def _add_audit_columns(df, source_table: str, dagster_run_id: str):
    """
    Add audit columns required by all Bronze tables:
      _sync_timestamp, _source_db, _source_table, _dagster_run_id
    """
    from pyspark.sql.functions import lit, current_timestamp

    return (
        df
        .withColumn("_sync_timestamp", current_timestamp())
        .withColumn("_source_db", lit(SOURCE_DB))
        .withColumn("_source_table", lit(source_table))
        .withColumn("_dagster_run_id", lit(dagster_run_id))
    )


def _sync_table(context: AssetExecutionContext, spark,
                source_table: str, bronze_table: str,
                watermark_col: str):
    """
    Core sync logic for a single Postgres table → Bronze Iceberg table.

    1. Read watermark from sync_watermarks
    2. Read incremental rows from Postgres where watermark_col > watermark
    3. Cast JSONB columns to STRING
    4. Add audit columns
    5. Write (append) to Iceberg Bronze table
    6. Update watermark
    """
    dagster_run_id = context.run_id

    # Step 1: Read watermark
    watermark = _read_watermark(spark, source_table)
    if watermark:
        context.log.info(f"[{source_table}] Last watermark: {watermark}")
        query = (
            f"(SELECT * FROM {source_table} "
            f"WHERE {watermark_col} > '{watermark}' "
            f"ORDER BY {watermark_col}) AS incremental"
        )
    else:
        context.log.info(f"[{source_table}] No watermark found — full load")
        query = f"(SELECT * FROM {source_table} ORDER BY {watermark_col}) AS full_load"

    # Step 2: Read from Postgres
    df = spark.read.jdbc(_pg_jdbc_url(), query, properties=_pg_jdbc_props())
    row_count = df.count()

    if row_count == 0:
        context.log.info(f"[{source_table}] No new rows to sync")
        return {"table": bronze_table, "rows_synced": 0, "watermark": watermark}

    context.log.info(f"[{source_table}] Found {row_count} new rows")

    # Step 3: Cast JSONB → STRING
    df = _cast_jsonb_to_string(df)

    # Step 4: Add audit columns
    df = _add_audit_columns(df, source_table, dagster_run_id)

    # Step 5: Write to Iceberg Bronze table (append mode, create if not exists)
    iceberg_table_path = f"rest_catalog.default.{bronze_table}"
    df.writeTo(iceberg_table_path).using("iceberg").createOrReplace()

    # Step 6: Compute new watermark (max of watermark_col in this batch)
    from pyspark.sql.functions import max as spark_max
    new_watermark = df.agg(spark_max(watermark_col)).collect()[0][0]
    new_watermark_str = str(new_watermark) if new_watermark else watermark

    _write_watermark(spark, source_table, new_watermark_str,
                     dagster_run_id, row_count)
    context.log.info(
        f"[{source_table}] Synced {row_count} rows → {bronze_table}, "
        f"new watermark: {new_watermark_str}"
    )

    return {
        "table": bronze_table,
        "rows_synced": row_count,
        "watermark": new_watermark_str,
    }


# ---------------------------------------------------------------------------
# Dagster Assets — one per Bronze table
# ---------------------------------------------------------------------------


@asset(group_name="bronze", description="Sync Postgres assets → bronze_assets (incremental by updated_at)")
def bronze_assets(context: AssetExecutionContext) -> Output[dict]:
    """
    Incrementally sync the Postgres `assets` table to the Bronze Iceberg
    table `bronze_assets`. JSONB columns (cf_meta, cf_algo, cf_tag, cf_files)
    are preserved as STRING.
    """
    spark = _get_spark()
    try:
        result = _sync_table(context, spark, "assets", "bronze_assets", "updated_at")
        return Output(result, metadata={
            "rows_synced": dagster.MetadataValue.int(result["rows_synced"]),
            "bronze_table": dagster.MetadataValue.text(result["table"]),
        })
    finally:
        spark.stop()


@asset(group_name="bronze", description="Sync Postgres mcap_files → bronze_mcap_files (incremental by updated_at)")
def bronze_mcap_files(context: AssetExecutionContext) -> Output[dict]:
    """
    Incrementally sync the Postgres `mcap_files` table to the Bronze Iceberg
    table `bronze_mcap_files`.
    """
    spark = _get_spark()
    try:
        result = _sync_table(context, spark, "mcap_files", "bronze_mcap_files", "updated_at")
        return Output(result, metadata={
            "rows_synced": dagster.MetadataValue.int(result["rows_synced"]),
            "bronze_table": dagster.MetadataValue.text(result["table"]),
        })
    finally:
        spark.stop()


@asset(group_name="bronze", description="Sync Postgres deliveries → bronze_deliveries (incremental by updated_at)")
def bronze_deliveries(context: AssetExecutionContext) -> Output[dict]:
    """
    Incrementally sync the Postgres `deliveries` table to the Bronze Iceberg
    table `bronze_deliveries`.
    """
    spark = _get_spark()
    try:
        result = _sync_table(context, spark, "deliveries", "bronze_deliveries", "updated_at")
        return Output(result, metadata={
            "rows_synced": dagster.MetadataValue.int(result["rows_synced"]),
            "bronze_table": dagster.MetadataValue.text(result["table"]),
        })
    finally:
        spark.stop()


@asset(group_name="bronze", description="Sync Postgres delivery_items → bronze_delivery_items (incremental by created_at)")
def bronze_delivery_items(context: AssetExecutionContext) -> Output[dict]:
    """
    Incrementally sync the Postgres `delivery_items` table to the Bronze
    Iceberg table `bronze_delivery_items`. Uses `created_at` as watermark
    since this table has no `updated_at`.
    """
    spark = _get_spark()
    try:
        result = _sync_table(context, spark, "delivery_items", "bronze_delivery_items", "created_at")
        return Output(result, metadata={
            "rows_synced": dagster.MetadataValue.int(result["rows_synced"]),
            "bronze_table": dagster.MetadataValue.text(result["table"]),
        })
    finally:
        spark.stop()


@asset(group_name="bronze", description="Sync Postgres asset_algo_events → bronze_asset_events (incremental by created_at)")
def bronze_asset_events(context: AssetExecutionContext) -> Output[dict]:
    """
    Incrementally sync the Postgres `asset_algo_events` table to the Bronze
    Iceberg table `bronze_asset_events`. Uses `created_at` as watermark.
    """
    spark = _get_spark()
    try:
        result = _sync_table(context, spark, "asset_algo_events", "bronze_asset_events", "created_at")
        return Output(result, metadata={
            "rows_synced": dagster.MetadataValue.int(result["rows_synced"]),
            "bronze_table": dagster.MetadataValue.text(result["table"]),
        })
    finally:
        spark.stop()

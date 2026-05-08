"""
Dagster assets: Bronze → Silver Iceberg tables.

Reads Bronze Iceberg tables and produces Silver "current state" tables
with JSONB columns expanded into columnar fields.

Silver tables:
  - silver_assets_current      (cf_meta/cf_files expanded to columns)
  - silver_asset_tags           (cf_tag exploded to multi-row)
  - silver_asset_algo_latest    (cf_algo exploded to multi-row)
  - silver_mcap_files_current   (latest version from bronze_mcap_files)
  - silver_deliveries_current   (latest version from bronze_deliveries)

All Silver tables include audit columns:
  _sync_timestamp, _source_db, _source_table, _dagster_run_id
"""

import os

import dagster
from dagster import asset, Output, AssetExecutionContext

# ---------------------------------------------------------------------------
# Configuration — same env vars as postgres_to_bronze.py
# ---------------------------------------------------------------------------

ICEBERG_REST_URI = os.getenv("ICEBERG_REST_URI", "http://localhost:8183")
S3_ENDPOINT = os.getenv("S3_ENDPOINT", "http://localhost:9000")
S3_ACCESS_KEY = os.getenv("S3_ACCESS_KEY", "admin")
S3_SECRET_KEY = os.getenv("S3_SECRET_KEY", "password")
S3_REGION = os.getenv("S3_REGION", "us-east-1")

SOURCE_DB = "data4cyber"

# Iceberg catalog / namespace prefix
CATALOG_NS = "rest_catalog.default"

# ---------------------------------------------------------------------------
# Spark session helper (shared config with Bronze layer)
# ---------------------------------------------------------------------------


def _get_spark():
    """
    Create or retrieve a SparkSession configured for Iceberg REST Catalog
    writing to MinIO (S3-compatible).
    """
    from pyspark.sql import SparkSession

    spark = (
        SparkSession.builder
        .appName("bronze_to_silver")
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
                "org.apache.iceberg:iceberg-aws-bundle:1.5.2")
        .getOrCreate()
    )
    return spark


# ---------------------------------------------------------------------------
# Audit column helper
# ---------------------------------------------------------------------------


def _add_audit_columns(df, source_table: str, dagster_run_id: str):
    """
    Add audit columns required by all Silver tables:
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


# ---------------------------------------------------------------------------
# Helper: deduplicate Bronze rows to keep latest version per primary key
# ---------------------------------------------------------------------------


def _latest_version(df, id_col: str):
    """
    Bronze tables may contain multiple versions of the same row (append mode).
    Keep only the row with the highest ``version`` for each ``id_col``.
    Falls back to ``_sync_timestamp`` if ``version`` column is absent.
    """
    from pyspark.sql.window import Window
    from pyspark.sql.functions import row_number, col, desc

    order_col = "version" if "version" in df.columns else "_sync_timestamp"
    w = Window.partitionBy(id_col).orderBy(desc(order_col))
    return (
        df
        .withColumn("_rn", row_number().over(w))
        .filter("_rn = 1")
        .drop("_rn")
    )


# ---------------------------------------------------------------------------
# Dagster Assets — Silver layer
# ---------------------------------------------------------------------------


@asset(
    group_name="silver",
    deps=["bronze_assets"],
    description=(
        "Expand cf_meta JSONB → columnar fields (duration_ms, reviewer, owner, "
        "env, task, delivery_count, etc.), expand cf_files. "
        "Produces silver_assets_current."
    ),
)
def silver_assets_current(context: AssetExecutionContext) -> Output[dict]:
    """
    Read ``bronze_assets``, deduplicate to latest version per asset_id,
    then expand cf_meta and cf_files JSONB strings into typed columns.

    Target columns from cf_meta:
      duration_ms (duration_sec × 1000), reviewer, owner, asset_type,
      city (from env), scenario_type (from task), delivery_count

    Target columns from cf_files:
      raw_mcap (first file URI)
    """
    spark = _get_spark()
    try:
        from pyspark.sql.functions import (
            col, get_json_object, lit, coalesce,
        )
        from pyspark.sql.types import DoubleType, IntegerType

        bronze = spark.read.table(f"{CATALOG_NS}.bronze_assets")
        latest = _latest_version(bronze, "asset_id")

        silver = (
            latest
            # --- cf_meta expansion ---
            .withColumn(
                "duration_ms",
                get_json_object(col("cf_meta"), "$.duration_sec")
                .cast(DoubleType()) * 1000,
            )
            .withColumn(
                "reviewer",
                get_json_object(col("cf_meta"), "$.reviewer"),
            )
            .withColumn(
                "owner",
                get_json_object(col("cf_meta"), "$.owner"),
            )
            .withColumn(
                "asset_type",
                get_json_object(col("cf_meta"), "$.type"),
            )
            .withColumn(
                "city",
                get_json_object(col("cf_meta"), "$.env"),
            )
            .withColumn(
                "scenario_type",
                coalesce(
                    get_json_object(col("cf_meta"), "$.task"),
                    get_json_object(col("cf_meta"), "$.env"),
                ),
            )
            .withColumn(
                "delivery_count",
                get_json_object(col("cf_meta"), "$.delivery_count")
                .cast(IntegerType()),
            )
            # --- cf_files expansion ---
            .withColumn(
                "raw_mcap",
                get_json_object(col("cf_files"), "$.raw_mcap"),
            )
            # Drop original JSONB string columns (kept in Bronze)
            .drop("cf_meta", "cf_algo", "cf_tag", "cf_files")
            # Drop Bronze audit columns — Silver gets its own
            .drop("_sync_timestamp", "_source_db", "_source_table", "_dagster_run_id")
        )

        silver = _add_audit_columns(silver, "bronze_assets", context.run_id)

        iceberg_path = f"{CATALOG_NS}.silver_assets_current"
        silver.writeTo(iceberg_path).using("iceberg").createOrReplace()

        row_count = silver.count()
        context.log.info(f"silver_assets_current: wrote {row_count} rows")

        return Output(
            {"table": "silver_assets_current", "rows": row_count},
            metadata={
                "rows": dagster.MetadataValue.int(row_count),
                "silver_table": dagster.MetadataValue.text("silver_assets_current"),
            },
        )
    finally:
        spark.stop()


@asset(
    group_name="silver",
    deps=["bronze_assets"],
    description=(
        "Parse cf_tag JSONB → multi-row table "
        "(asset_id, tag_key, tag_value, updated_at). "
        "Produces silver_asset_tags."
    ),
)
def silver_asset_tags(context: AssetExecutionContext) -> Output[dict]:
    """
    Read ``bronze_assets``, deduplicate to latest version, then explode
    the ``cf_tag`` JSONB object into one row per (asset_id, tag_key, tag_value).

    cf_tag example: ``{"priority":"high","quality":"excellent","scene":"indoor"}``
    Result rows:
      (asset_id, "priority", "high", updated_at)
      (asset_id, "quality", "excellent", updated_at)
      (asset_id, "scene", "indoor", updated_at)
    """
    spark = _get_spark()
    try:
        from pyspark.sql.functions import (
            col, from_json, explode, map_keys, map_values,
            create_map, arrays_zip, lit,
        )
        from pyspark.sql.types import MapType, StringType

        bronze = spark.read.table(f"{CATALOG_NS}.bronze_assets")
        latest = _latest_version(bronze, "asset_id")

        # Parse cf_tag STRING → Map<String, String>
        tag_schema = MapType(StringType(), StringType())
        parsed = latest.withColumn("tag_map", from_json(col("cf_tag"), tag_schema))

        # Explode map into rows
        exploded = (
            parsed
            .select(
                col("asset_id"),
                explode(col("tag_map")).alias("tag_key", "tag_value"),
                col("updated_at"),
            )
        )

        silver = _add_audit_columns(exploded, "bronze_assets", context.run_id)

        iceberg_path = f"{CATALOG_NS}.silver_asset_tags"
        silver.writeTo(iceberg_path).using("iceberg").createOrReplace()

        row_count = silver.count()
        context.log.info(f"silver_asset_tags: wrote {row_count} rows")

        return Output(
            {"table": "silver_asset_tags", "rows": row_count},
            metadata={
                "rows": dagster.MetadataValue.int(row_count),
                "silver_table": dagster.MetadataValue.text("silver_asset_tags"),
            },
        )
    finally:
        spark.stop()


@asset(
    group_name="silver",
    deps=["bronze_assets"],
    description=(
        "Parse cf_algo JSONB → multi-row table "
        "(asset_id, algo_name, algo_version, status, run_id, output_uri, "
        "started_at, finished_at). Produces silver_asset_algo_latest."
    ),
)
def silver_asset_algo_latest(context: AssetExecutionContext) -> Output[dict]:
    """
    Read ``bronze_assets``, deduplicate to latest version, then parse the
    flat ``cf_algo`` JSONB into structured rows.

    cf_algo stores keys like ``"hand_tracking@1.2.0:status": "ok"`` — a flat
    map where each key is ``<algo_name>@<version>:<field>``.

    This asset pivots those flat keys into one row per algorithm with columns:
      asset_id, algo_name, algo_version, status, run_id, output_uri,
      started_at, finished_at
    """
    spark = _get_spark()
    try:
        from pyspark.sql.functions import (
            col, from_json, explode, udf, lit,
        )
        from pyspark.sql.types import (
            MapType, StringType, StructType, StructField, ArrayType,
        )

        bronze = spark.read.table(f"{CATALOG_NS}.bronze_assets")
        latest = _latest_version(bronze, "asset_id")

        # Parse cf_algo STRING → Map<String, String>
        algo_schema = MapType(StringType(), StringType())
        parsed = latest.withColumn(
            "algo_map", from_json(col("cf_algo"), algo_schema)
        )

        # UDF: pivot flat map → list of structs
        algo_row_schema = ArrayType(
            StructType([
                StructField("algo_name", StringType()),
                StructField("algo_version", StringType()),
                StructField("status", StringType()),
                StructField("run_id", StringType()),
                StructField("output_uri", StringType()),
                StructField("started_at", StringType()),
                StructField("finished_at", StringType()),
                StructField("method", StringType()),
            ])
        )

        @udf(algo_row_schema)
        def pivot_algo(algo_map):
            """
            Convert flat map like:
              {"hand_tracking@1.2.0:status": "ok",
               "hand_tracking@1.2.0:run_id": "dagster-run-001", ...}
            into a list of dicts grouped by algo key (name@version).
            """
            if not algo_map:
                return []

            grouped: dict = {}
            for k, v in algo_map.items():
                # Expected format: <algo_name>@<version>:<field>
                if ":" not in k:
                    continue
                algo_key_part, field = k.rsplit(":", 1)
                if algo_key_part not in grouped:
                    grouped[algo_key_part] = {}
                grouped[algo_key_part][field] = v

            rows = []
            for algo_key, fields in grouped.items():
                if "@" in algo_key:
                    name, version = algo_key.split("@", 1)
                else:
                    name, version = algo_key, ""
                rows.append({
                    "algo_name": name,
                    "algo_version": version,
                    "status": fields.get("status"),
                    "run_id": fields.get("run_id"),
                    "output_uri": fields.get("output_uri"),
                    "started_at": fields.get("started_at"),
                    "finished_at": fields.get("finished_at"),
                    "method": fields.get("method"),
                })
            return rows

        pivoted = parsed.withColumn("algo_rows", pivot_algo(col("algo_map")))

        exploded = (
            pivoted
            .select(
                col("asset_id"),
                explode(col("algo_rows")).alias("algo"),
            )
            .select(
                col("asset_id"),
                col("algo.algo_name").alias("algo_name"),
                col("algo.algo_version").alias("algo_version"),
                col("algo.status").alias("status"),
                col("algo.run_id").alias("run_id"),
                col("algo.output_uri").alias("output_uri"),
                col("algo.started_at").alias("started_at"),
                col("algo.finished_at").alias("finished_at"),
                col("algo.method").alias("method"),
            )
        )

        silver = _add_audit_columns(exploded, "bronze_assets", context.run_id)

        iceberg_path = f"{CATALOG_NS}.silver_asset_algo_latest"
        silver.writeTo(iceberg_path).using("iceberg").createOrReplace()

        row_count = silver.count()
        context.log.info(f"silver_asset_algo_latest: wrote {row_count} rows")

        return Output(
            {"table": "silver_asset_algo_latest", "rows": row_count},
            metadata={
                "rows": dagster.MetadataValue.int(row_count),
                "silver_table": dagster.MetadataValue.text("silver_asset_algo_latest"),
            },
        )
    finally:
        spark.stop()


@asset(
    group_name="silver",
    deps=["bronze_mcap_files"],
    description=(
        "Take latest version from bronze_mcap_files. "
        "Produces silver_mcap_files_current."
    ),
)
def silver_mcap_files_current(context: AssetExecutionContext) -> Output[dict]:
    """
    Read ``bronze_mcap_files``, deduplicate to latest version per
    mcap_file_id, drop Bronze audit columns, and write to Silver.
    """
    spark = _get_spark()
    try:
        bronze = spark.read.table(f"{CATALOG_NS}.bronze_mcap_files")
        latest = _latest_version(bronze, "mcap_file_id")

        silver = (
            latest
            .drop("_sync_timestamp", "_source_db", "_source_table", "_dagster_run_id")
        )
        silver = _add_audit_columns(silver, "bronze_mcap_files", context.run_id)

        iceberg_path = f"{CATALOG_NS}.silver_mcap_files_current"
        silver.writeTo(iceberg_path).using("iceberg").createOrReplace()

        row_count = silver.count()
        context.log.info(f"silver_mcap_files_current: wrote {row_count} rows")

        return Output(
            {"table": "silver_mcap_files_current", "rows": row_count},
            metadata={
                "rows": dagster.MetadataValue.int(row_count),
                "silver_table": dagster.MetadataValue.text("silver_mcap_files_current"),
            },
        )
    finally:
        spark.stop()


@asset(
    group_name="silver",
    deps=["bronze_deliveries"],
    description=(
        "Take latest version from bronze_deliveries. "
        "Produces silver_deliveries_current."
    ),
)
def silver_deliveries_current(context: AssetExecutionContext) -> Output[dict]:
    """
    Read ``bronze_deliveries``, deduplicate to latest version per
    delivery_id, drop Bronze audit columns, and write to Silver.
    """
    spark = _get_spark()
    try:
        bronze = spark.read.table(f"{CATALOG_NS}.bronze_deliveries")
        latest = _latest_version(bronze, "delivery_id")

        silver = (
            latest
            .drop("_sync_timestamp", "_source_db", "_source_table", "_dagster_run_id")
        )
        silver = _add_audit_columns(silver, "bronze_deliveries", context.run_id)

        iceberg_path = f"{CATALOG_NS}.silver_deliveries_current"
        silver.writeTo(iceberg_path).using("iceberg").createOrReplace()

        row_count = silver.count()
        context.log.info(f"silver_deliveries_current: wrote {row_count} rows")

        return Output(
            {"table": "silver_deliveries_current", "rows": row_count},
            metadata={
                "rows": dagster.MetadataValue.int(row_count),
                "silver_table": dagster.MetadataValue.text("silver_deliveries_current"),
            },
        )
    finally:
        spark.stop()

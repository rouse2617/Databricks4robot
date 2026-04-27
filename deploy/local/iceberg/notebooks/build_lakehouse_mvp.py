import json
import os
from datetime import date, datetime
from pathlib import Path

from pyspark.sql import SparkSession


POSTGRES_URL = os.getenv("POSTGRES_URL", "jdbc:postgresql://postgres:5432/data4cyber")
POSTGRES_PROPERTIES = {
    "user": "postgres",
    "password": "postgres",
    "driver": "org.postgresql.Driver",
}
REPORT_PATH = Path("/home/iceberg/notebooks/notebooks/lakehouse_report.json")


spark = (
    SparkSession.builder.appName("build-lakehouse-mvp")
    .config("spark.jars.packages", "org.postgresql:postgresql:42.7.4")
    .config("spark.sql.catalog.demo", "org.apache.iceberg.spark.SparkCatalog")
    .config("spark.sql.catalog.demo.type", "rest")
    .config("spark.sql.catalog.demo.uri", "http://iceberg-rest:8181")
    .config("spark.sql.catalog.demo.warehouse", "s3://warehouse/")
    .config("spark.sql.catalog.demo.io-impl", "org.apache.iceberg.aws.s3.S3FileIO")
    .config("spark.sql.catalog.demo.s3.endpoint", "http://minio:9000")
    .config("spark.sql.catalog.demo.s3.path-style-access", "true")
    .getOrCreate()
)


def read_postgres_query(query: str):
    return (
        spark.read.format("jdbc")
        .option("url", POSTGRES_URL)
        .option("query", query)
        .options(**POSTGRES_PROPERTIES)
        .load()
    )


def write_iceberg_table(table_name: str, query: str):
    target = f"demo.robot.{table_name}"
    df = read_postgres_query(query)
    df.writeTo(target).using("iceberg").createOrReplace()
    row_count = df.count()
    print(f"Wrote {row_count} rows to {target}")
    return row_count


def json_default(value):
    if isinstance(value, datetime | date):
        return value.isoformat()
    return str(value)


def collect_dicts(query: str, limit: int | None = None):
    df = spark.sql(query)
    if limit is not None:
        df = df.limit(limit)
    return [row.asDict(recursive=True) for row in df.collect()]


spark.sql("CREATE NAMESPACE IF NOT EXISTS demo.robot")


postgres_to_iceberg_tables = {
    "bronze_asset_algo_events": """
        SELECT
          event_id::text,
          asset_id::text,
          algo_key,
          prev_status,
          new_status,
          run_id,
          reason,
          created_at
        FROM asset_algo_events
    """,
    "bronze_delivery_items": """
        SELECT
          delivery_id::text,
          asset_id::text,
          created_at
        FROM delivery_items
    """,
    "silver_mcap_files_current": """
        SELECT
          mcap_file_id::text,
          raw_hash_md5,
          is_deleted,
          cf_meta::text AS cf_meta_json,
          cf_process::text AS cf_process_json,
          created_at,
          updated_at,
          version
        FROM mcap_files
    """,
    "silver_assets_current": """
        SELECT
          asset_id::text,
          mcap_file_id::text,
          start_timestamp_ns,
          end_timestamp_ns,
          status,
          is_deleted,
          segment_locator,
          cf_meta ->> 'duration_sec' AS duration_sec,
          cf_meta ->> 'reviewer' AS reviewer,
          cf_meta ->> 'owner' AS owner,
          cf_meta ->> 'type' AS asset_type,
          cf_meta ->> 'env' AS env,
          cf_meta ->> 'task' AS task,
          cf_meta::text AS cf_meta_json,
          cf_files::text AS cf_files_json,
          created_at,
          updated_at,
          version
        FROM assets
    """,
    "silver_deliveries_current": """
        SELECT
          delivery_id::text,
          customer_id,
          status,
          delivered_at,
          is_deleted,
          cf_meta ->> 'manifest_uri' AS manifest_uri,
          cf_meta ->> 'contract_id' AS contract_id,
          cf_meta ->> 'asset_count' AS asset_count,
          cf_meta ->> 'owner' AS owner,
          cf_meta::text AS cf_meta_json,
          created_at,
          updated_at,
          version
        FROM deliveries
    """,
    "silver_asset_algo_latest": """
        SELECT
          a.asset_id::text,
          regexp_replace(kv.key, ':status$', '') AS algo_key,
          kv.value AS status,
          NULLIF(a.cf_algo ->> (regexp_replace(kv.key, ':status$', '') || ':started_at'), '')::timestamptz AS started_at,
          NULLIF(a.cf_algo ->> (regexp_replace(kv.key, ':status$', '') || ':finished_at'), '')::timestamptz AS finished_at,
          a.cf_algo ->> (regexp_replace(kv.key, ':status$', '') || ':method') AS method,
          a.cf_algo ->> (regexp_replace(kv.key, ':status$', '') || ':run_id') AS run_id,
          a.cf_algo ->> (regexp_replace(kv.key, ':status$', '') || ':output_uri') AS output_uri,
          a.cf_algo ->> (regexp_replace(kv.key, ':status$', '') || ':reason') AS reason,
          a.updated_at
        FROM assets a
        CROSS JOIN LATERAL jsonb_each_text(a.cf_algo) kv
        WHERE kv.key LIKE '%:status'
    """,
    "silver_asset_tags": """
        SELECT
          a.asset_id::text,
          kv.key AS tag_key,
          kv.value AS tag_value,
          'cf_tag' AS source,
          a.updated_at
        FROM assets a
        CROSS JOIN LATERAL jsonb_each_text(a.cf_tag) kv
    """,
}


for table_name, query in postgres_to_iceberg_tables.items():
    write_iceberg_table(table_name, query)


spark.sql(
    """
    CREATE OR REPLACE TABLE demo.robot.gold_dataset_snapshot_items
    USING iceberg
    AS
    SELECT
      'mvp_hand_tracking_quality_v1' AS dataset_snapshot_id,
      a.asset_id,
      a.mcap_file_id,
      a.segment_locator,
      a.env,
      a.task,
      algo.algo_key,
      algo.status AS algo_status,
      algo.output_uri,
      tag.tag_value AS quality,
      current_timestamp() AS created_at
    FROM demo.robot.silver_assets_current a
    JOIN demo.robot.silver_asset_algo_latest algo
      ON a.asset_id = algo.asset_id
    JOIN demo.robot.silver_asset_tags tag
      ON a.asset_id = tag.asset_id
    WHERE algo.algo_key = 'hand_tracking@1.2.0'
      AND algo.status = 'ok'
      AND tag.tag_key = 'quality'
      AND tag.tag_value IN ('good', 'excellent')
      AND a.is_deleted = false
    """
)


print("Lakehouse MVP tables:")
spark.sql("SHOW TABLES IN demo.robot").show(truncate=False)

print("Algo status distribution:")
spark.sql(
    """
    SELECT algo_key, status, count(*) AS asset_count
    FROM demo.robot.silver_asset_algo_latest
    GROUP BY algo_key, status
    ORDER BY algo_key, status
    """
).show(100, truncate=False)

print("Tag distribution:")
spark.sql(
    """
    SELECT tag_key, tag_value, count(*) AS asset_count
    FROM demo.robot.silver_asset_tags
    GROUP BY tag_key, tag_value
    ORDER BY tag_key, tag_value
    """
).show(100, truncate=False)

print("Gold dataset snapshot sample:")
spark.sql(
    """
    SELECT *
    FROM demo.robot.gold_dataset_snapshot_items
    ORDER BY asset_id
    LIMIT 20
    """
).show(truncate=False)

print("Gold dataset snapshot count:")
gold_snapshot_count = collect_dicts(
    """
    SELECT dataset_snapshot_id, count(*) AS asset_count
    FROM demo.robot.gold_dataset_snapshot_items
    GROUP BY dataset_snapshot_id
    """
)
spark.sql(
    """
    SELECT dataset_snapshot_id, count(*) AS asset_count
    FROM demo.robot.gold_dataset_snapshot_items
    GROUP BY dataset_snapshot_id
    """
).show(truncate=False)


report = {
    "generated_at": datetime.utcnow().isoformat() + "Z",
    "sync_mode": "mvp_batch_sync",
    "sync_note": "MVP uses a Spark batch job to emulate the PG -> Iceberg CDC path. Replace with RisingWave/Debezium for continuous CDC.",
    "tables": collect_dicts(
        """
        SELECT 'bronze_asset_algo_events' AS table_name, count(*) AS row_count FROM demo.robot.bronze_asset_algo_events
        UNION ALL SELECT 'bronze_delivery_items', count(*) FROM demo.robot.bronze_delivery_items
        UNION ALL SELECT 'silver_mcap_files_current', count(*) FROM demo.robot.silver_mcap_files_current
        UNION ALL SELECT 'silver_assets_current', count(*) FROM demo.robot.silver_assets_current
        UNION ALL SELECT 'silver_deliveries_current', count(*) FROM demo.robot.silver_deliveries_current
        UNION ALL SELECT 'silver_asset_algo_latest', count(*) FROM demo.robot.silver_asset_algo_latest
        UNION ALL SELECT 'silver_asset_tags', count(*) FROM demo.robot.silver_asset_tags
        UNION ALL SELECT 'gold_dataset_snapshot_items', count(*) FROM demo.robot.gold_dataset_snapshot_items
        """
    ),
    "questions": {
        "training_assets": {
            "question": "某次训练当时用了哪些 asset？",
            "answer": "Use gold_dataset_snapshot_items as the frozen training candidate manifest.",
            "snapshot_counts": gold_snapshot_count,
            "sample": collect_dicts(
                """
                SELECT dataset_snapshot_id, asset_id, mcap_file_id, env, task, algo_key, algo_status, quality
                FROM demo.robot.gold_dataset_snapshot_items
                ORDER BY asset_id
                """,
                limit=20,
            ),
        },
        "algorithm_recompute": {
            "question": "某个算法版本变更后，哪些历史 asset 要重算？",
            "answer": "This MVP treats assets with an existing hand_tracking@1.2.0 result as candidates for a hypothetical hand_tracking@1.3.0 recompute.",
            "candidate_counts": collect_dicts(
                """
                SELECT algo_key, status, count(*) AS asset_count
                FROM demo.robot.silver_asset_algo_latest
                WHERE algo_key = 'hand_tracking@1.2.0'
                GROUP BY algo_key, status
                ORDER BY status
                """
            ),
            "sample": collect_dicts(
                """
                SELECT a.asset_id, a.mcap_file_id, a.env, a.task, algo.algo_key, algo.status, algo.updated_at
                FROM demo.robot.silver_asset_algo_latest algo
                JOIN demo.robot.silver_assets_current a ON a.asset_id = algo.asset_id
                WHERE algo.algo_key = 'hand_tracking@1.2.0'
                  AND algo.status IN ('ok', 'failed', 'running', 'pending')
                ORDER BY algo.updated_at DESC
                """,
                limit=20,
            ),
        },
        "tag_timeline": {
            "question": "某个 tag 是什么时候被算法追加的？",
            "answer": "Current schema only has tag current-state updated_at. Exact tag append history needs asset_mutation_outbox in Postgres and bronze_asset_mutations in Iceberg.",
            "sample": collect_dicts(
                """
                SELECT tag_key, tag_value, updated_at, count(*) AS asset_count
                FROM demo.robot.silver_asset_tags
                WHERE tag_key = 'quality'
                GROUP BY tag_key, tag_value, updated_at
                ORDER BY updated_at DESC
                """,
                limit=20,
            ),
        },
        "monthly_quality": {
            "question": "上个月所有 MCAP segment 的质量分布是什么？",
            "answer": "Group current asset segments by quality tag for the latest 30-day demo window.",
            "distribution": collect_dicts(
                """
                SELECT tag.tag_value AS quality, count(*) AS asset_count
                FROM demo.robot.silver_assets_current a
                JOIN demo.robot.silver_asset_tags tag ON a.asset_id = tag.asset_id
                WHERE tag.tag_key = 'quality'
                  AND a.created_at >= date_sub(current_date(), 30)
                  AND a.created_at < date_add(current_date(), 1)
                GROUP BY tag.tag_value
                ORDER BY asset_count DESC
                """
            ),
        },
        "customer_delivery_replay": {
            "question": "某个客户交付过的数据是否能完整回放？",
            "answer": "Join delivery current-state, delivery_items, and assets to reconstruct the delivered asset manifest.",
            "customer_id": "urn:grace:customer:A",
            "delivery_counts": collect_dicts(
                """
                SELECT d.customer_id, d.status, count(DISTINCT d.delivery_id) AS delivery_count, count(di.asset_id) AS asset_count
                FROM demo.robot.silver_deliveries_current d
                JOIN demo.robot.bronze_delivery_items di ON d.delivery_id = di.delivery_id
                WHERE d.customer_id = 'urn:grace:customer:A'
                GROUP BY d.customer_id, d.status
                ORDER BY d.status
                """
            ),
            "sample": collect_dicts(
                """
                SELECT d.customer_id, d.delivery_id, d.status, d.delivered_at, di.asset_id, a.mcap_file_id, a.segment_locator
                FROM demo.robot.silver_deliveries_current d
                JOIN demo.robot.bronze_delivery_items di ON d.delivery_id = di.delivery_id
                JOIN demo.robot.silver_assets_current a ON a.asset_id = di.asset_id
                WHERE d.customer_id = 'urn:grace:customer:A'
                ORDER BY d.delivered_at DESC, di.asset_id
                """,
                limit=20,
            ),
        },
    },
}

REPORT_PATH.write_text(json.dumps(report, ensure_ascii=False, indent=2, default=json_default), encoding="utf-8")
print(f"Wrote lakehouse report to {REPORT_PATH}")

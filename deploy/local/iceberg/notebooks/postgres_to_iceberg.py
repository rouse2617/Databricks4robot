from pyspark.sql import SparkSession


POSTGRES_URL = "jdbc:postgresql://postgres:5432/data4cyber"
POSTGRES_PROPERTIES = {
    "user": "postgres",
    "password": "postgres",
    "driver": "org.postgresql.Driver",
}


spark = (
    SparkSession.builder.appName("postgres-to-iceberg")
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


tables = {
    "pg_assets": """
        SELECT
          asset_id::text,
          mcap_file_id::text,
          start_timestamp_ns,
          end_timestamp_ns,
          status,
          is_deleted,
          segment_locator,
          cf_meta::text AS cf_meta_json,
          cf_algo::text AS cf_algo_json,
          cf_tag::text AS cf_tag_json,
          cf_files::text AS cf_files_json,
          created_at,
          updated_at,
          version
        FROM assets
    """,
    "pg_mcap_files": """
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
    "pg_deliveries": """
        SELECT
          delivery_id::text,
          customer_id,
          status,
          delivered_at,
          is_deleted,
          cf_meta::text AS cf_meta_json,
          created_at,
          updated_at,
          version
        FROM deliveries
    """,
}

spark.sql("CREATE NAMESPACE IF NOT EXISTS demo.robot")

for table_name, query in tables.items():
    df = read_postgres_query(query)
    target = f"demo.robot.{table_name}"
    df.writeTo(target).using("iceberg").createOrReplace()
    print(f"Wrote {df.count()} rows to {target}")

spark.sql("SHOW TABLES IN demo.robot").show(truncate=False)
spark.sql("SELECT asset_id, status, segment_locator FROM demo.robot.pg_assets ORDER BY asset_id").show(
    truncate=False
)

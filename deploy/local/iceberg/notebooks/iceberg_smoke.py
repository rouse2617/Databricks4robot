from pyspark.sql import SparkSession


spark = (
    SparkSession.builder.appName("local-iceberg-smoke")
    .config("spark.sql.catalog.demo", "org.apache.iceberg.spark.SparkCatalog")
    .config("spark.sql.catalog.demo.type", "rest")
    .config("spark.sql.catalog.demo.uri", "http://iceberg-rest:8181")
    .config("spark.sql.catalog.demo.warehouse", "s3://warehouse/")
    .config("spark.sql.catalog.demo.io-impl", "org.apache.iceberg.aws.s3.S3FileIO")
    .config("spark.sql.catalog.demo.s3.endpoint", "http://minio:9000")
    .config("spark.sql.catalog.demo.s3.path-style-access", "true")
    .getOrCreate()
)

spark.sql("CREATE NAMESPACE IF NOT EXISTS demo.robot")

spark.sql(
    """
    CREATE TABLE IF NOT EXISTS demo.robot.assets (
      asset_id STRING,
      mcap_file_id STRING,
      segment_start_ns BIGINT,
      segment_end_ns BIGINT,
      tag STRING
    )
    USING iceberg
    """
)

spark.sql(
    """
    INSERT INTO demo.robot.assets VALUES
      ('asset-001', 'mcap-001', 0, 1000000000, 'pick'),
      ('asset-002', 'mcap-001', 1000000000, 2000000000, 'place')
    """
)

spark.sql("SELECT * FROM demo.robot.assets ORDER BY asset_id").show(truncate=False)

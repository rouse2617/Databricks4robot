# Local Iceberg Stack

This directory contains a small container-based Iceberg environment for local development.

## Services

- `minio`: S3-compatible object storage.
- `minio-init`: creates the local `warehouse` and `datalake` buckets.
- `iceberg-rest`: Apache Iceberg REST Catalog backed by MinIO.
- `spark-iceberg`: Spark + Jupyter Notebook image configured for Iceberg experiments.
- `trino`: SQL query engine for Iceberg analytical reads.

## Start

From the repository root:

```bash
make iceberg-up
```

Or directly:

```bash
cd deploy/local
docker compose --profile lakehouse up -d
```

## Access

- Spark Notebook: http://localhost:8889
- Spark UI: http://localhost:8083
- MinIO Console: http://localhost:9001
- MinIO credentials: `admin` / `password`
- Iceberg REST Catalog: http://localhost:8183
- Trino: http://localhost:8082

The Spark container mounts `deploy/local/iceberg/notebooks` into the notebook workspace.

## Smoke Test

Run the automated smoke test to verify PyIceberg + Trino connectivity:

```bash
bash deploy/local/iceberg/smoke-test.sh
```

Or open `notebooks/iceberg_smoke.py` in Jupyter and run the cells. It creates a small Iceberg table at:

```text
demo.robot_assets
```

The table data is stored in the MinIO `warehouse` bucket.

## Postgres to Iceberg Test

Start the local Postgres service:

```bash
cd deploy/local
docker compose up -d postgres
```

Then run:

```bash
cd deploy/local
docker compose --profile lakehouse exec -T spark-iceberg \
  python /home/iceberg/notebooks/notebooks/postgres_to_iceberg.py
```

This reads the seeded Postgres tables and writes these Iceberg tables:

- `demo.robot.pg_assets`
- `demo.robot.pg_mcap_files`
- `demo.robot.pg_deliveries`

## Lakehouse MVP

To generate local scale-test data in Postgres before syncing:

```bash
ROW_COUNT=100000 BATCH_ID=scale_100k make pg-generate-scale
```

Run the Bronze/Silver/Gold MVP builder:

```bash
make iceberg-mvp
```

If the backend is running on a host Postgres at `localhost:5432`, run the same MVP against that source so newly uploaded segments appear in Iceberg:

```bash
make iceberg-mvp-host
```

Or directly:

```bash
cd deploy/local
docker compose --profile lakehouse exec -T spark-iceberg \
  python /home/iceberg/notebooks/notebooks/build_lakehouse_mvp.py
```

This reads Postgres and writes:

- Bronze:
  - `demo.robot.bronze_asset_algo_events`
  - `demo.robot.bronze_delivery_items`
- Silver:
  - `demo.robot.silver_mcap_files_current`
  - `demo.robot.silver_assets_current`
  - `demo.robot.silver_deliveries_current`
  - `demo.robot.silver_asset_algo_latest`
  - `demo.robot.silver_asset_tags`
- Gold:
  - `demo.robot.gold_dataset_snapshot_items`

The Gold table is a sample dataset snapshot for assets with
`hand_tracking@1.2.0 = ok` and `quality IN ('good', 'excellent')`.

The script also writes `notebooks/lakehouse_report.json`, which is served by the backend at `/api/v1/lakehouse/report` for the frontend lakehouse validation page.

## Trino Query Test

After running the MVP builder, query Iceberg through Trino:

```bash
make trino-smoke
```

Or directly:

```bash
cd deploy/local
docker compose --profile lakehouse exec -T trino \
  trino --server http://localhost:8080 --catalog iceberg --schema robot \
  --execute "SELECT count(*) FROM silver_assets_current"
```

The backend lakehouse APIs use Trino for typed analytical reads:

- `/api/v1/lakehouse/tables`
- `/api/v1/lakehouse/training-assets`
- `/api/v1/lakehouse/recompute-candidates`
- `/api/v1/lakehouse/tag-timeline`
- `/api/v1/lakehouse/quality-distribution`
- `/api/v1/lakehouse/customer-replay`

## Stop

```bash
make iceberg-down
```

To remove persisted MinIO data too:

```bash
cd deploy/local
docker compose --profile lakehouse down -v
```

## Bronze MERGE (CDC → Iceberg)

The two-stage ingestion pipeline (§5.6.2):

1. **CDC Bronze Sink** writes staging JSONL files to `/tmp/iceberg-staging/`
2. **PyIceberg CronJob** merges staging files into `bronze_asset_events` with event_seq dedup

Run the merge manually:

```bash
python deploy/local/iceberg/bronze_merge.py
```

Dry run (report only):

```bash
DRY_RUN=true python deploy/local/iceberg/bronze_merge.py
```

## Iceberg Maintenance (Compact)

Run snapshot expiry, data file compaction, and orphan file cleanup:

```bash
python deploy/local/iceberg/compact.py
python deploy/local/iceberg/compact.py --dry-run
python deploy/local/iceberg/compact.py --expire-only
python deploy/local/iceberg/compact.py --compact-only
```

## Verification Scripts

24h no-loss verification:

```bash
bash deploy/local/iceberg/verify_no_loss.sh
```

Small file count stability (run daily for 7+ days):

```bash
bash deploy/local/iceberg/verify_small_files.sh
```

## ADR

Catalog selection decision: [docs/adr/001-iceberg-catalog-selection.md](../../docs/adr/001-iceberg-catalog-selection.md)

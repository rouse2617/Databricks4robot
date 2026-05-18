"""Incremental Postgres → Iceberg Bronze ingestion (Cloud Run Job).

Run shape:
- Triggered every 5 min by Cloud Scheduler.
- Reads cursor = MAX(event_seq) directly from the Bronze table (the table is
  its own cursor — no separate checkpoint store).
- Pulls asset_events with event_seq > cursor from Postgres in BATCH_SIZE-sized
  chunks until either the table is drained for this run or JOB_DEADLINE_SEC
  elapses.
- Appends each batch via PyIceberg `table.append`. Iceberg commits are atomic,
  so a crash mid-run leaves a clean state for the next tick.

Why this design instead of a Pub/Sub subscriber: Bronze is append-only event log
with batch-friendly Iceberg writes; per-message append would create millions of
tiny parquet files. 5-min lag is acceptable for analytics. See
docs/review/lakehouse-incremental-ingestion.md.

Bronze may contain duplicate rows. BigLake REST Catalog returns 429
mid-commit under burst load; PyIceberg's internal tenacity retry then
succeeds on the commit but still surfaces the original exception, causing
our append_with_retry below to re-append the same batch. The cursor
(MAX(event_seq)) is unaffected — so re-runs never miss events — but row
counts can exceed event counts by up to ~10% on backfill / catch-up runs.
This is by design; Silver MUST dedupe on event_id before downstream use.
See lakehouse-incremental-ingestion.md §"Duplicates in Bronze are EXPECTED".
"""

from __future__ import annotations

import json
import os
import random
import sys
import time
import traceback
from datetime import datetime, timezone

import google.auth
import google.auth.transport.requests
import psycopg2
import pyarrow as pa
import pyarrow.compute as pc
from pyiceberg.catalog import load_catalog
from pyiceberg.exceptions import NoSuchTableError
from pyiceberg.schema import Schema
from pyiceberg.types import BooleanType, NestedField, StringType, TimestamptzType


PROJECT = os.getenv("GCP_PROJECT", "green-valley-442103")
BUCKET = os.getenv("WAREHOUSE_BUCKET", "cyber-databrew-iceberg-warehouse-prod")
NAMESPACE = os.getenv("ICEBERG_NAMESPACE", "robot")
TABLE = os.getenv("ICEBERG_TABLE", "bronze_asset_events")
SILVER_QUALITY_ENABLED = os.getenv("SILVER_QUALITY_ENABLED", "true").lower() == "true"
SILVER_QUALITY_TABLE = os.getenv("SILVER_QUALITY_TABLE", "silver_asset_quality_current")
SILVER_QUALITY_MODE = os.getenv("SILVER_QUALITY_MODE", "incremental").strip().lower()

# Per-fetch chunk from the server-side cursor. 5000 keeps memory bounded
# (~10 MB arrow batch) while amortizing PG round trips.
BATCH_SIZE = int(os.getenv("BATCH_SIZE", "2000"))
SILVER_BATCH_SIZE = int(os.getenv("SILVER_BATCH_SIZE", str(BATCH_SIZE)))
SILVER_DIRTY_ID_BATCH = int(os.getenv("SILVER_DIRTY_ID_BATCH", "1000"))
# Hard ceiling for one Cloud Run Job execution. Cloud Scheduler triggers
# every 5 min, so we cap at ~4 min to leave headroom for shutdown.
JOB_DEADLINE_SEC = int(os.getenv("JOB_DEADLINE_SEC", "240"))
CURSOR_SOURCE = os.getenv("CURSOR_SOURCE", "checkpoint").strip().lower()
CURSOR_SCAN_FALLBACK = os.getenv("CURSOR_SCAN_FALLBACK", "false").lower() == "true"

DB_HOST = os.environ["DB_HOST"]
DB_PORT = int(os.getenv("DB_PORT", "5432"))
DB_USER = os.environ["DB_USER"]
DB_PASSWORD = os.environ["DB_PASSWORD"]
DB_NAME = os.environ["DB_NAME"]
DB_SSLMODE = os.getenv("DB_SSLMODE", "prefer")

# Cloud Run injects CLOUD_RUN_EXECUTION; used as a run id for the
# _source_file column so a single bad run can be filtered in audits.
RUN_ID = os.getenv("CLOUD_RUN_EXECUTION", f"local-{int(time.time())}")


def log_kv(event: str, **kwargs: object) -> None:
    payload = " ".join(f"{k}={v}" for k, v in kwargs.items())
    print(f"{event} {payload}".strip())


def elapsed_ms(start_ts: float) -> int:
    return int((time.time() - start_ts) * 1000)


def get_catalog() -> object:
    t0 = time.time()
    log_kv(
        "CATALOG_INIT_START",
        run=RUN_ID,
        project=PROJECT,
        bucket=BUCKET,
        namespace=NAMESPACE,
        table=TABLE,
        silver_enabled=SILVER_QUALITY_ENABLED,
        silver_table=SILVER_QUALITY_TABLE,
        batch_size=BATCH_SIZE,
        deadline_sec=JOB_DEADLINE_SEC,
    )
    creds, _ = google.auth.default(
        scopes=["https://www.googleapis.com/auth/cloud-platform"]
    )
    creds.refresh(google.auth.transport.requests.Request())
    catalog = load_catalog(
        "biglake",
        **{
            "type": "rest",
            "uri": "https://biglake.googleapis.com/iceberg/v1/restcatalog",
            "warehouse": f"gs://{BUCKET}",
            "token": creds.token,
            "header.x-goog-user-project": PROJECT,
            "py-io-impl": "pyiceberg.io.pyarrow.PyArrowFileIO",
            "gcs.project-id": PROJECT,
            "gcs.oauth2.token": creds.token,
            "gcs.oauth2.token-expires-at": str(
                int((creds.expiry.timestamp() if creds.expiry else time.time() + 3600) * 1000)
            ),
        },
    )
    log_kv("CATALOG_INIT_DONE", elapsed_ms=elapsed_ms(t0))
    return catalog


def get_bronze_cursor(table: object) -> int:
    """Read MAX(event_seq) from Bronze. Returns 0 for an empty table.

    Iceberg keeps per-column min/max in manifest metadata, so this is O(1)
    even when Bronze grows to billions of rows — no parquet data scanned.
    """
    seq_arr = table.scan(selected_fields=("event_seq",)).to_arrow()["event_seq"]
    if len(seq_arr) == 0:
        return 0
    return int(pc.max(seq_arr).as_py())


def get_checkpoint_cursor(conn: psycopg2.extensions.connection) -> int:
    cur = conn.cursor()
    cur.execute("SELECT COALESCE(applied_seq, 0) FROM lakehouse_bronze_checkpoint WHERE id = 1")
    row = cur.fetchone()
    cur.close()
    if not row:
        return 0
    return int(row[0] or 0)


REQUIRED_SCHEMA = pa.schema(
    [
        pa.field("event_id", pa.string(), nullable=False),
        pa.field("event_seq", pa.int64(), nullable=False),
        pa.field("event_type", pa.string(), nullable=False),
        pa.field("payload_schema_version", pa.string(), nullable=False),
        pa.field("aggregate_type", pa.string(), nullable=False),
        pa.field("asset_id", pa.string(), nullable=True),
        pa.field("mcap_file_id", pa.string(), nullable=True),
        pa.field("tenant_id", pa.string(), nullable=True),
        pa.field("project_id", pa.string(), nullable=True),
        pa.field("event_source", pa.string(), nullable=False),
        pa.field("actor_type", pa.string(), nullable=True),
        pa.field("actor_id", pa.string(), nullable=True),
        pa.field("request_id", pa.string(), nullable=True),
        pa.field("idempotency_key", pa.string(), nullable=True),
        pa.field("run_id", pa.string(), nullable=True),
        pa.field("occurred_at", pa.timestamp("us", tz="UTC"), nullable=False),
        pa.field("created_at", pa.timestamp("us", tz="UTC"), nullable=False),
        pa.field("publish_state", pa.string(), nullable=False),
        pa.field("published_at", pa.timestamp("us", tz="UTC"), nullable=True),
        pa.field("retry_count", pa.int64(), nullable=False),
        pa.field("last_error", pa.string(), nullable=True),
        pa.field("event_payload", pa.string(), nullable=False),
        pa.field("_ingested_at", pa.timestamp("us", tz="UTC"), nullable=True),
        pa.field("_source_file", pa.string(), nullable=True),
    ]
)

QUALITY_ICEBERG_SCHEMA = Schema(
    NestedField(1, "asset_id", StringType(), required=True),
    NestedField(2, "created_at", TimestamptzType(), required=True),
    NestedField(3, "updated_at", TimestamptzType(), required=True),
    NestedField(4, "is_deleted", BooleanType(), required=True),
    NestedField(5, "quality", StringType(), required=True),
    NestedField(6, "_ingested_at", TimestamptzType(), required=False),
    NestedField(7, "_source_file", StringType(), required=False),
)

QUALITY_ARROW_SCHEMA = pa.schema(
    [
        pa.field("asset_id", pa.string(), nullable=False),
        pa.field("created_at", pa.timestamp("us", tz="UTC"), nullable=False),
        pa.field("updated_at", pa.timestamp("us", tz="UTC"), nullable=False),
        pa.field("is_deleted", pa.bool_(), nullable=False),
        pa.field("quality", pa.string(), nullable=False),
        pa.field("_ingested_at", pa.timestamp("us", tz="UTC"), nullable=True),
        pa.field("_source_file", pa.string(), nullable=True),
    ]
)


def rows_to_arrow(rows: list[tuple], ingested_at: datetime) -> pa.Table:
    cols: dict[str, list] = {name: [] for name in REQUIRED_SCHEMA.names}
    for r in rows:
        (
            event_id,
            event_seq,
            event_type,
            payload_schema_version,
            aggregate_type,
            asset_id,
            mcap_file_id,
            tenant_id,
            project_id,
            event_source,
            actor_type,
            actor_id,
            request_id,
            idempotency_key,
            run_id,
            occurred_at,
            created_at,
            publish_state,
            published_at,
            retry_count,
            last_error,
            event_payload,
        ) = r
        cols["event_id"].append(str(event_id))
        cols["event_seq"].append(int(event_seq))
        cols["event_type"].append(event_type)
        cols["payload_schema_version"].append(payload_schema_version or "v1")
        cols["aggregate_type"].append(aggregate_type or "asset")
        cols["asset_id"].append(asset_id)
        cols["mcap_file_id"].append(mcap_file_id)
        cols["tenant_id"].append(tenant_id)
        cols["project_id"].append(project_id)
        cols["event_source"].append(event_source or "backend")
        cols["actor_type"].append(actor_type)
        cols["actor_id"].append(actor_id)
        cols["request_id"].append(request_id)
        cols["idempotency_key"].append(idempotency_key)
        cols["run_id"].append(run_id)
        cols["occurred_at"].append(occurred_at)
        cols["created_at"].append(created_at)
        cols["publish_state"].append(publish_state or "pending")
        cols["published_at"].append(published_at)
        cols["retry_count"].append(int(retry_count or 0))
        cols["last_error"].append(last_error)
        if isinstance(event_payload, (dict, list)):
            cols["event_payload"].append(json.dumps(event_payload, ensure_ascii=False))
        elif event_payload is None:
            cols["event_payload"].append("{}")
        else:
            cols["event_payload"].append(str(event_payload))
        cols["_ingested_at"].append(ingested_at)
        cols["_source_file"].append(RUN_ID)
    return pa.table(cols, schema=REQUIRED_SCHEMA)


def quality_rows_to_arrow(rows: list[tuple], ingested_at: datetime) -> pa.Table:
    cols: dict[str, list] = {name: [] for name in QUALITY_ARROW_SCHEMA.names}
    for asset_id, created_at, updated_at, is_deleted, quality in rows:
        cols["asset_id"].append(str(asset_id))
        cols["created_at"].append(created_at)
        cols["updated_at"].append(updated_at)
        cols["is_deleted"].append(bool(is_deleted))
        cols["quality"].append(quality or "unknown")
        cols["_ingested_at"].append(ingested_at)
        cols["_source_file"].append(RUN_ID)
    return pa.table(cols, schema=QUALITY_ARROW_SCHEMA)


def append_with_retry(
    catalog: object, table: object, table_id: str, arrow_table: pa.Table
) -> object:
    """Iceberg commit retry on transient REST/network failures.

    Returns a (possibly re-loaded) table handle. Reloading after a failure
    guards against stale metadata refs after a partial commit.
    """
    for attempt in range(1, 6):
        t0 = time.time()
        try:
            log_kv(
                "APPEND_ATTEMPT_START",
                table=table_id,
                attempt=attempt,
                rows=arrow_table.num_rows,
                columns=arrow_table.num_columns,
            )
            table.append(arrow_table)
            log_kv(
                "APPEND_ATTEMPT_OK",
                table=table_id,
                attempt=attempt,
                elapsed_ms=elapsed_ms(t0),
            )
            return table
        except Exception as e:
            msg = str(e)
            # 4xx argument/schema errors are deterministic; retrying only burns time.
            if "RESTError 400" in msg or "INVALID_ARGUMENT" in msg:
                log_kv(
                    "APPEND_FATAL_4XX",
                    table=table_id,
                    attempt=attempt,
                    elapsed_ms=elapsed_ms(t0),
                    err=repr(e),
                    schema=arrow_table.schema,
                )
                raise
            if attempt == 5:
                log_kv(
                    "APPEND_FATAL_MAX_RETRY",
                    table=table_id,
                    attempt=attempt,
                    elapsed_ms=elapsed_ms(t0),
                    err=repr(e),
                )
                raise
            sleep_s = min(20, 2**attempt)
            jitter = random.uniform(0, 0.5)
            sleep_total = sleep_s + jitter
            print(
                f"APPEND_RETRY attempt={attempt} base_sleep={sleep_s}s jitter={jitter:.3f}s sleep_total={sleep_total:.3f}s err={e}",
                file=sys.stderr,
            )
            time.sleep(sleep_total)
            table = catalog.load_table(table_id)
    return table


def rebuild_quality_silver(
    catalog: object, conn: psycopg2.extensions.connection, ingested_at: datetime
) -> tuple[object, int]:
    t0 = time.time()
    table_id = f"{NAMESPACE}.{SILVER_QUALITY_TABLE}"
    log_kv("SILVER_QUALITY_BUILD_START", table=table_id, batch_size=SILVER_BATCH_SIZE)
    try:
        catalog.drop_table(table_id)
        print(f"SILVER_QUALITY_DROP_OK table={table_id}")
    except Exception:
        pass

    table = catalog.create_table(
        table_id,
        schema=QUALITY_ICEBERG_SCHEMA,
        properties={
            "format-version": "2",
            "write.format.default": "parquet",
            "write.parquet.compression-codec": "zstd",
        },
    )

    cur = conn.cursor(name="silver_quality_cur")
    cur.itersize = SILVER_BATCH_SIZE
    cur.execute(
        """
        SELECT
          a.asset_id::text,
          a.created_at,
          a.updated_at,
          a.is_deleted,
          COALESCE(NULLIF(t.tag_value, ''), 'unknown') AS quality
        FROM assets a
        LEFT JOIN asset_tags t
          ON t.asset_id = a.asset_id AND t.tag_key = 'quality'
        ORDER BY a.asset_id ASC
        """
    )

    total = 0
    batch_index = 0
    while True:
        fetch_t0 = time.time()
        rows = cur.fetchmany(SILVER_BATCH_SIZE)
        log_kv(
            "SILVER_QUALITY_FETCH",
            batch_index=batch_index + 1,
            fetched=len(rows),
            elapsed_ms=elapsed_ms(fetch_t0),
        )
        if not rows:
            break
        batch_index += 1
        if batch_index == 1:
            first = rows[0]
            log_kv(
                "SILVER_QUALITY_FIRST_ROW",
                asset_id=first[0],
                created_at=first[1],
                updated_at=first[2],
                is_deleted=first[3],
                quality=first[4],
            )
        arrow_table = quality_rows_to_arrow(rows, ingested_at)
        table = append_with_retry(catalog, table, table_id, arrow_table)
        total += len(rows)
        print(
            f"SILVER_QUALITY_BATCH_OK index={batch_index} rows={len(rows)} total={total}"
        )
    cur.close()
    log_kv(
        "SILVER_QUALITY_DONE",
        rows=total,
        table=table_id,
        elapsed_ms=elapsed_ms(t0),
    )
    return table, total


def ensure_quality_table(catalog: object, table_id: str) -> object:
    try:
        return catalog.load_table(table_id)
    except NoSuchTableError:
        log_kv("SILVER_QUALITY_TABLE_MISSING", table=table_id, action="create")
        return catalog.create_table(
            table_id,
            schema=QUALITY_ICEBERG_SCHEMA,
            properties={
                "format-version": "2",
                "write.format.default": "parquet",
                "write.parquet.compression-codec": "zstd",
            },
        )


def fetch_quality_rows_for_asset_ids(
    conn: psycopg2.extensions.connection, asset_ids: list[str]
) -> list[tuple]:
    cur = conn.cursor()
    cur.execute(
        """
        SELECT
          a.asset_id::text,
          a.created_at,
          a.updated_at,
          a.is_deleted,
          COALESCE(NULLIF(t.tag_value, ''), 'unknown') AS quality
        FROM assets a
        LEFT JOIN asset_tags t
          ON t.asset_id = a.asset_id AND t.tag_key = 'quality'
        WHERE a.asset_id = ANY(%s)
        ORDER BY a.asset_id ASC
        """,
        (asset_ids,),
    )
    rows = cur.fetchall()
    cur.close()
    return rows


def rebuild_quality_silver_incremental(
    catalog: object,
    conn: psycopg2.extensions.connection,
    ingested_at: datetime,
    dirty_asset_ids: set[str],
) -> tuple[object, int]:
    t0 = time.time()
    table_id = f"{NAMESPACE}.{SILVER_QUALITY_TABLE}"
    table = ensure_quality_table(catalog, table_id)
    if not dirty_asset_ids:
        log_kv("SILVER_QUALITY_INCREMENTAL_SKIP", reason="no_dirty_assets")
        return table, 0

    dirty_ids = sorted(dirty_asset_ids)
    log_kv(
        "SILVER_QUALITY_INCREMENTAL_START",
        table=table_id,
        dirty_assets=len(dirty_ids),
        id_batch=SILVER_DIRTY_ID_BATCH,
    )
    total_rows = 0
    chunk_idx = 0
    for i in range(0, len(dirty_ids), SILVER_DIRTY_ID_BATCH):
        chunk_idx += 1
        ids = dirty_ids[i : i + SILVER_DIRTY_ID_BATCH]
        fetch_t0 = time.time()
        rows = fetch_quality_rows_for_asset_ids(conn, ids)
        log_kv(
            "SILVER_QUALITY_INCREMENTAL_FETCH",
            chunk=chunk_idx,
            ids=len(ids),
            rows=len(rows),
            elapsed_ms=elapsed_ms(fetch_t0),
        )
        if not rows:
            continue
        arrow_table = quality_rows_to_arrow(rows, ingested_at)
        table = append_with_retry(catalog, table, table_id, arrow_table)
        total_rows += len(rows)
        log_kv(
            "SILVER_QUALITY_INCREMENTAL_APPEND_OK",
            chunk=chunk_idx,
            appended_rows=len(rows),
            total_rows=total_rows,
        )
    log_kv(
        "SILVER_QUALITY_INCREMENTAL_DONE",
        dirty_assets=len(dirty_ids),
        appended_rows=total_rows,
        elapsed_ms=elapsed_ms(t0),
    )
    return table, total_rows


def main() -> int:
    start = time.time()
    deadline = start + JOB_DEADLINE_SEC

    log_kv("JOB_START", run=RUN_ID)
    conn = psycopg2.connect(
        host=DB_HOST,
        port=DB_PORT,
        user=DB_USER,
        password=DB_PASSWORD,
        dbname=DB_NAME,
        sslmode=DB_SSLMODE,
    )
    conn.autocommit = False
    catalog = get_catalog()
    bronze_table_id = f"{NAMESPACE}.{TABLE}"
    load_t0 = time.time()
    table = catalog.load_table(bronze_table_id)
    log_kv("BRONZE_TABLE_LOAD_DONE", table=bronze_table_id, elapsed_ms=elapsed_ms(load_t0))
    cursor_t0 = time.time()
    if CURSOR_SOURCE == "checkpoint":
        cursor = get_checkpoint_cursor(conn)
        log_kv("CHECKPOINT_CURSOR_DONE", cursor=cursor, elapsed_ms=elapsed_ms(cursor_t0))
        if cursor == 0 and CURSOR_SCAN_FALLBACK:
            scan_t0 = time.time()
            cursor = get_bronze_cursor(table)
            log_kv("BRONZE_CURSOR_SCAN_FALLBACK_DONE", cursor=cursor, elapsed_ms=elapsed_ms(scan_t0))
    else:
        cursor = get_bronze_cursor(table)
        log_kv("BRONZE_CURSOR_SCAN_DONE", cursor=cursor, elapsed_ms=elapsed_ms(cursor_t0))
    print(
        f"START run={RUN_ID} cursor={cursor} batch_size={BATCH_SIZE} "
        f"deadline_sec={JOB_DEADLINE_SEC}"
    )

    # Named cursor → server-side streaming, never loads all rows at once.
    cur = conn.cursor(name="bronze_incremental_cur")
    cur.itersize = BATCH_SIZE
    cur.execute(
        """
        SELECT
          event_id::text,
          event_seq,
          event_type,
          payload_schema_version,
          aggregate_type,
          asset_id,
          mcap_file_id,
          tenant_id,
          project_id,
          event_source,
          actor_type,
          actor_id,
          request_id,
          idempotency_key,
          run_id,
          occurred_at,
          created_at,
          publish_state,
          published_at,
          retry_count,
          last_error,
          event_payload
        FROM asset_events
        WHERE event_seq > %s
        ORDER BY event_seq ASC
        """,
        (cursor,),
    )

    total = 0
    batch_index = 0
    last_seq = cursor
    ingested_at = datetime.now(timezone.utc)
    dirty_asset_ids: set[str] = set()

    while True:
        if time.time() >= deadline:
            print(
                f"DEADLINE_HIT processed={total} last_seq={last_seq} "
                f"elapsed_sec={int(time.time() - start)}"
            )
            break
        fetch_t0 = time.time()
        rows = cur.fetchmany(BATCH_SIZE)
        log_kv(
            "BRONZE_FETCH",
            batch_index=batch_index + 1,
            fetched=len(rows),
            elapsed_ms=elapsed_ms(fetch_t0),
        )
        if not rows:
            break
        batch_index += 1
        build_t0 = time.time()
        arrow_table = rows_to_arrow(rows, ingested_at)
        log_kv(
            "BRONZE_ROWS_TO_ARROW_DONE",
            batch_index=batch_index,
            rows=arrow_table.num_rows,
            elapsed_ms=elapsed_ms(build_t0),
        )
        table = append_with_retry(catalog, table, bronze_table_id, arrow_table)
        total += len(rows)
        last_seq = rows[-1][1]
        for r in rows:
            aid = r[5]
            if aid:
                dirty_asset_ids.add(str(aid))
        print(
            f"BATCH_OK index={batch_index} rows={len(rows)} total={total} "
            f"min_seq={rows[0][1]} max_seq={last_seq}"
        )

    cur.close()

    # Persist the checkpoint so backend /api/v1/lakehouse/sync-progress and
    # the lakehouse_bronze_* Prometheus gauges can report accurate watermarks
    # without depending on Trino's view of the BigLake catalog.
    # Only advance applied_seq forwards (GREATEST) so an idle run with
    # processed=0 still refreshes ingested_at + run_id without rolling back.
    if last_seq > 0:
        upsert_cur = conn.cursor()
        upsert_cur.execute(
            """
            INSERT INTO lakehouse_bronze_checkpoint (id, applied_seq, ingested_at, run_id, updated_at)
            VALUES (1, %s, %s, %s, now())
            ON CONFLICT (id) DO UPDATE
              SET applied_seq = GREATEST(lakehouse_bronze_checkpoint.applied_seq, EXCLUDED.applied_seq),
                  ingested_at = EXCLUDED.ingested_at,
                  run_id      = EXCLUDED.run_id,
                  updated_at  = now()
            """,
            (last_seq, ingested_at, RUN_ID),
        )
        conn.commit()
        upsert_cur.close()
        print(f"CHECKPOINT_UPSERT applied_seq={last_seq} run_id={RUN_ID}")

    silver_table = None
    silver_total = 0
    if SILVER_QUALITY_ENABLED:
        try:
            if SILVER_QUALITY_MODE == "full":
                silver_table, silver_total = rebuild_quality_silver(catalog, conn, ingested_at)
            else:
                silver_table, silver_total = rebuild_quality_silver_incremental(
                    catalog, conn, ingested_at, dirty_asset_ids
                )
        except Exception as exc:  # noqa: BLE001
            print(f"SILVER_QUALITY_FAILED err={exc!r}", file=sys.stderr)
            print(traceback.format_exc(), file=sys.stderr)
            conn.close()
            return 1

    conn.close()

    # Refresh the BigQuery external table pointer so analytical queries (via
    # BQ) see the latest Iceberg snapshot. Best-effort: failures here do NOT
    # fail the ingest run — Bronze data is already durably committed in GCS.
    if total > 0:
        try:
            refresh_bq_external_table(
                table=table,
                dataset=os.getenv("BQ_DATASET", "").strip(),
                bq_table=os.getenv("BQ_TABLE", TABLE).strip(),
                label="bronze",
            )
        except Exception as exc:  # noqa: BLE001
            print(f"BQ_REFRESH_FAILED err={exc!r}", file=sys.stderr)
    if SILVER_QUALITY_ENABLED and silver_table is not None:
        silver_dataset = os.getenv("BQ_SILVER_DATASET", os.getenv("BQ_DATASET", "")).strip()
        silver_bq_table = os.getenv("BQ_SILVER_TABLE", SILVER_QUALITY_TABLE).strip()
        try:
            refresh_bq_external_table(
                table=silver_table,
                dataset=silver_dataset,
                bq_table=silver_bq_table,
                label="silver_quality",
            )
        except Exception as exc:  # noqa: BLE001
            print(f"BQ_SILVER_REFRESH_FAILED err={exc!r}", file=sys.stderr)

    elapsed = int(time.time() - start)
    print(
        f"DONE run={RUN_ID} processed={total} dirty_assets={len(dirty_asset_ids)} "
        f"silver_quality_rows={silver_total} last_seq={last_seq} "
        f"elapsed_sec={elapsed}"
    )
    return 0


def refresh_bq_external_table(table: object, dataset: str, bq_table: str, label: str) -> None:
    """Point the BigQuery external Iceberg table at the latest metadata.json.

    The BQ external table is created once via `bq mk
    --external_table_definition` (see deploy/cloudrun/bronze-incremental/README.md).
    After each PyIceberg commit, the snapshot moves forward to a new
    metadata.json — we update the table's source_uris so the next BQ query
    sees fresh data. The BQ_DATASET env var gates this step: leave it empty
    to skip (e.g. for environments that do not query Bronze via BQ).
    """

    if not dataset:
        print(f"BQ_REFRESH_SKIPPED label={label} reason=dataset_unset")
        return

    metadata_uri = getattr(table, "metadata_location", None)
    if not metadata_uri:
        print(f"BQ_REFRESH_SKIPPED label={label} reason=metadata_location_missing")
        return

    from google.api_core.exceptions import NotFound
    from google.cloud import bigquery  # local import; optional dep
    bq = bigquery.Client(project=PROJECT)
    full_id = f"{PROJECT}.{dataset}.{bq_table}"
    try:
        tbl = bq.get_table(full_id)
    except NotFound:
        tbl = bigquery.Table(full_id)
        cfg = bigquery.ExternalConfig("ICEBERG")
        cfg.source_uris = [metadata_uri]
        tbl.external_data_configuration = cfg
        bq.create_table(tbl)
        print(f"BQ_REFRESH_CREATE_OK label={label} table={full_id} metadata={metadata_uri}")
        return
    cfg = tbl.external_data_configuration
    if cfg is None:
        print(
            f"BQ_REFRESH_SKIPPED label={label} reason=not_external_table table={full_id}"
        )
        return
    cfg.source_uris = [metadata_uri]
    tbl.external_data_configuration = cfg
    bq.update_table(tbl, ["external_data_configuration"])
    print(f"BQ_REFRESH_OK label={label} table={full_id} metadata={metadata_uri}")


if __name__ == "__main__":
    sys.exit(main())

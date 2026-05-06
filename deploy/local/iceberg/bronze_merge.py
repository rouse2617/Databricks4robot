#!/usr/bin/env python3
"""
bronze_merge.py — PyIceberg CronJob: MERGE INTO bronze with event_seq dedup.

Design reference: data-platform-design.md §5.6.2 (two-stage ingestion)

This script implements the second stage of the two-stage Iceberg ingestion:
  1. Lists new JSONL staging files written by the CDC Bronze Sink
  2. Reads events from staging files
  3. Performs MERGE INTO bronze.asset_events USING staging ON event_seq
     (idempotent dedup — same event_seq is never inserted twice)
  4. Deletes consumed staging files after successful merge

File-based work units (no PG cursor needed):
  - The staging directory IS the work queue
  - Files named: events_{minSeq}_{maxSeq}_{timestamp}.jsonl
  - event_seq in MERGE ON clause guarantees idempotent dedup
  - Structural impossibility of event loss (§5.6.2)

Usage:
  # Local (staging dir on filesystem):
  python bronze_merge.py

  # With environment overrides:
  STAGING_DIR=/path/to/staging \
  ICEBERG_REST_URI=http://localhost:8181 \
  python bronze_merge.py

  # As k8s CronJob (every 5 minutes):
  # See deploy/k8s/cronjob-bronze-merge.yaml

Environment variables:
  STAGING_DIR          — path to staging JSONL files (default: /tmp/iceberg-staging)
  ICEBERG_REST_URI     — Iceberg REST Catalog URI (default: http://localhost:8181)
  S3_ENDPOINT          — MinIO/S3 endpoint (default: http://localhost:9000)
  S3_ACCESS_KEY        — S3 access key (default: admin)
  S3_SECRET_KEY        — S3 secret key (default: password)
  S3_REGION            — S3 region (default: us-east-1)
  WAREHOUSE            — Iceberg warehouse path (default: s3://warehouse/)
  ICEBERG_NAMESPACE    — target namespace (default: robot)
  BRONZE_TABLE         — target table name (default: bronze_asset_events)
  DRY_RUN              — if "true", skip actual merge and delete (default: false)
"""

import json
import os
import sys
from datetime import datetime, timezone
from glob import glob
from pathlib import Path

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

STAGING_DIR = os.getenv("STAGING_DIR", "/tmp/iceberg-staging")
ICEBERG_REST_URI = os.getenv("ICEBERG_REST_URI", "http://localhost:8181")
S3_ENDPOINT = os.getenv("S3_ENDPOINT", "http://localhost:9000")
S3_ACCESS_KEY = os.getenv("S3_ACCESS_KEY", "admin")
S3_SECRET_KEY = os.getenv("S3_SECRET_KEY", "password")
S3_REGION = os.getenv("S3_REGION", "us-east-1")
WAREHOUSE = os.getenv("WAREHOUSE", "s3://warehouse/")
ICEBERG_NAMESPACE = os.getenv("ICEBERG_NAMESPACE", "robot")
BRONZE_TABLE = os.getenv("BRONZE_TABLE", "bronze_asset_events")
DRY_RUN = os.getenv("DRY_RUN", "false").lower() == "true"

# ---------------------------------------------------------------------------
# PyIceberg catalog setup
# ---------------------------------------------------------------------------


def get_catalog():
    """Create a PyIceberg REST catalog connection."""
    from pyiceberg.catalog import load_catalog

    catalog = load_catalog(
        "rest",
        **{
            "type": "rest",
            "uri": ICEBERG_REST_URI,
            "s3.endpoint": S3_ENDPOINT,
            "s3.access-key-id": S3_ACCESS_KEY,
            "s3.secret-access-key": S3_SECRET_KEY,
            "s3.region": S3_REGION,
            "warehouse": WAREHOUSE,
        },
    )
    return catalog


def ensure_bronze_table(catalog):
    """
    Ensure the bronze_asset_events Iceberg table exists.
    Creates it if missing with the expected schema.
    """
    import pyarrow as pa
    from pyiceberg.schema import Schema
    from pyiceberg.types import (
        LongType,
        NestedField,
        StringType,
        TimestamptzType,
    )

    table_id = (ICEBERG_NAMESPACE, BRONZE_TABLE)

    try:
        return catalog.load_table(table_id)
    except Exception:
        pass  # Table doesn't exist, create it

    # Create namespace if needed
    try:
        catalog.create_namespace(ICEBERG_NAMESPACE)
    except Exception:
        pass  # Already exists

    schema = Schema(
        NestedField(1, "event_id", StringType(), required=True),
        NestedField(2, "event_seq", LongType(), required=True),
        NestedField(3, "event_type", StringType(), required=True),
        NestedField(4, "aggregate_type", StringType(), required=False),
        NestedField(5, "payload_schema_version", StringType(), required=False),
        NestedField(6, "asset_id", StringType(), required=False),
        NestedField(7, "mcap_file_id", StringType(), required=False),
        NestedField(8, "tenant_id", StringType(), required=False),
        NestedField(9, "project_id", StringType(), required=False),
        NestedField(10, "event_source", StringType(), required=False),
        NestedField(11, "publish_state", StringType(), required=False),
        NestedField(12, "event_payload", StringType(), required=False),
        NestedField(13, "occurred_at", TimestamptzType(), required=False),
        NestedField(14, "created_at", TimestamptzType(), required=False),
        NestedField(15, "_staging_file", StringType(), required=False),
        NestedField(16, "_merged_at", TimestamptzType(), required=False),
    )

    table = catalog.create_table(table_id, schema=schema)
    print(f"Created bronze table: {ICEBERG_NAMESPACE}.{BRONZE_TABLE}")
    return table


# ---------------------------------------------------------------------------
# Staging file processing
# ---------------------------------------------------------------------------


def list_staging_files():
    """List all JSONL staging files, sorted by name (oldest first)."""
    pattern = os.path.join(STAGING_DIR, "events_*.jsonl")
    files = sorted(glob(pattern))
    return files


def read_staging_file(filepath):
    """Read a JSONL staging file and return a list of event dicts."""
    events = []
    with open(filepath, "r") as f:
        for line in f:
            line = line.strip()
            if line:
                events.append(json.loads(line))
    return events


def events_to_arrow(events, staging_file):
    """Convert event dicts to a PyArrow table for Iceberg append."""
    import pyarrow as pa

    now = datetime.now(timezone.utc)

    rows = {
        "event_id": [],
        "event_seq": [],
        "event_type": [],
        "aggregate_type": [],
        "payload_schema_version": [],
        "asset_id": [],
        "mcap_file_id": [],
        "tenant_id": [],
        "project_id": [],
        "event_source": [],
        "publish_state": [],
        "event_payload": [],
        "occurred_at": [],
        "created_at": [],
        "_staging_file": [],
        "_merged_at": [],
    }

    for e in events:
        rows["event_id"].append(e.get("event_id", ""))
        rows["event_seq"].append(e.get("event_seq", 0))
        rows["event_type"].append(e.get("event_type", ""))
        rows["aggregate_type"].append(e.get("aggregate_type", ""))
        rows["payload_schema_version"].append(e.get("payload_schema_version", ""))
        rows["asset_id"].append(e.get("asset_id", ""))
        rows["mcap_file_id"].append(e.get("mcap_file_id", ""))
        rows["tenant_id"].append(e.get("tenant_id", ""))
        rows["project_id"].append(e.get("project_id", ""))
        rows["event_source"].append(e.get("event_source", ""))
        rows["publish_state"].append(e.get("publish_state", ""))
        # Store event_payload as JSON string
        payload = e.get("event_payload")
        if isinstance(payload, dict):
            rows["event_payload"].append(json.dumps(payload))
        else:
            rows["event_payload"].append(str(payload) if payload else "{}")

        # Parse timestamps
        for ts_field in ("occurred_at", "created_at"):
            ts_str = e.get(ts_field, "")
            if ts_str:
                try:
                    ts = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
                    rows[ts_field].append(ts)
                except (ValueError, TypeError):
                    rows[ts_field].append(None)
            else:
                rows[ts_field].append(None)

        rows["_staging_file"].append(os.path.basename(staging_file))
        rows["_merged_at"].append(now)

    # Iceberg bronze table schema marks event_id/event_seq/event_type as REQUIRED.
    # PyArrow defaults fields to nullable=True; set nullable=False for the 3 required columns
    # to avoid append-time schema mismatch.
    schema = pa.schema([
        pa.field("event_id", pa.string(), nullable=False),
        pa.field("event_seq", pa.int64(), nullable=False),
        pa.field("event_type", pa.string(), nullable=False),
        ("aggregate_type", pa.string()),
        ("payload_schema_version", pa.string()),
        ("asset_id", pa.string()),
        ("mcap_file_id", pa.string()),
        ("tenant_id", pa.string()),
        ("project_id", pa.string()),
        ("event_source", pa.string()),
        ("publish_state", pa.string()),
        ("event_payload", pa.string()),
        ("occurred_at", pa.timestamp("us", tz="UTC")),
        ("created_at", pa.timestamp("us", tz="UTC")),
        ("_staging_file", pa.string()),
        ("_merged_at", pa.timestamp("us", tz="UTC")),
    ])

    return pa.table(rows, schema=schema)


def get_existing_event_seqs(table):
    """
    Read existing event_seq values from the bronze table for dedup.
    Returns a set of event_seq values.
    """
    try:
        scan = table.scan(selected_fields=("event_seq",))
        result = scan.to_arrow()
        return set(result.column("event_seq").to_pylist())
    except Exception:
        return set()


def merge_batch(table, arrow_table, existing_seqs):
    """
    Append only events whose event_seq is not already in the bronze table.
    This implements the MERGE INTO ... ON event_seq dedup logic using
    PyIceberg's append API with pre-filtered data.
    """
    import pyarrow.compute as pc
    import pyarrow as pa

    # Fast path: empty existing set → append everything.
    if not existing_seqs:
        table.append(arrow_table)
        return arrow_table.num_rows

    # Filter out already-existing event_seqs (dedup).
    # Note: pyarrow.compute has no `array`; build value_set via pyarrow.array.
    seq_col = arrow_table.column("event_seq")
    value_set = pa.array(list(existing_seqs))
    mask = pc.invert(pc.is_in(seq_col, value_set=value_set))
    new_events = arrow_table.filter(mask)

    if new_events.num_rows == 0:
        return 0

    table.append(new_events)
    return new_events.num_rows


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main():
    print(f"[{datetime.now(timezone.utc).isoformat()}] Bronze MERGE starting")
    print(f"  staging_dir: {STAGING_DIR}")
    print(f"  catalog:     {ICEBERG_REST_URI}")
    print(f"  target:      {ICEBERG_NAMESPACE}.{BRONZE_TABLE}")
    print(f"  dry_run:     {DRY_RUN}")

    # List staging files
    files = list_staging_files()
    if not files:
        print("  No staging files found. Nothing to do.")
        return

    print(f"  Found {len(files)} staging file(s)")

    if DRY_RUN:
        for f in files:
            events = read_staging_file(f)
            print(f"  [dry-run] {os.path.basename(f)}: {len(events)} events")
        return

    # Connect to catalog and ensure table exists
    catalog = get_catalog()
    table = ensure_bronze_table(catalog)

    # Get existing event_seqs for dedup
    existing_seqs = get_existing_event_seqs(table)
    print(f"  Existing events in bronze: {len(existing_seqs)}")

    total_merged = 0
    total_skipped = 0
    files_processed = 0

    for staging_file in files:
        basename = os.path.basename(staging_file)
        events = read_staging_file(staging_file)
        if not events:
            print(f"  [{basename}] Empty file, removing")
            os.remove(staging_file)
            continue

        arrow_table = events_to_arrow(events, staging_file)
        merged = merge_batch(table, arrow_table, existing_seqs)
        skipped = len(events) - merged

        # Update existing_seqs with newly merged events
        for e in events:
            existing_seqs.add(e.get("event_seq", 0))

        total_merged += merged
        total_skipped += skipped
        files_processed += 1

        print(f"  [{basename}] {merged} merged, {skipped} deduped")

        # Delete consumed staging file
        os.remove(staging_file)
        print(f"  [{basename}] Staging file deleted")

    print(f"\nBronze MERGE complete:")
    print(f"  Files processed: {files_processed}")
    print(f"  Events merged:   {total_merged}")
    print(f"  Events deduped:  {total_skipped}")


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)

#!/usr/bin/env python3
"""
HTTP load generator: POST one MCAP file then many assets via /api/v1.

Designed for local CDC verification against Postgres → Debezium → Kafka → ES.

Run from repo root (uses sdk env with httpx):

  cd sdk && uv sync --dev && uv run python ../backend/scripts/api_loadtest_assets.py \\
    --base-url http://localhost:8080 --token dev-token --count 100000 --concurrency 48

Verify Postgres vs Elasticsearch (marker in asset metadata.loadtest_batch):

  psql ... -c \"SELECT count(*) FROM assets WHERE metadata->>'loadtest_batch'='api100k';\"

  curl -s http://localhost:9200/assets/_count -H 'Content-Type: application/json' \\
    -d '{\"query\":{\"term\":{\"metadata.loadtest_batch.keyword\":\"api100k\"}}}'
"""

from __future__ import annotations

import argparse
import asyncio
import hashlib
import json
import sys
import time
import uuid


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="POST assets via API for load / CDC testing.")
    p.add_argument("--base-url", default="http://localhost:8080", help="API origin (no trailing slash)")
    p.add_argument("--token", default="dev-token", help="X-Grace-Token header")
    p.add_argument("--batch-marker", default="api100k", help="metadata.loadtest_batch value")
    p.add_argument("--count", type=int, default=100_000, help="number of assets to create")
    p.add_argument("--concurrency", type=int, default=48, help="async HTTP concurrency")
    p.add_argument("--dry-run", action="store_true", help="print plan and exit")
    return p.parse_args()


async def _main() -> int:
    try:
        import httpx
    except ImportError:
        print("Missing httpx. Run from sdk with: cd sdk && uv run python ../backend/scripts/api_loadtest_assets.py ...", file=sys.stderr)
        return 2

    args = _parse_args()
    base = args.base_url.rstrip("/")
    headers = {"X-Grace-Token": args.token, "Content-Type": "application/json"}

    mcap_id = str(uuid.uuid4())
    # mcap_files enforces unique raw_hash_md5 locally — derive from this run's id.
    raw_md5 = hashlib.md5(f"loadtest:{args.batch_marker}:{mcap_id}".encode()).hexdigest()
    mcap_body = {
        "mcap_file_id": mcap_id,
        "gcs_path": f"gs://loadtest/{args.batch_marker}/{mcap_id}.mcap",
        "size_bytes": 1024,
        "raw_hash_md5": raw_md5,
        "start_timestamp_ns": 1_700_000_000_000_000_000,
        "end_timestamp_ns": 1_700_000_060_000_000_000,
        "channel_count": 4,
        "chunk_count": 8,
        "owner": "loadtest",
        "metadata": {"loadtest_batch": args.batch_marker},
    }

    if args.dry_run:
        print(json.dumps({"mcap": mcap_body, "first_asset_template": "..."}, indent=2))
        return 0

    limits = httpx.Limits(max_connections=args.concurrency + 8, max_keepalive_connections=args.concurrency + 8)
    timeout = httpx.Timeout(120.0, connect=30.0)

    t0 = time.perf_counter()
    sem = asyncio.Semaphore(args.concurrency)
    sample_lock = asyncio.Lock()
    fail_samples: list[tuple[int, str]] = []

    async def one_asset(client: httpx.AsyncClient, idx: int) -> bool:
        start = 1_700_000_000_000_000_000 + idx * 1_000_000_000
        end = start + 500_000_000
        body = {
            "mcap_file_id": mcap_id,
            "start_timestamp_ns": start,
            "end_timestamp_ns": end,
            "reviewer": "loadtest",
            "owner": "loadtest",
            "metadata": {"loadtest_batch": args.batch_marker},
        }
        async with sem:
            r = await client.post(f"{base}/api/v1/assets", headers=headers, json=body)
        if r.status_code == 201:
            return True
        async with sample_lock:
            if len(fail_samples) < 8:
                fail_samples.append((r.status_code, r.text[:400]))
        return False

    async with httpx.AsyncClient(limits=limits, timeout=timeout) as client:
        r = await client.post(f"{base}/api/v1/mcap-files", headers=headers, json=mcap_body)
        if r.status_code != 201:
            print(f"mcap-files failed: {r.status_code} {r.text}", file=sys.stderr)
            return 1
        print(f"mcap_file_id={mcap_id} created ({r.status_code})")

        tasks = [asyncio.create_task(one_asset(client, i)) for i in range(args.count)]
        outcomes = await asyncio.gather(*tasks)

    ok = sum(1 for o in outcomes if o)
    fail = args.count - ok
    for code, body in fail_samples:
        print(f"[fail sample] {code} {body}", file=sys.stderr)

    elapsed = time.perf_counter() - t0
    rate = args.count / elapsed if elapsed > 0 else 0
    print(
        json.dumps(
            {
                "elapsed_sec": round(elapsed, 3),
                "assets_requested": args.count,
                "assets_ok": ok,
                "assets_failed": fail,
                "assets_per_sec": round(rate, 2),
                "mcap_file_id": mcap_id,
                "batch_marker": args.batch_marker,
            },
            indent=2,
        )
    )
    return 0 if fail == 0 else 1


if __name__ == "__main__":
    raise SystemExit(asyncio.run(_main()))

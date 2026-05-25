#!/usr/bin/env python3
"""
Create sample deliveries via POST /api/v1/deliveries (requires Idempotency-Key).

Typical flow after seed_rich_dataset.py:
  python3 backend/scripts/seed_sample_deliveries.py \\
    --sample-json /tmp/seed_rich_sample.json \\
    --count 5

Requires: httpx
"""

from __future__ import annotations

import argparse
import json
import random
import uuid
from typing import Any

import httpx

BASE = "http://127.0.0.1:8080"
TOKEN = "dev-token"


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="POST sample deliveries for UI testing")
    p.add_argument("--base", default=BASE, help="API base URL")
    p.add_argument("--token", default=TOKEN, help="X-Databrew-Token")
    p.add_argument(
        "--sample-json",
        default="/tmp/seed_rich_sample.json",
        help="JSON file with key sample_asset_ids (from seed_rich_dataset.py)",
    )
    p.add_argument("--count", type=int, default=5, help="How many deliveries to create")
    p.add_argument("--customer-id", default="seed_customer", help="customer_id field")
    p.add_argument("--batch-size", type=int, default=12, help="asset_ids per delivery (random subset)")
    return p.parse_args()


def load_asset_ids(path: str) -> list[str]:
    with open(path) as f:
        data: dict[str, Any] = json.load(f)
    ids = data.get("all_asset_ids") or data.get("sample_asset_ids") or data.get("asset_ids")
    if not ids:
        raise SystemExit(f"No sample_asset_ids / asset_ids in {path}")
    return [str(x) for x in ids]


def main() -> None:
    args = parse_args()
    asset_ids = load_asset_ids(args.sample_json)
    if len(asset_ids) < 1:
        raise SystemExit("Need at least one asset id")

    url = f"{args.base.rstrip('/')}/api/v1/deliveries"
    headers = {
        "X-Databrew-Token": args.token,
        "Content-Type": "application/json",
    }

    created: list[str] = []
    with httpx.Client(timeout=30.0) as client:
        for i in range(args.count):
            k = min(args.batch_size, len(asset_ids))
            batch = random.sample(asset_ids, k=k)
            idem = str(uuid.uuid4())
            payload = {
                "customer_id": args.customer_id,
                "asset_ids": batch,
                "note": f"seed_sample_deliveries batch {i + 1}/{args.count}",
            }
            resp = client.post(
                url,
                json=payload,
                headers={**headers, "Idempotency-Key": idem},
            )
            if resp.status_code != 201:
                print(f"FAIL {resp.status_code}: {resp.text[:500]}")
                continue
            body = resp.json()
            did = body.get("delivery_id") or body.get("id")
            if did:
                created.append(str(did))
                print(f"Created delivery {did} ({len(batch)} assets)")
            else:
                print(f"201 but unexpected body: {body}")

    print(f"\nDone: {len(created)} deliveries. Open /deliveries/{{id}} in the UI.")
    for did in created[:20]:
        print(f"  {args.base.rstrip('/')}/deliveries/{did}")


if __name__ == "__main__":
    main()

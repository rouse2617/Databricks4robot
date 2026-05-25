#!/usr/bin/env python3
"""
Simulate many delivery commits against Cloud Run dev (or any BASE URL).

Streams asset IDs via POST /api/v1/queries/run (page_size fixed), splits into
variable-size chunks (waves / customers / contracts / notes / owners), and
commits each chunk with POST /api/v1/deliveries + Idempotency-Key.

  export DATABREW_TOKEN=...
  python3 scripts/bulk_deliveries_simulate_cloudrun.py [--max-assets 0] [--dry-run]

--max-assets 0 means no cap (all pages until empty). Use a positive cap for smoke tests.
"""
from __future__ import annotations

import argparse
import http.client
import json
import random
import ssl
import sys
import time
import uuid
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Iterator

BASE = "https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app"
PAGE_SIZE = 200

# “复杂”：多客户 / 合同 / owner，轮替使用；note 里塞结构化摘要
WAVES = [
    {
        "customer_id": "cust-robotics-east-01",
        "contract_id": "cnt-2026-ROBO-ALPHA",
        "owner": "delivery-bot-east",
    },
    {
        "customer_id": "cust-sim-lab-sandbox",
        "contract_id": "cnt-sim-STREAM-7",
        "owner": "sim-ops-sandbox",
    },
    {
        "customer_id": "cust-fleet-replay-co",
        "contract_id": "cnt-FLEET-REPLAY-2026Q1",
        "owner": "fleet-replay-admin",
    },
    {
        "customer_id": "cust-eval-harness",
        "contract_id": "cnt-EVAL-MULTI-WAVE",
        "owner": "eval-harness-ci",
    },
    {
        "customer_id": "cust-cross-border-x",
        "contract_id": "cnt-XBORDER-TRAIN-99",
        "owner": "xb-train-scheduler",
    },
]


def http_json(
    method: str,
    url: str,
    token: str,
    body: dict[str, Any] | None = None,
    extra_headers: dict[str, str] | None = None,
    timeout: int = 300,
    retries: int = 5,
) -> tuple[int, Any]:
    data = None
    headers = {
        "X-Databrew-Token": token,
        "Accept": "application/json",
    }
    if extra_headers:
        headers.update(extra_headers)
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    ctx = ssl.create_default_context()
    last_err: Exception | None = None
    for attempt in range(retries):
        req = urllib.request.Request(url, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=timeout, context=ctx) as resp:
                raw = resp.read().decode("utf-8")
                code = resp.getcode()
                if not raw:
                    return code, None
                return code, json.loads(raw)
        except urllib.error.HTTPError as e:
            raw = e.read().decode("utf-8", errors="replace")
            try:
                parsed = json.loads(raw) if raw else None
            except json.JSONDecodeError:
                parsed = raw
            return e.code, parsed
        except (TimeoutError, OSError, http.client.IncompleteRead) as e:
            last_err = e
            time.sleep(1.0 * (attempt + 1) + random.random())
    raise last_err if last_err else TimeoutError(url)


def query_page(token: str, page: int, base: str) -> dict[str, Any]:
    url = f"{base}/api/v1/queries/run"
    body = {
        "scope": {"resource": "assets"},
        "select": {"fields": ["asset_id"]},
        "where": None,
        "sort": [{"field": "created_at", "direction": "desc"}],
        "page": {"page": page, "page_size": PAGE_SIZE},
    }
    code, data = http_json("POST", url, token, body)
    if code != 200:
        raise RuntimeError(f"queries/run page={page} status={code} body={data}")
    assert isinstance(data, dict)
    return data


def iter_all_asset_ids(token: str, max_assets: int, base: str) -> Iterator[str]:
    """Yield asset_id strings until pages exhausted or max_assets reached (0 = no cap)."""
    page = 1
    yielded = 0
    cap = max_assets if max_assets > 0 else None
    while True:
        j = query_page(token, page, base)
        items = j.get("items") or []
        if not items:
            break
        for it in items:
            aid = it.get("asset_id")
            if not isinstance(aid, str) or len(aid) != 8:
                continue
            yield aid
            yielded += 1
            if cap is not None and yielded >= cap:
                return
        page += 1
        if page > 100_000:
            break


def commit_delivery(
    base: str,
    token: str,
    asset_ids: list[str],
    meta: dict[str, str],
    note_obj: dict[str, Any],
) -> tuple[int, Any]:
    url = f"{base}/api/v1/deliveries"
    body = {
        "asset_ids": asset_ids,
        "customer_id": meta["customer_id"],
        "contract_id": meta["contract_id"],
        "note": json.dumps(note_obj, ensure_ascii=False)[:4000],
        "owner": meta["owner"],
    }
    idem = str(uuid.uuid4())
    return http_json(
        "POST",
        url,
        token,
        body,
        extra_headers={"Idempotency-Key": idem},
    )


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default=BASE, help="API base (no trailing /api)")
    ap.add_argument(
        "--max-assets",
        type=int,
        default=0,
        help="Stop after this many assets (0 = no cap, entire catalog)",
    )
    ap.add_argument("--min-chunk", type=int, default=10)
    ap.add_argument("--max-chunk", type=int, default=95)
    ap.add_argument("--sleep", type=float, default=0.05, help="Seconds between commits")
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--token", default="", help="X-Databrew-Token (else env DATABREW_TOKEN)")
    args = ap.parse_args()

    base = args.base.rstrip("/")

    token = (args.token or "").strip() or __import__("os").environ.get("DATABREW_TOKEN", "").strip()
    if not token:
        print("ERROR: set DATABREW_TOKEN or pass --token", file=sys.stderr)
        return 2

    lo = max(1, min(args.min_chunk, args.max_chunk))
    hi = max(lo, args.max_chunk)

    random.seed()
    buf: list[str] = []
    wave_idx = 0
    chunk_idx = 0
    assets_committed = 0
    deliveries_ok = 0
    deliveries_err = 0
    t0 = time.time()

    def flush_buffer(force_all: bool) -> None:
        nonlocal buf, wave_idx, chunk_idx, assets_committed, deliveries_ok, deliveries_err
        while buf and (force_all or len(buf) >= lo):
            take = len(buf) if force_all else random.randint(lo, min(hi, len(buf)))
            chunk = buf[:take]
            buf = buf[take:]
            meta = WAVES[wave_idx % len(WAVES)]
            note_obj = {
                "sim": "bulk_deliveries_cloudrun",
                "wave_idx": wave_idx,
                "chunk_idx": chunk_idx,
                "n_assets": len(chunk),
                "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                "checksum_style": uuid.uuid4().hex[:12],
            }
            if args.dry_run:
                deliveries_ok += 1
            else:
                code, data = commit_delivery(base, token, chunk, meta, note_obj)
                if code == 201:
                    deliveries_ok += 1
                else:
                    deliveries_err += 1
                    print(f"FAIL chunk {chunk_idx} code={code} body={data!s}", file=sys.stderr)
            assets_committed += len(chunk)
            chunk_idx += 1
            wave_idx += 1
            if chunk_idx % 25 == 0:
                elapsed = time.time() - t0
                print(
                    f"progress deliveries_ok={deliveries_ok} err={deliveries_err} "
                    f"assets_committed={assets_committed} elapsed={elapsed:.1f}s",
                    flush=True,
                )
            if args.sleep > 0:
                time.sleep(args.sleep)

    print(
        f"Streaming assets (max_assets={args.max_assets or 'unlimited'}) chunk [{lo},{hi}] dry_run={args.dry_run}",
        flush=True,
    )
    for aid in iter_all_asset_ids(token, args.max_assets, base):
        buf.append(aid)
        flush_buffer(force_all=False)

    flush_buffer(force_all=True)

    print(
        f"DONE deliveries_ok={deliveries_ok} deliveries_err={deliveries_err} "
        f"assets_in_deliveries={assets_committed} elapsed={time.time()-t0:.1f}s"
    )
    return 0 if deliveries_err == 0 else 1


if __name__ == "__main__":
    sys.exit(main())

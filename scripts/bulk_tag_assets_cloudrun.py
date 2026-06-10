#!/usr/bin/env python3
"""
Bulk-tag assets on Cloud Run dev using only public API:
  POST /api/v1/queries/run  — collect asset_id pages (page_size max 200)
  PATCH /api/v1/assets/{id} — body `{"tags": {...}}` updates several keys in one request

Usage:
  export DATABREW_TOKEN=...   # or pass --token
  python3 scripts/bulk_tag_assets_cloudrun.py [--dry-run] [--workers 24]

Requires: network to Cloud Run; token with same rights as X-Databrew-Token.
"""
from __future__ import annotations

import argparse
import http.client
import json
import random
import ssl
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Any

BASE = "https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app"
TARGET_IDS = 10_000
PAGE_SIZE = 200

PRIORITY = ["critical", "high", "medium", "low"]
QUALITY = ["excellent", "good", "acceptable", "poor", "unusable"]
SCENE = ["indoor", "outdoor", "warehouse", "office", "factory"]

TAG_CHOICES: list[tuple[str, Any]] = [
    ("priority", PRIORITY),
    ("quality", QUALITY),
    ("scene", SCENE),
    ("task", None),  # free string
    ("batch", None),
    ("notes", None),
]


def http_json(
    method: str,
    url: str,
    token: str,
    body: dict[str, Any] | None = None,
    timeout: int = 300,
    retries: int = 4,
) -> tuple[int, Any]:
    data = None
    headers = {
        "X-Databrew-Token": token,
        "Accept": "application/json",
    }
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


def query_page(token: str, page: int) -> dict[str, Any]:
    url = f"{BASE}/api/v1/queries/run"
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


def collect_asset_ids(token: str, want: int) -> list[str]:
    ids: list[str] = []
    seen: set[str] = set()
    page = 1
    while len(ids) < want:
        j = query_page(token, page)
        items = j.get("items") or []
        if not items:
            break
        for it in items:
            aid = it.get("asset_id")
            if isinstance(aid, str) and aid and aid not in seen:
                seen.add(aid)
                ids.append(aid)
                if len(ids) >= want:
                    return ids
        page += 1
        if page > 50_000:
            break
    return ids


def random_tag_payload() -> list[tuple[str, str]]:
    n = random.randint(1, 4)
    picks = random.sample(TAG_CHOICES, n)
    out: list[tuple[str, str]] = []
    for key, pool in picks:
        if pool is None:
            if key == "notes":
                out.append((key, f"bulk-auto {random.randint(0, 10**6)}"[:500]))
            else:
                out.append((key, f"st-{random.randint(0, 99999)}"))
        else:
            out.append((key, random.choice(pool)))
    return out


def upsert_tags_for_asset(token: str, asset_id: str, dry: bool) -> tuple[str, int, str | None]:
    pairs = random_tag_payload()
    if dry:
        return asset_id, 0, None
    tags = dict(pairs)
    url = f"{BASE}/api/v1/assets/{urllib.parse.quote(asset_id, safe='')}"
    code, data = http_json("PATCH", url, token, {"tags": tags})
    if code == 200:
        return asset_id, len(tags), None
    if code in (429, 503):
        time.sleep(0.5 + random.random())
        code2, data2 = http_json("PATCH", url, token, {"tags": tags})
        if code2 == 200:
            return asset_id, len(tags), None
        return asset_id, 0, f"{code2} {data2}"
    return asset_id, 0, f"{code} {data}"


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--workers", type=int, default=32)
    ap.add_argument("--max-assets", type=int, default=TARGET_IDS, help="number of assets to sample")
    ap.add_argument("--token", default="", help="X-Databrew-Token (else env DATABREW_TOKEN)")
    args = ap.parse_args()

    token = (args.token or "").strip() or __import__("os").environ.get("DATABREW_TOKEN", "").strip()
    if not token:
        print("ERROR: set DATABREW_TOKEN or pass --token", file=sys.stderr)
        return 2

    random.seed()
    want = max(1, args.max_assets)
    print("Collecting asset IDs (random page order)…", flush=True)
    ids = collect_asset_ids(token, want)
    print(f"Collected {len(ids)} asset_ids (requested {want})", flush=True)
    if len(ids) < want:
        print("WARNING: fewer IDs than requested; continuing with available set.", flush=True)

    if args.dry_run:
        print("Dry-run: no PATCH issued.")
        return 0

    errors = 0
    posts_ok = 0
    t0 = time.time()
    with ThreadPoolExecutor(max_workers=max(1, args.workers)) as ex:
        futs = {
            ex.submit(upsert_tags_for_asset, token, aid, False): aid
            for aid in ids
        }
        done = 0
        for fut in as_completed(futs):
            aid, ok, err = fut.result()
            posts_ok += ok
            if err:
                errors += 1
                print(f"FAIL {aid}: {err}", file=sys.stderr)
            done += 1
            if done % 500 == 0 or done == len(futs):
                elapsed = time.time() - t0
                print(
                    f"progress {done}/{len(futs)} tag_keys_written={posts_ok} errors={errors} elapsed={elapsed:.1f}s",
                    flush=True,
                )

    print(f"DONE assets={len(ids)} tag_keys_written={posts_ok} asset_errors={errors} elapsed={time.time()-t0:.1f}s")
    return 0 if errors == 0 else 1


if __name__ == "__main__":
    sys.exit(main())

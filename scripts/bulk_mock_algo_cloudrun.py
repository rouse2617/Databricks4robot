#!/usr/bin/env python3
"""
Drive algorithm lifecycle on Cloud Run dev for many assets (mock / load-test).

Uses only public API:
  POST /api/v1/queries/run          — sample asset_id pages
  POST /api/v1/assets/{id}/algo/{algo_key}/reset  — ok/failed → pending
  POST /api/v1/assets/{id}/algo/{algo_key}/start  — body {"method": "..."}
  POST /api/v1/assets/{id}/algo/{algo_key}/finish — status ok|failed

Per asset: pick one registered algo with no upstream deps, then try
reset → start → finish (ok for env_analysis with empty registry output reqs,
otherwise failed with a reason).

  export GRACE_TOKEN=...
  python3 scripts/bulk_mock_algo_cloudrun.py [--workers 24] [--dry-run]

See backend/config/algo_registry.yaml for valid algo_key strings.
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
DEFAULT_TARGET_IDS = 10_000
PAGE_SIZE = 200

# Algorithms with depends_on: [] — Start is not blocked on deps.
ALGO_KEYS = [
    "hand_tracking@1.2.0",
    "head_tracking@1.0.0",
    "body_tracking@1.0.0",
    "deface@2.0.0",
    "env_analysis@1.0.0",
]

ENV_ANALYSIS_KEY = "env_analysis@1.0.0"


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
        "X-Grace-Token": token,
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
    """Walk pages 1..N in order until `want` ids (stable, fewer heavy random ES pages)."""
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


def algo_paths(asset_id: str, algo_key: str) -> tuple[str, str, str]:
    aidq = urllib.parse.quote(asset_id, safe="")
    aq = urllib.parse.quote(algo_key, safe="")
    root = f"{BASE}/api/v1/assets/{aidq}/algo/{aq}"
    return f"{root}/reset", f"{root}/start", f"{root}/finish"


def mock_algo_for_asset(token: str, asset_id: str, dry: bool) -> tuple[str, bool, str | None]:
    algo = random.choice(ALGO_KEYS)
    reset_u, start_u, finish_u = algo_paths(asset_id, algo)
    if dry:
        return asset_id, True, None

    # ok/failed → pending (ignored if not applicable)
    http_json("POST", reset_u, token, {})

    code, body = http_json("POST", start_u, token, {"method": "bulk-mock"})
    if code != 200:
        if code == 409:
            # running: finish failed to clear; or ok/failed without reset taking — try finish failed
            c2, _ = http_json(
                "POST",
                finish_u,
                token,
                {"status": "failed", "reason": "bulk-mock-clear-running"},
            )
            if c2 == 200:
                http_json("POST", reset_u, token, {})
                code, body = http_json("POST", start_u, token, {"method": "bulk-mock"})
        if code != 200:
            return asset_id, False, f"start {code} {body}"

    # running → finish
    if algo == ENV_ANALYSIS_KEY and random.random() < 0.45:
        fin = {"status": "ok"}
    else:
        fin = {"status": "failed", "reason": f"bulk-mock-{random.randint(0, 999999)}"}

    c3, b3 = http_json("POST", finish_u, token, fin)
    if c3 != 200:
        return asset_id, False, f"finish {c3} {b3}"
    return asset_id, True, None


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--workers", type=int, default=20)
    ap.add_argument("--max-assets", type=int, default=DEFAULT_TARGET_IDS, help="cap sampled assets")
    ap.add_argument("--token", default="", help="X-Grace-Token (else env GRACE_TOKEN)")
    args = ap.parse_args()

    token = (args.token or "").strip() or __import__("os").environ.get("GRACE_TOKEN", "").strip()
    if not token:
        print("ERROR: set GRACE_TOKEN or pass --token", file=sys.stderr)
        return 2

    random.seed()
    print("Collecting asset IDs…", flush=True)
    want = max(1, args.max_assets)
    ids = collect_asset_ids(token, want)
    print(f"Collected {len(ids)} asset_ids (requested {want})", flush=True)

    if args.dry_run:
        print("Dry-run: no algo POSTs.")
        return 0

    ok_n = 0
    err_n = 0
    t0 = time.time()
    with ThreadPoolExecutor(max_workers=max(1, args.workers)) as ex:
        futs = {ex.submit(mock_algo_for_asset, token, aid, False): aid for aid in ids}
        done = 0
        for fut in as_completed(futs):
            aid, ok, err = fut.result()
            if ok:
                ok_n += 1
            else:
                err_n += 1
                print(f"FAIL {aid}: {err}", file=sys.stderr)
            done += 1
            if done % 500 == 0 or done == len(futs):
                print(
                    f"progress {done}/{len(futs)} ok={ok_n} err={err_n} elapsed={time.time()-t0:.1f}s",
                    flush=True,
                )

    print(f"DONE assets={len(ids)} ok={ok_n} err={err_n} elapsed={time.time()-t0:.1f}s")
    return 0 if err_n == 0 else 1


if __name__ == "__main__":
    sys.exit(main())

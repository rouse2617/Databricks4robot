#!/usr/bin/env python3
"""
Bulk-mock Action + Eval APIs on Cloud Run dev.

  - POST /api/v1/queries/run  (filter asset_type=segment, paginate)
  - POST /api/v1/assets/{id}/eval-results
  - POST /api/v1/assets/{id}/actions

Per asset: one eval row (queryable metrics from metric_registry) + one action
(primary_label from action_label_registry, time window inside seg bounds).

  export DATABREW_TOKEN=...
  python3 scripts/bulk_action_eval_cloudrun.py [--max-assets 25000] [--workers 20]
    [--page-size 100] [--in-flight-mult 4]

Uses streaming: fetches queries/run pages while POSTing eval+action with bounded
concurrency (no pre-loading all rows into RAM). Each queries page logs to stdout.
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
from concurrent.futures import FIRST_COMPLETED, ThreadPoolExecutor, wait
from typing import Any, Iterator

BASE = "https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app"
DEFAULT_PAGE_SIZE = 100

PRIMARY_LABELS = [
    "pickup",
    "place",
    "push",
    "pull",
    "rotate",
    "inspect",
    "clean",
    "adjust",
]
EXTRA_LABELS = [
    "left_hand",
    "right_hand",
    "tool",
    "object",
    "success",
    "retry",
]

# Subset of metric_registry.yaml (queryable: true, numeric-friendly)
EVAL_METRIC_KEYS = [
    "good_frames_ratio",
    "hand_good_frames",
    "total_good_frames",
    "total_video_frames",
    "weighted_avg_convex_hull_volume",
    "qc_score",
    "blur_ratio",
    "occlusion_ratio",
]


def http_json(
    method: str,
    url: str,
    token: str,
    body: dict[str, Any] | None = None,
    extra_headers: dict[str, str] | None = None,
    timeout: int = 180,
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


def query_segment_page(token: str, page: int, base: str, page_size: int) -> dict[str, Any]:
    url = f"{base}/api/v1/queries/run"
    body = {
        "scope": {"resource": "assets"},
        "select": {"fields": ["asset_id"]},
        "where": {"pred": {"field": "asset_type", "op": "eq", "value": "segment"}},
        "sort": [{"field": "created_at", "direction": "desc"}],
        "page": {"page": page, "page_size": page_size},
    }
    t0 = time.time()
    code, data = http_json("POST", url, token, body, timeout=240)
    if code != 200:
        raise RuntimeError(f"queries/run page={page} status={code} body={data}")
    assert isinstance(data, dict)
    elapsed = time.time() - t0
    n = len(data.get("items") or [])
    total = data.get("total", "?")
    print(
        f"[collect] page={page} page_size={page_size} rows={n} total={total} http_wall={elapsed:.2f}s",
        flush=True,
    )
    return data


def iter_segment_asset_items(
    token: str, base: str, page_size: int
) -> Iterator[dict[str, Any]]:
    """Yield full asset dicts from queries/run (segment filter), newest first."""
    page = 1
    while page <= 100_000:
        j = query_segment_page(token, page, base, page_size)
        items = j.get("items") or []
        if not items:
            break
        for it in items:
            if not isinstance(it, dict):
                continue
            aid = it.get("asset_id")
            if isinstance(aid, str) and len(aid) == 8 and aid.isalnum():
                yield it
        page += 1


def ns_window_for_asset(item: dict[str, Any]) -> tuple[int, int]:
    st = item.get("start_timestamp_ns")
    en = item.get("end_timestamp_ns")
    if isinstance(st, int) and isinstance(en, int) and st < en:
        span = en - st
        # keep a short window (>=1ns) fully inside [st, en]
        w = min(max(span // 200, 1), span)
        offset = random.randint(0, max(0, span - w))
        a0 = st + offset
        a1 = a0 + max(w - 1, 0)
        if a1 > en:
            a1 = en
        if a0 >= a1:
            a1 = a0 + 1 if a0 + 1 <= en else a0
        return a0, a1
    return 0, 1000


def build_eval_payload() -> dict[str, Any]:
    keys = random.sample(EVAL_METRIC_KEYS, k=random.randint(3, min(6, len(EVAL_METRIC_KEYS))))
    p: dict[str, Any] = {}
    for k in keys:
        if k in ("hand_good_frames", "total_good_frames", "total_video_frames"):
            p[k] = random.randint(10, 500_000)
        else:
            p[k] = round(random.uniform(0.0, 1.0), 6)
    return p


def post_eval(base: str, token: str, asset_id: str) -> tuple[int, Any]:
    url = f"{base}/api/v1/assets/{urllib.parse.quote(asset_id, safe='')}/eval-results"
    body = {
        "eval_name": "bulk_sim_eval",
        "eval_version": "1.0.0",
        "parameter_version": "p-" + uuid.uuid4().hex[:8],
        "run_id": "run-" + uuid.uuid4().hex,
        "status": "ok",
        "target_type": "segment",
        "target_id": "",
        "result_payload": build_eval_payload(),
        "source_type": "algo",
        "source_name": "bulk-action-eval-sim",
        "source_version": "1.0.0",
    }
    return http_json("POST", url, token, body, timeout=120)


def post_action(base: str, token: str, asset_id: str, item: dict[str, Any]) -> tuple[int, Any]:
    s0, s1 = ns_window_for_asset(item)
    url = f"{base}/api/v1/assets/{urllib.parse.quote(asset_id, safe='')}/actions"
    pl = random.choice(PRIMARY_LABELS)
    nlab = random.randint(1, 3)
    labels = random.sample(EXTRA_LABELS, k=min(nlab, len(EXTRA_LABELS)))
    body = {
        "start_ns": s0,
        "end_ns": s1,
        "primary_label": pl,
        "labels": labels,
        "description": f"bulk-sim action {uuid.uuid4().hex[:10]}",
        "attrs": {"sim_batch": True, "idx": random.randint(0, 999999)},
        "source_type": random.choice(["human", "algo", "rule", "system"]),
        "source_name": "bulk-action-eval-sim",
        "source_version": "1.0.0",
        "run_id": "run-" + uuid.uuid4().hex,
        "confidence": round(random.uniform(0.3, 0.99), 4),
        "external_id": "ext-" + uuid.uuid4().hex,
    }
    return http_json("POST", url, token, body, timeout=120)


def process_one(
    base: str,
    token: str,
    item: dict[str, Any],
    dry: bool,
    skip_eval: bool,
    skip_action: bool,
) -> tuple[str, str | None]:
    aid = item.get("asset_id") or ""
    if not aid:
        return "", "missing asset_id"
    if dry:
        return aid, None
    if not skip_eval:
        c, b = post_eval(base, token, aid)
        if c != 201:
            return aid, f"eval {c} {b}"
    if not skip_action:
        c2, b2 = post_action(base, token, aid, item)
        if c2 != 201:
            return aid, f"action {c2} {b2}"
    return aid, None


def run_streaming(
    base: str,
    token: str,
    want: int,
    workers: int,
    page_size: int,
    in_flight_mult: int,
    dry: bool,
    skip_eval: bool,
    skip_action: bool,
) -> tuple[int, int]:
    """Paginate + submit work concurrently (no giant in-memory list). Returns (ok, err)."""
    max_flight = max(workers * max(2, in_flight_mult), workers + 4)
    max_flight = min(max_flight, 600)

    it = iter_segment_asset_items(token, base, page_size)
    pending: set = set()
    submitted = 0
    ok = 0
    err = 0
    t0 = time.time()
    exhausted = False

    def submit_batch(ex: ThreadPoolExecutor, item: dict[str, Any]) -> None:
        nonlocal submitted
        fut = ex.submit(process_one, base, token, item, dry, skip_eval, skip_action)
        pending.add(fut)
        submitted += 1

    with ThreadPoolExecutor(max_workers=max(1, workers)) as ex:
        while True:
            while len(pending) < max_flight and submitted < want and not exhausted:
                try:
                    item = next(it)
                except StopIteration:
                    exhausted = True
                    break
                submit_batch(ex, item)

            if not pending:
                break
            done, not_done = wait(pending, return_when=FIRST_COMPLETED, timeout=120)
            pending = set(not_done)
            if not done:
                print(f"WARN: wait returned no completions in-flight={len(pending)}", flush=True)
                time.sleep(0.3)
                continue
            for fut in done:
                aid, e = fut.result()
                if e:
                    err += 1
                    print(f"FAIL {aid}: {e}", file=sys.stderr)
                else:
                    ok += 1
                n = ok + err
                if n % 500 == 0:
                    print(
                        f"progress {n}/{submitted} ok={ok} err={err} "
                        f"in_flight={len(pending)} elapsed={time.time()-t0:.1f}s",
                        flush=True,
                    )

            if exhausted and not pending:
                break

    return ok, err


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default=BASE)
    ap.add_argument("--max-assets", type=int, default=25_000)
    ap.add_argument("--workers", type=int, default=20)
    ap.add_argument("--page-size", type=int, default=DEFAULT_PAGE_SIZE, help="queries/run page size (max 200)")
    ap.add_argument(
        "--in-flight-mult",
        type=int,
        default=4,
        help="max in-flight futures ≈ workers * this (capped)",
    )
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--eval-only", action="store_true")
    ap.add_argument("--action-only", action="store_true")
    ap.add_argument("--token", default="")
    args = ap.parse_args()

    base = args.base.rstrip("/")
    token = (args.token or "").strip() or __import__("os").environ.get("DATABREW_TOKEN", "").strip()
    if not token:
        print("ERROR: set DATABREW_TOKEN or --token", file=sys.stderr)
        return 2
    if args.eval_only and args.action_only:
        print("ERROR: choose at most one of --eval-only / --action-only", file=sys.stderr)
        return 2

    skip_eval = args.action_only
    skip_action = args.eval_only

    want = max(1, args.max_assets)
    ps = max(1, min(args.page_size, 200))
    random.seed()

    print(
        f"Streaming up to {want} segment assets (page_size={ps}, workers={args.workers}, "
        f"in_flight_mult={args.in_flight_mult}) dry_run={args.dry_run}",
        flush=True,
    )

    ok, err = run_streaming(
        base,
        token,
        want,
        args.workers,
        ps,
        args.in_flight_mult,
        args.dry_run,
        skip_eval,
        skip_action,
    )

    print(f"DONE ok={ok} err={err} submitted={ok+err}")
    return 0 if err == 0 else 1


if __name__ == "__main__":
    sys.exit(main())

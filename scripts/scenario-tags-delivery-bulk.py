#!/usr/bin/env python3
"""
Large-scale API-only run of: mcap-files → assets → POST tags → queries/run → POST deliveries (+ optional GET).

All traffic is HTTP(S) to the databrew backend (same contract as scripts/scenario-tags-delivery.sh).

Default --base / $BASE: hosted **dev** Cloud Run URL (same as deploy/cloudrun/frontend-nginx.conf).
Override for local: --base http://127.0.0.1:8080

Examples:
  python3 scripts/scenario-tags-delivery-bulk.py --token dev-token --count 200 --concurrency 20

  python3 scripts/scenario-tags-delivery-bulk.py \\
    --base http://127.0.0.1:8080 --token dev-token --count 200 --concurrency 20 --es-wait 0

  # After bulk, delete created assets (extra DELETE /api/v1/assets/{id} per success)
  python3 scripts/scenario-tags-delivery-bulk.py ... --cleanup
"""
from __future__ import annotations

import argparse
import json
import os
import random
import secrets
import string
import ssl
import sys
import threading
import time
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from typing import Any


def _rand_alnum(n: int) -> str:
    alphabet = string.ascii_letters + string.digits
    return "".join(secrets.choice(alphabet) for _ in range(n))


@dataclass
class HttpResult:
    status: int
    body: bytes


class ApiClient:
    def __init__(self, base: str, token: str, timeout_sec: float) -> None:
        self.base = base.rstrip("/")
        self.token = token
        self.timeout = timeout_sec
        self._ctx = ssl.create_default_context()

    def _request(
        self,
        method: str,
        path: str,
        body: dict[str, Any] | None = None,
        extra_headers: dict[str, str] | None = None,
    ) -> HttpResult:
        url = f"{self.base}{path}"
        data = json.dumps(body).encode("utf-8") if body is not None else None
        headers = {"X-Databrew-Token": self.token}
        if data is not None:
            headers["Content-Type"] = "application/json"
        if extra_headers:
            headers.update(extra_headers)
        req = urllib.request.Request(url, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=self.timeout, context=self._ctx) as resp:
                return HttpResult(resp.status, resp.read())
        except urllib.error.HTTPError as e:
            return HttpResult(e.code, e.read() if e.fp else b"")
        except OSError as e:
            return HttpResult(0, str(e).encode("utf-8"))

    def post_json(self, path: str, body: dict[str, Any], extra_headers: dict[str, str] | None = None) -> HttpResult:
        return self._request("POST", path, body, extra_headers)

    def get(self, path: str) -> HttpResult:
        return self._request("GET", path, None, {"Content-Type": "application/json"})

    def delete(self, path: str) -> HttpResult:
        return self._request("DELETE", path, None, {"Content-Type": "application/json"})


def _json_body(res: HttpResult) -> Any:
    if not res.body:
        return None
    return json.loads(res.body.decode("utf-8"))


def run_scenario(
    api: ApiClient,
    run_index: int,
    es_wait: float,
    query_retries: int,
    query_retry_delay: float,
    verify_get: bool,
    cleanup: bool,
) -> tuple[bool, str, float]:
    """
    Returns (ok, error_or_ok_label, elapsed_seconds).
    """
    t0 = time.perf_counter()
    ts = int(time.time() * 1000) + run_index
    owner = f"bulk-owner-{ts}-{_rand_alnum(6)}"
    customer_id = f"bulk-cust-{ts}-{_rand_alnum(4)}"
    batch_tag = f"bulk-batch-{ts}-{_rand_alnum(8)}"
    mcap_id = _rand_alnum(8)
    raw_hash = secrets.token_hex(16)
    run_label = f"bulk-{ts}"

    def fail(step: str, res: HttpResult) -> tuple[bool, str, float]:
        snippet = res.body[:400].decode("utf-8", errors="replace")
        return False, f"{step} http={res.status} body={snippet}", time.perf_counter() - t0

    # POST mcap-files
    mcap_body = {
        "mcap_file_id": mcap_id,
        "gcs_path": f"gs://scenario-bulk/path/{run_label}.mcap",
        "raw_hash_md5": raw_hash,
        "ingest_state": "summarized",
        "size_bytes": 1048576,
        "file_duration_ms": 60000,
        "start_timestamp_ns": 1700000000000000000,
        "end_timestamp_ns": 1700000060000000000,
        "channel_count": 12,
        "chunk_count": 5,
        "vendor_id": "bulk",
        "device_id": "bulk-1",
        "scene_id": "indoor",
        "owner": owner,
    }
    r = api.post_json("/api/v1/mcap-files", mcap_body)
    if r.status < 200 or r.status >= 300:
        return fail("mcap-files", r)

    asset_body = {
        "mcap_file_id": mcap_id,
        "start_timestamp_ns": 1700000000000000000,
        "end_timestamp_ns": 1700000060000000000,
        "reviewer": "bulk",
        "owner": owner,
        "type": "task_demo",
        "tags": {"priority": "low", "quality": "good", "scene": "indoor"},
    }
    r = api.post_json("/api/v1/assets", asset_body)
    if r.status < 200 or r.status >= 300:
        return fail("assets", r)
    asset_id = _json_body(r)["asset_id"]

    tag_body = {"key": "batch", "value": batch_tag}
    r = api.post_json(f"/api/v1/assets/{asset_id}/tags", tag_body)
    if r.status < 200 or r.status >= 300:
        return fail("tags", r)

    if es_wait > 0:
        time.sleep(es_wait)

    query_body = {
        "schema_version": "v1",
        "mode": "structured",
        "scope": {"resource": "assets"},
        "select": {"fields": ["asset_id", "owner", "tags", "lifecycle_state"]},
        "where": {
            "and": [
                {"pred": {"field": "owner", "op": "eq", "value": owner}},
                {"pred": {"field": "tags_flat.batch", "op": "eq", "value": batch_tag}},
            ]
        },
        "page": {"page": 1, "page_size": 20},
    }

    found = False
    last_r = r
    for attempt in range(query_retries + 1):
        last_r = api.post_json("/api/v1/queries/run", query_body)
        if last_r.status < 200 or last_r.status >= 300:
            if attempt < query_retries:
                time.sleep(query_retry_delay)
                continue
            return fail("queries/run", last_r)
        data = _json_body(last_r)
        items = data.get("items") or []
        if any((it.get("asset_id") == asset_id) for it in items):
            found = True
            break
        if attempt < query_retries:
            time.sleep(query_retry_delay)
    if not found:
        return fail(
            "queries/run(no_hit)",
            last_r,
        )

    idem = f"bulk-delivery-{run_label}-{_rand_alnum(6)}"
    delivery_body = {
        "asset_ids": [asset_id],
        "customer_id": customer_id,
        "contract_id": f"contract-{run_label}",
        "note": "bulk scenario tag→query→delivery",
        "owner": owner,
    }
    r = api.post_json(
        "/api/v1/deliveries",
        delivery_body,
        {"Idempotency-Key": idem},
    )
    if r.status < 200 or r.status >= 300:
        return fail("deliveries", r)
    delivery_id = _json_body(r)["delivery_id"]

    if verify_get:
        r = api.get(f"/api/v1/deliveries/{delivery_id}")
        if r.status < 200 or r.status >= 300:
            return fail("get_delivery", r)
        r = api.get(f"/api/v1/customers/{customer_id}/deliveries?page=1&page_size=20")
        if r.status < 200 or r.status >= 300:
            return fail("get_customer_deliveries", r)

    if cleanup:
        r = api.delete(f"/api/v1/assets/{asset_id}")
        if r.status < 200 or r.status >= 300:
            return fail("delete_asset", r)

    return True, "ok", time.perf_counter() - t0


def main() -> int:
    p = argparse.ArgumentParser(description="Bulk API scenario: tag → query → delivery")
    _dev_default = "https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app"
    p.add_argument(
        "--base",
        default=os.environ.get("BASE", _dev_default),
        help="Backend base URL (default: hosted dev Cloud Run; override for local)",
    )
    p.add_argument("--token", default=os.environ.get("TOKEN", "dev-token"), help="X-Databrew-Token value")
    p.add_argument("--count", type=int, default=100, help="Number of full scenarios to run")
    p.add_argument("--concurrency", type=int, default=20, help="Parallel workers (ThreadPoolExecutor)")
    p.add_argument("--timeout", type=float, default=120.0, help="Per-request socket timeout (seconds)")
    p.add_argument(
        "--es-wait",
        type=float,
        default=2.0,
        help="Sleep after POST tags before queries/run (default 2s for hosted dev ES lag; use 0 for local)",
    )
    p.add_argument("--query-retries", type=int, default=5, help="Retries if queries/run misses asset_id")
    p.add_argument("--query-retry-delay", type=float, default=0.5, help="Seconds between query retries")
    p.add_argument("--verify-get", action="store_true", help="Also GET delivery + customer deliveries per scenario")
    p.add_argument("--cleanup", action="store_true", help="DELETE asset after each successful scenario")
    p.add_argument("--seed", type=int, default=None, help="Optional random seed (debug)")
    args = p.parse_args()

    if args.seed is not None:
        random.seed(args.seed)

    if args.count < 1:
        print("count must be >= 1", file=sys.stderr)
        return 2
    if args.concurrency < 1:
        print("concurrency must be >= 1", file=sys.stderr)
        return 2

    api = ApiClient(args.base, args.token, args.timeout)
    print(
        f"bulk scenario: base={args.base} count={args.count} "
        f"concurrency={args.concurrency} es_wait={args.es_wait} verify_get={args.verify_get} cleanup={args.cleanup}",
        flush=True,
    )

    lock = threading.Lock()
    ok_n = 0
    fail_n = 0
    latencies: list[float] = []
    errors_sample: list[str] = []

    wall0 = time.perf_counter()

    def task(idx: int) -> tuple[bool, str, float]:
        return run_scenario(
            api,
            idx,
            args.es_wait,
            args.query_retries,
            args.query_retry_delay,
            args.verify_get,
            args.cleanup,
        )

    with ThreadPoolExecutor(max_workers=args.concurrency) as ex:
        futs = {ex.submit(task, i): i for i in range(args.count)}
        for fut in as_completed(futs):
            ok, msg, elapsed = fut.result()
            with lock:
                if ok:
                    ok_n += 1
                    latencies.append(elapsed)
                else:
                    fail_n += 1
                    if len(errors_sample) < 15:
                        errors_sample.append(msg)

    wall = time.perf_counter() - wall0
    latencies.sort()
    nlat = len(latencies)

    def pct(p: float) -> float:
        if nlat == 0:
            return 0.0
        return latencies[int((nlat - 1) * p)]

    print("", flush=True)
    print(f"wall_seconds={wall:.3f} scenarios={args.count} ok={ok_n} fail={fail_n}", flush=True)
    if nlat:
        print(
            f"latency_s per successful scenario: p50={pct(0.50):.3f} "
            f"p95={pct(0.95):.3f} p99={pct(0.99):.3f} max={latencies[-1]:.3f}",
            flush=True,
        )
        print(f"throughput_scenarios_per_s={ok_n / wall:.3f} (ok only / wall)", flush=True)
    if errors_sample:
        print("sample_errors:", flush=True)
        for e in errors_sample:
            print(f"  - {e[:500]}", flush=True)

    return 0 if fail_n == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())

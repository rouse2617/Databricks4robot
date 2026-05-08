#!/usr/bin/env python3
"""
Seed rich mock data via HTTP (local Compose / staging).

Per asset (rotating asset_type over segment / clip / frame_set / derived_asset):
  - POST /api/v1/mcap-files (optional explicit mcap_file_id or server-generated 8-char id)
  - POST /api/v1/assets + tags (+ optional task/notes tags)
  - POST algo env_analysis finish; optionally hand_tracking@1.2.0 (about 40% of assets by default)
  - POST multiple /eval-results rows (--evals-per-asset, distinct eval_name/version draws)

Defaults ~100 assets. Use --total / --evals-per-asset / --hand-tracking-pct for heavier datasets.
POST /assets/:id/actions only allows parent asset_type=segment; clip/frame_set/derived_asset rows are for list/filter/UI coverage.

Requires: pip install httpx
"""

from __future__ import annotations

import argparse
import concurrent.futures
import itertools
import json
import random
import secrets
import string
import threading
import time
import uuid
from dataclasses import dataclass
from typing import Optional, Set

import httpx

BASE = "http://127.0.0.1:8080"
TOKEN = "dev-token"
# Safe defaults for CI / laptop (avoid accidental 10k run)
DEFAULT_TOTAL = 100
DEFAULT_WORKERS = 8
MAX_RETRIES = 5
RETRY_BACKOFF_SEC = 0.2

# Structural `assets.asset_type` values (docs/review/sql.md §4.2). Rotated by seed index so batches cover all kinds.
STRUCTURAL_ASSET_TYPES = ("segment", "clip", "frame_set", "derived_asset")

SCENES = [
    "highway",
    "urban",
    "parking_lot",
    "tunnel",
    "intersection",
    "rural",
    "factory_floor",
    "warehouse",
    "campus",
    "stadium",
]
WEATHER = ["sunny", "cloudy", "rainy", "foggy", "night", "dusk", "dawn", "snowy", "overcast", "partly_cloudy"]
LOCATIONS = ["beijing", "shanghai", "shenzhen", "chengdu", "wuhan", "hangzhou", "nanjing", "guangzhou", "xi_an", "tianjin"]
VEHICLES = ["sedan_a", "sedan_b", "suv_x", "truck_t", "van_v", "pickup_p"]
CAMERAS = ["front_wide", "front_tele", "rear_wide", "left_fisheye", "right_fisheye", "top_panoramic"]
OWNERS = ["team_alpha", "team_beta", "team_gamma", "team_delta", "team_epsilon", "team_zeta"]
REVIEWERS = ["alice", "bob", "carol", "dave", "eve", "frank", "grace"]
# Must match DB chk_lifecycle_state (see migrations/009_lifecycle_state_check.sql)
LIFECYCLE = ["created", "processing", "ready", "delivered", "archived", "superseded", "failed", "rejected"]
QUALITY_VALS = ["excellent", "good", "acceptable", "poor", "unusable"]
PRIORITY_VALS = ["critical", "high", "medium", "low"]
SCENE_TAG_VALS = ["indoor", "outdoor", "warehouse", "office", "factory"]
SENSOR_MODES = ["360_surround", "front_only", "stereo", "lidar_cam_fusion"]
ENVS = ["kitchen", "outdoor", "warehouse", "office", "factory"]

EVAL_NAMES = ["hand_tracking_qc", "body_qc", "env_qc", "scene_classifier"]
EVAL_VERSIONS = ["v1.0", "v1.1", "v2.0", "v2.1"]

# Precomputed (eval_name, eval_version) pairs for multi-eval rows without tiny Cartesian explosion at runtime.
_EVAL_COMBOS: list[tuple[str, str]] = list(itertools.product(EVAL_NAMES, EVAL_VERSIONS))

ALGO_ENV = "env_analysis@1.0.0"
ALGO_HAND = "hand_tracking@1.2.0"


@dataclass
class SeedConfig:
    """Per-asset richness: multiple eval rows, optional second algo, extended tags."""

    evals_per_asset: int = 2
    hand_tracking_pct: int = 40
    rich_tags: bool = True
    bias_metrics_search: bool = True
    include_all_ids: bool = False

T0_NS = 1_640_000_000_000_000_000
ONE_DAY_NS = 86_400_000_000_000

# Matches backend/internal/id/assetid.go — same as asset_id / mcap_file_id.
_SHORT_ID_ALPHABET = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"


def generate_mcap_file_id() -> str:
    return "".join(secrets.choice(_SHORT_ID_ALPHABET) for _ in range(8))


def rand_str(n: int = 8) -> str:
    return "".join(random.choices(string.ascii_lowercase + string.digits, k=n))


# Set in main() so repeated runs do not reuse (mcap_id, start, end) collision windows.
_RUN_TIME_SALT_NS = 0


def make_mcap_payload(mcap_id: str, t_start: int, t_end: int, seq: int) -> dict:
    """Fields aligned with POST /api/v1/mcap-files (see handler CreateFile)."""
    dur_ms = max(1, int((t_end - t_start) / 1_000_000))
    # Unique MD5-shaped hex (uq_mcap_files_hash_md5); avoid random collisions under load.
    hash_hex = uuid.uuid4().hex
    return {
        "mcap_file_id": mcap_id,
        "gcs_path": f"gs://seed-rich/{seq:06d}/{mcap_id}.mcap",
        "raw_hash_md5": hash_hex,
        "ingest_state": "summarized",
        "size_bytes": random.randint(50_000, 80_000_000),
        "file_duration_ms": dur_ms,
        "start_timestamp_ns": t_start,
        "end_timestamp_ns": t_end,
        "channel_count": random.randint(3, 12),
        "chunk_count": random.randint(1, 24),
        "vendor_id": "seed_rich",
        "device_id": f"dev-{rand_str(4)}",
        "scene_id": random.choice(SCENES),
        "owner": random.choice(OWNERS),
    }


def make_asset_payload(mcap_id: str, t_start: int, t_end: int, structural_type: str) -> dict:
    """structural_type is assets.asset_type (segment | clip | frame_set | derived_asset)."""
    scene = random.choice(SCENES)
    env = random.choice(ENVS)
    return {
        "mcap_file_id": mcap_id,
        "start_timestamp_ns": t_start,
        "end_timestamp_ns": t_end,
        "owner": random.choice(OWNERS),
        "reviewer": random.choice(REVIEWERS),
        "lifecycle_state": random.choice(LIFECYCLE),
        "asset_type": structural_type,
        # Legacy dual-write field (Phase 0); kept stable so filters that still read `type` behave predictably.
        "type": "task_demo",
        "metadata": {
            "scene": scene,
            "env": env,
            "weather": random.choice(WEATHER),
            "location": random.choice(LOCATIONS),
            "vehicle_id": random.choice(VEHICLES),
            "sensor_mode": random.choice(SENSOR_MODES),
            "camera_config": random.choice(CAMERAS),
            "speed_kmh": round(random.uniform(0, 130), 1),
            "batch_id": f"batch-{random.randint(1, 500):04d}",
            "seed_structural_kind": structural_type,
        },
    }


def make_tags(rich: bool) -> list[dict]:
    """Tag keys aligned with backend/config/tag_registry.yaml enums where applicable."""
    base = [
        {"key": "quality", "value": random.choice(QUALITY_VALS)},
        {"key": "priority", "value": random.choice(PRIORITY_VALS)},
        {"key": "scene", "value": random.choice(SCENE_TAG_VALS)},
        {"key": "batch", "value": f"B-{random.randint(1, 9999):04d}"},
    ]
    if not rich:
        return base
    extra = [
        {
            "key": "task",
            "value": random.choice(["pick_place", "navigation", "assembly", "qc_scan", "dock_charge"]),
        },
        {"key": "notes", "value": f"seed_rich idx notes len={random.randint(12, 180)}"[:500]},
    ]
    return base + extra


def make_eval_payload(eval_name: str, eval_ver: str, *, biased_high_quality: bool = False) -> dict:
    if biased_high_quality:
        gfr = round(random.uniform(0.35, 1.0), 4)
    else:
        gfr = round(random.uniform(0.0, 1.0), 4)
    tvf = random.randint(300, 3600)
    tgf = int(tvf * gfr)
    hgf = random.randint(0, tgf)
    return {
        "eval_name": eval_name,
        "eval_version": eval_ver,
        "parameter_version": f"p{random.randint(1, 20)}",
        "run_id": f"run-{rand_str(10)}",
        "status": "ok",
        "target_type": "segment",
        "target_id": f"seg-{rand_str(6)}",
        "source_type": "algo",
        "source_name": eval_name.split("_")[0],
        "result_payload": {
            "good_frames_ratio": gfr,
            "hand_good_frames": hgf,
            "total_good_frames": tgf,
            "total_video_frames": tvf,
            "weighted_avg_convex_hull_volume": round(random.uniform(0, 5000), 2),
            "qc_score": round(random.uniform(0, 1), 4),
            "blur_ratio": round(random.uniform(0, 0.3), 4),
            "occlusion_ratio": round(random.uniform(0, 0.5), 4),
            "raw_confidence_list_len": random.randint(0, tvf),
        },
    }


_tls = threading.local()
_counters_lock = threading.Lock()
counters = {"ok": 0, "err": 0}
error_buckets: dict[str, int] = {}
type_counts: dict[str, int] = {}


def get_client(base_url: str, token: str) -> httpx.Client:
    c = getattr(_tls, "client", None)
    if c is None:
        c = httpx.Client(
            base_url=base_url,
            headers={"X-Grace-Token": token, "Content-Type": "application/json"},
            timeout=httpx.Timeout(30.0, connect=5.0),
            limits=httpx.Limits(max_connections=50, max_keepalive_connections=20),
        )
        _tls.client = c
    return c


def bump_error(key: str) -> None:
    with _counters_lock:
        error_buckets[key] = error_buckets.get(key, 0) + 1


def request_json_retry(
    client: httpx.Client,
    method: str,
    path: str,
    expected_statuses: Set[int],
    payload: Optional[dict] = None,
):
    last_error = ""
    for i in range(MAX_RETRIES):
        try:
            resp = client.request(method, path, json=payload)
            if resp.status_code in expected_statuses:
                if resp.text:
                    return resp.json()
                return {}
            if resp.status_code in (429, 500, 502, 503, 504):
                last_error = f"{path}:status={resp.status_code}"
                time.sleep(RETRY_BACKOFF_SEC * (2**i))
                continue
            bump_error(f"{path}:status={resp.status_code}:{resp.text[:200]}")
            return None
        except Exception as ex:  # noqa: BLE001
            last_error = f"{path}:ex={type(ex).__name__}"
            time.sleep(RETRY_BACKOFF_SEC * (2**i))
    bump_error(last_error or f"{path}:unknown")
    return None


def build_algo_finish_body(algo_key: str, run_id: str) -> dict:
    """Align with backend/config/algo_registry.yaml output requirements."""
    base = algo_key.split("@")[0]
    body: dict = {"status": "ok", "run_id": run_id}
    if base == "env_analysis":
        return body
    if base in ("hand_tracking", "head_tracking", "body_tracking"):
        body["output_uri"] = f"gs://seed-rich/algo/{base}/{rand_str(14)}.npz"
        body["result_size_bytes"] = random.randint(2048, 2_000_000)
        body["extra_fields"] = {"type": "mcap"}
        return body
    # Conservative default for any future seeded algos with uri + report_size
    body["output_uri"] = f"gs://seed-rich/algo/{base}/{rand_str(12)}.bin"
    body["result_size_bytes"] = random.randint(1024, 500_000)
    body["extra_fields"] = {"type": "mcap"}
    return body


def run_algo_pipeline(c: httpx.Client, asset_id: str, algo_key: str) -> bool:
    run_id = f"run-{rand_str(10)}"
    if request_json_retry(
        c,
        "POST",
        f"/api/v1/assets/{asset_id}/algo/{algo_key}/start",
        {200, 409},
        {"method": "k8s_job", "run_id": run_id},
    ) is None:
        return False
    finish_body = build_algo_finish_body(algo_key, run_id)
    return (
        request_json_retry(c, "POST", f"/api/v1/assets/{asset_id}/algo/{algo_key}/finish", {200}, finish_body)
        is not None
    )


def seed_one(idx: int, base_url: str, token: str, cfg: SeedConfig) -> str:
    try:
        # Spread timestamps so segment_locator almost never collides under concurrency.
        slot_ns = _RUN_TIME_SALT_NS + int(idx) * 1_000_000_000_000  # 1000s apart per idx within this run
        jitter = random.randint(0, 999_999_999)
        t_start = T0_NS + slot_ns + jitter
        dur_s = random.randint(10, 300)
        t_end = t_start + dur_s * 1_000_000_000
        structural_type = STRUCTURAL_ASSET_TYPES[idx % len(STRUCTURAL_ASSET_TYPES)]
        mcap_id = generate_mcap_file_id()
        c = get_client(base_url, token)

        mcap_p = make_mcap_payload(mcap_id, t_start, t_end, idx)
        if request_json_retry(c, "POST", "/api/v1/mcap-files", {201}, mcap_p) is None:
            with _counters_lock:
                counters["err"] += 1
            return ""

        asset_p = make_asset_payload(mcap_id, t_start, t_end, structural_type)
        asset_data = request_json_retry(c, "POST", "/api/v1/assets", {201}, asset_p)
        if asset_data is None:
            with _counters_lock:
                counters["err"] += 1
            return ""
        asset_id = asset_data["asset_id"]

        for tag in make_tags(cfg.rich_tags):
            if request_json_retry(c, "POST", f"/api/v1/assets/{asset_id}/tags", {200}, tag) is None:
                with _counters_lock:
                    counters["err"] += 1
                return ""

        if not run_algo_pipeline(c, asset_id, ALGO_ENV):
            with _counters_lock:
                counters["err"] += 1
            return ""

        if cfg.hand_tracking_pct > 0 and random.randint(1, 100) <= cfg.hand_tracking_pct:
            if not run_algo_pipeline(c, asset_id, ALGO_HAND):
                with _counters_lock:
                    counters["err"] += 1
                return ""

        combos = random.sample(_EVAL_COMBOS, k=min(cfg.evals_per_asset, len(_EVAL_COMBOS)))
        if len(combos) < cfg.evals_per_asset:
            combos.extend(random.choices(_EVAL_COMBOS, k=cfg.evals_per_asset - len(combos)))
        for ei, (ename, ever) in enumerate(combos[: cfg.evals_per_asset]):
            biased = cfg.bias_metrics_search and ei == 0
            eval_p = make_eval_payload(ename, ever, biased_high_quality=biased)
            if request_json_retry(c, "POST", f"/api/v1/assets/{asset_id}/eval-results", {201}, eval_p) is None:
                with _counters_lock:
                    counters["err"] += 1
                return ""

        with _counters_lock:
            counters["ok"] += 1
            type_counts[structural_type] = type_counts.get(structural_type, 0) + 1
        return asset_id

    except Exception as ex:  # noqa: BLE001
        bump_error(f"seed_one:{type(ex).__name__}")
        with _counters_lock:
            counters["err"] += 1
        return ""


def parse_args():
    p = argparse.ArgumentParser(description="Seed rich mock dataset (mcap + asset + tags + algo + eval metrics)")
    p.add_argument("--base", default=BASE, help="API base URL")
    p.add_argument("--token", default=TOKEN, help="X-Grace-Token")
    p.add_argument("--total", type=int, default=DEFAULT_TOTAL, help="Number of assets to create")
    p.add_argument("--workers", type=int, default=DEFAULT_WORKERS, help="Concurrency")
    p.add_argument("--sample-output", default="/tmp/seed_rich_sample.json", help="Write sample asset IDs JSON")
    p.add_argument(
        "--evals-per-asset",
        type=int,
        default=2,
        metavar="N",
        help="POST N eval-result rows per asset (distinct eval_name/version draws; default 2)",
    )
    p.add_argument(
        "--hand-tracking-pct",
        type=int,
        default=40,
        metavar="PCT",
        help="Probability 1–100 to also run hand_tracking@1.2.0 after env_analysis (default 40)",
    )
    p.add_argument("--no-rich-tags", action="store_true", help="Only 4 base tags (no task/notes)")
    p.add_argument(
        "--no-metrics-bias",
        action="store_true",
        help="Disable skewed good_frames_ratio on first eval row (full 0..1 uniform)",
    )
    p.add_argument(
        "--include-all-ids",
        action="store_true",
        help="Also write all_asset_ids in sample-output (large JSON for big --total)",
    )
    return p.parse_args()


def main():
    global _RUN_TIME_SALT_NS
    args = parse_args()
    counters["ok"] = 0
    counters["err"] = 0
    error_buckets.clear()
    type_counts.clear()
    ev = max(1, min(args.evals_per_asset, 8))
    ht = max(0, min(args.hand_tracking_pct, 100))
    cfg = SeedConfig(
        evals_per_asset=ev,
        hand_tracking_pct=ht,
        rich_tags=not args.no_rich_tags,
        bias_metrics_search=not args.no_metrics_bias,
        include_all_ids=args.include_all_ids,
    )
    _RUN_TIME_SALT_NS = time.time_ns() % (10**12)
    print(
        f"Seeding {args.total} assets ({args.workers} workers) → {args.base} "
        f"[evals/asset={cfg.evals_per_asset}, hand_tracking%={cfg.hand_tracking_pct}, rich_tags={cfg.rich_tags}]"
    )
    t0 = time.time()
    asset_ids: list[str] = []
    step = max(1, min(50, args.total // 10 or 1))

    with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as pool:
        futures = {pool.submit(seed_one, i, args.base, args.token, cfg): i for i in range(args.total)}
        for done_count, fut in enumerate(concurrent.futures.as_completed(futures), 1):
            aid = fut.result()
            if aid:
                asset_ids.append(aid)
            if done_count % step == 0 or done_count == args.total:
                elapsed = time.time() - t0
                rate = done_count / elapsed if elapsed > 0 else 0
                print(f"  {done_count}/{args.total}  ok={counters['ok']}  err={counters['err']}  {rate:.1f}/s")

    elapsed = time.time() - t0
    print(
        f"\nDone: ok={counters['ok']}  err={counters['err']}  elapsed={elapsed:.1f}s  "
        f"success={counters['ok'] * 100.0 / max(1, counters['ok'] + counters['err']):.1f}%"
    )
    if type_counts:
        parts = ", ".join(f"{k}={v}" for k, v in sorted(type_counts.items()))
        print(f"By asset_type: {parts}")

    if error_buckets:
        print("\nTop errors:")
        for key, cnt in sorted(error_buckets.items(), key=lambda kv: kv[1], reverse=True)[:12]:
            print(f"  {cnt:>5}  {key}")

    sample_n = min(30, len(asset_ids))
    sample = random.sample(asset_ids, sample_n) if sample_n else []
    out_obj: dict = {"sample_asset_ids": sample, "created": len(asset_ids)}
    if cfg.include_all_ids:
        out_obj["all_asset_ids"] = asset_ids
    with open(args.sample_output, "w") as f:
        json.dump(out_obj, f, indent=2)
    print(f"Wrote sample ({sample_n} ids){' + all_asset_ids' if cfg.include_all_ids else ''} → {args.sample_output}")


if __name__ == "__main__":
    main()

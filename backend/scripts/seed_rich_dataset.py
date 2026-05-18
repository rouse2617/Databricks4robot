#!/usr/bin/env python3
"""
Seed rich mock data via HTTP (local Compose / staging).

Per asset (business-like weighted distribution by default):
  - POST /api/v1/mcap-files (one MCAP can host multiple assets via --assets-per-mcap)
  - POST /api/v1/assets + weighted tags (+ optional task/notes tags)
  - POST algo env_analysis finish; optional hand/head/body/deface/action by configurable probabilities
  - POST /api/v1/deliveries in long-tail distribution (can disable with --no-deliveries)
  - POST multiple /eval-results rows (--evals-per-asset, distinct eval_name/version draws)

Defaults ~100 assets. Use --total / --workers and profile knobs for heavier datasets.
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
from typing import Sequence, TypeVar, Optional, Set

import httpx

BASE = "http://127.0.0.1:8080"
TOKEN = "dev-token"
# Safe defaults for CI / laptop (avoid accidental 10k run)
DEFAULT_TOTAL = 100
DEFAULT_WORKERS = 8
MAX_RETRIES = 5
RETRY_BACKOFF_SEC = 0.2

# Structural `assets.asset_type` values (docs/review/sql.md §4.2).
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
TASK_VALS = ["pick_place", "navigation", "assembly", "qc_scan", "dock_charge"]
CUSTOMERS = [
    "urn:grace:customer:A",
    "urn:grace:customer:B",
    "urn:grace:customer:C",
    "urn:grace:customer:D",
    "urn:grace:customer:E",
]
MCAP_VENDOR_IDS = [
    "seed_rich",
    "grace_vendor_alpha",
    "grace_vendor_beta",
    "grace_vendor_gamma",
    "grace_vendor_delta",
]

EVAL_NAMES = ["hand_tracking_qc", "body_qc", "env_qc", "scene_classifier"]
EVAL_VERSIONS = ["v1.0", "v1.1", "v2.0", "v2.1"]

# Precomputed (eval_name, eval_version) pairs for multi-eval rows without tiny Cartesian explosion at runtime.
_EVAL_COMBOS: list[tuple[str, str]] = list(itertools.product(EVAL_NAMES, EVAL_VERSIONS))

ALGO_ENV = "env_analysis@1.0.0"
ALGO_HAND = "hand_tracking@1.2.0"
ALGO_HEAD = "head_tracking@1.0.0"
ALGO_BODY = "body_tracking@1.0.0"
ALGO_DEFACE = "deface@2.0.0"
ALGO_ACTION = "action_annotation@1.0.0"

# Business-like weighted profile.
ASSET_TYPE_WEIGHTS = [("segment", 60), ("clip", 25), ("frame_set", 12), ("derived_asset", 3)]
LIFECYCLE_WEIGHTS = [
    ("created", 5),
    ("processing", 15),
    ("ready", 35),
    ("delivered", 30),
    ("archived", 10),
    ("superseded", 3),
    ("failed", 1),
    ("rejected", 1),
]
OWNER_WEIGHTS = [
    ("team_alpha", 30),
    ("team_beta", 16),
    ("team_gamma", 16),
    ("team_delta", 14),
    ("team_epsilon", 12),
    ("team_zeta", 12),
]
REVIEWER_WEIGHTS = [("alice", 25), ("bob", 20), ("carol", 15), ("dave", 12), ("eve", 10), ("frank", 10), ("grace", 8)]
QUALITY_WEIGHTS = [("excellent", 20), ("good", 35), ("acceptable", 25), ("poor", 12), ("unusable", 8)]
PRIORITY_WEIGHTS = [("critical", 5), ("high", 25), ("medium", 50), ("low", 20)]
SCENE_TAG_WEIGHTS = [("indoor", 18), ("outdoor", 32), ("warehouse", 22), ("office", 16), ("factory", 12)]
CUSTOMER_WEIGHTS = [
    ("urn:grace:customer:A", 35),
    ("urn:grace:customer:B", 25),
    ("urn:grace:customer:C", 18),
    ("urn:grace:customer:D", 12),
    ("urn:grace:customer:E", 10),
]
MCAP_VENDOR_WEIGHTS = [
    ("seed_rich", 42),
    ("grace_vendor_alpha", 24),
    ("grace_vendor_beta", 18),
    ("grace_vendor_gamma", 10),
    ("grace_vendor_delta", 6),
]
# Long-tail delivery-count distribution by asset.
DELIVERY_COUNT_WEIGHTS = [(0, 35), (1, 30), (2, 17), (3, 8), (4, 5), (5, 3), (8, 2)]
T = TypeVar("T")


@dataclass
class SeedConfig:
    """Per-asset richness: multiple eval rows, optional second algo, extended tags."""

    evals_per_asset: int = 2
    hand_tracking_pct: int = 40
    head_tracking_pct: int = 25
    body_tracking_pct: int = 20
    deface_pct: int = 30
    action_annotation_pct: int = 12
    delivery_enable: bool = True
    delivery_customer_skew: bool = True
    rich_tags: bool = True
    bias_metrics_search: bool = True
    include_all_ids: bool = False
    profile: str = "business"
    quality_tag_pct: int = 92
    priority_tag_pct: int = 92
    scene_tag_pct: int = 78
    batch_tag_pct: int = 88
    task_tag_pct: int = 45
    notes_tag_pct: int = 20
    assets_per_mcap: int = 1

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


def weighted_pick(weighted: Sequence[tuple[T, int]]) -> T:
    values = [v for v, _ in weighted]
    weights = [w for _, w in weighted]
    return random.choices(values, weights=weights, k=1)[0]


def should_pick(pct: int) -> bool:
    return random.randint(1, 100) <= max(0, min(100, pct))


def pick_asset_type(cfg: SeedConfig, idx: int) -> str:
    if cfg.profile == "uniform":
        return STRUCTURAL_ASSET_TYPES[idx % len(STRUCTURAL_ASSET_TYPES)]
    return weighted_pick(ASSET_TYPE_WEIGHTS)


def pick_lifecycle(cfg: SeedConfig) -> str:
    if cfg.profile == "uniform":
        return random.choice(LIFECYCLE)
    return weighted_pick(LIFECYCLE_WEIGHTS)


def pick_owner(cfg: SeedConfig) -> str:
    if cfg.profile == "uniform":
        return random.choice(OWNERS)
    return weighted_pick(OWNER_WEIGHTS)


def pick_reviewer(cfg: SeedConfig) -> str:
    if cfg.profile == "uniform":
        return random.choice(REVIEWERS)
    return weighted_pick(REVIEWER_WEIGHTS)


def pick_quality(cfg: SeedConfig) -> str:
    if cfg.profile == "uniform":
        return random.choice(QUALITY_VALS)
    return weighted_pick(QUALITY_WEIGHTS)


def pick_priority(cfg: SeedConfig) -> str:
    if cfg.profile == "uniform":
        return random.choice(PRIORITY_VALS)
    return weighted_pick(PRIORITY_WEIGHTS)


def pick_scene_tag(cfg: SeedConfig) -> str:
    if cfg.profile == "uniform":
        return random.choice(SCENE_TAG_VALS)
    return weighted_pick(SCENE_TAG_WEIGHTS)


def pick_vendor_id(cfg: SeedConfig) -> str:
    if cfg.profile == "uniform":
        return random.choice(MCAP_VENDOR_IDS)
    return weighted_pick(MCAP_VENDOR_WEIGHTS)


def pick_customer_id(cfg: SeedConfig) -> str:
    if cfg.profile == "uniform" or not cfg.delivery_customer_skew:
        return random.choice(CUSTOMERS)
    return weighted_pick(CUSTOMER_WEIGHTS)


def pick_delivery_count(cfg: SeedConfig) -> int:
    if cfg.profile == "uniform":
        return random.randint(0, 4)
    return weighted_pick(DELIVERY_COUNT_WEIGHTS)


def make_mcap_payload(mcap_id: str, t_start: int, t_end: int, seq: int, owner: str, cfg: SeedConfig) -> dict:
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
        "vendor_id": pick_vendor_id(cfg),
        "device_id": f"dev-{rand_str(4)}",
        "scene_id": random.choice(SCENES),
        "owner": owner,
    }


def make_asset_payload(mcap_id: str, t_start: int, t_end: int, structural_type: str, cfg: SeedConfig, owner: str) -> dict:
    """structural_type is assets.asset_type (segment | clip | frame_set | derived_asset)."""
    scene = random.choice(SCENES)
    env = random.choice(ENVS)
    return {
        "mcap_file_id": mcap_id,
        "start_timestamp_ns": t_start,
        "end_timestamp_ns": t_end,
        "owner": owner,
        "reviewer": pick_reviewer(cfg),
        "lifecycle_state": pick_lifecycle(cfg),
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


def make_tags(cfg: SeedConfig) -> list[dict]:
    """Tag keys aligned with backend/config/tag_registry.yaml enums where applicable."""
    tags: list[dict] = []

    if should_pick(cfg.quality_tag_pct):
        tags.append({"key": "quality", "value": pick_quality(cfg)})
    if should_pick(cfg.priority_tag_pct):
        tags.append({"key": "priority", "value": pick_priority(cfg)})
    if should_pick(cfg.scene_tag_pct):
        tags.append({"key": "scene", "value": pick_scene_tag(cfg)})
    if should_pick(cfg.batch_tag_pct):
        tags.append({"key": "batch", "value": f"B-{random.randint(1, 9999):04d}"})

    if cfg.rich_tags:
        if should_pick(cfg.task_tag_pct):
            tags.append({"key": "task", "value": random.choice(TASK_VALS)})
        if should_pick(cfg.notes_tag_pct):
            tags.append({"key": "notes", "value": f"seed_rich idx notes len={random.randint(12, 180)}"[:500]})

    # Keep at least one tag so tag views are always populated.
    if not tags:
        tags.append({"key": "batch", "value": f"B-{random.randint(1, 9999):04d}"})
    return tags


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
lifecycle_counts: dict[str, int] = {}
tag_key_counts: dict[str, int] = {}
algo_counts: dict[str, int] = {}
delivery_hist: dict[int, int] = {}


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
    if base == "deface":
        width = random.choice([1280, 1920, 2560])
        height = random.choice([720, 1080, 1440])
        fps = random.choice([24, 25, 30, 60])
        source_stream = random.choice(["front_wide", "front_tele", "rear_wide"])
        eye = random.choice(["left", "right", "both"])
        body["output_uri"] = f"gs://seed-rich/algo/{base}/{rand_str(14)}.mp4"
        body["result_size_bytes"] = random.randint(10_000_000, 600_000_000)
        # Keep required fields top-level for strict validators.
        body["width"] = width
        body["height"] = height
        body["fps"] = fps
        body["source_stream"] = source_stream
        body["eye"] = eye
        body["extra_fields"] = {
            "type": "video",
            "width": width,
            "height": height,
            "fps": fps,
            "source_stream": source_stream,
            "eye": eye,
        }
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


def commit_delivery(c: httpx.Client, asset_id: str, owner: str, customer_id: str) -> bool:
    payload = {
        "asset_ids": [asset_id],
        "customer_id": customer_id,
        "contract_id": f"ctr-{owner}-{rand_str(6)}",
        "note": f"seed_rich delivery for {owner}",
        "owner": owner,
    }
    idem_key = f"seed-delivery-{asset_id}-{uuid.uuid4()}"
    for i in range(MAX_RETRIES):
        try:
            resp = c.post(
                "/api/v1/deliveries",
                json=payload,
                headers={"Idempotency-Key": idem_key},
            )
            if resp.status_code == 201:
                return True
            if resp.status_code in (429, 500, 502, 503, 504):
                time.sleep(RETRY_BACKOFF_SEC * (2**i))
                continue
            bump_error(f"/api/v1/deliveries:status={resp.status_code}:{resp.text[:200]}")
            return False
        except Exception as ex:  # noqa: BLE001
            time.sleep(RETRY_BACKOFF_SEC * (2**i))
            if i == MAX_RETRIES - 1:
                bump_error(f"/api/v1/deliveries:ex={type(ex).__name__}")
    return False


def record_seed_success(
    structural_type: str,
    lifecycle_state: str,
    tags: list[dict],
    ran_algos: set[str],
    committed_deliveries: int,
) -> None:
    with _counters_lock:
        counters["ok"] += 1
        type_counts[structural_type] = type_counts.get(structural_type, 0) + 1
        lifecycle_counts[lifecycle_state] = lifecycle_counts.get(lifecycle_state, 0) + 1
        for tag in tags:
            key = str(tag.get("key", ""))
            tag_key_counts[key] = tag_key_counts.get(key, 0) + 1
        for algo_key in ran_algos:
            algo_counts[algo_key] = algo_counts.get(algo_key, 0) + 1
        delivery_hist[committed_deliveries] = delivery_hist.get(committed_deliveries, 0) + 1


def record_seed_error(count: int = 1) -> None:
    with _counters_lock:
        counters["err"] += count


def sample_recent_window_ns() -> tuple[int, int]:
    """Sample a recent time window used for MCAP and its child assets."""
    lookback_days = int(random.betavariate(2.2, 5.0) * 120)
    day_jitter_ns = random.randint(0, int(ONE_DAY_NS) - 1)
    start = max(T0_NS, time.time_ns() - lookback_days * ONE_DAY_NS - day_jitter_ns + (_RUN_TIME_SALT_NS % ONE_DAY_NS))
    dur_s = random.randint(10, 300)
    end = start + dur_s * 1_000_000_000
    return start, end


def sample_asset_range_from_window(window: tuple[int, int]) -> tuple[int, int]:
    """Sample an asset time range that is always inside a shared MCAP window."""
    w_start, w_end = window
    span_ns = max(int(60 * 1_000_000_000), w_end - w_start)
    dur_ns = random.randint(int(10 * 1_000_000_000), int(300 * 1_000_000_000))
    if dur_ns >= span_ns:
        return w_start, min(w_end, w_start + dur_ns)
    max_offset = span_ns - dur_ns
    offset = random.randint(0, max_offset)
    a_start = w_start + offset
    a_end = a_start + dur_ns
    return a_start, a_end


def seed_one(
    idx: int,
    base_url: str,
    token: str,
    cfg: SeedConfig,
    *,
    shared_mcap_id: str | None = None,
    shared_owner: str | None = None,
    shared_window: tuple[int, int] | None = None,
) -> str:
    try:
        if shared_mcap_id and shared_owner and shared_window:
            t_start, t_end = sample_asset_range_from_window(shared_window)
            owner = shared_owner
            mcap_id = shared_mcap_id
            c = get_client(base_url, token)
        else:
            t_start, t_end = sample_recent_window_ns()
            owner = pick_owner(cfg)
            mcap_id = generate_mcap_file_id()
            c = get_client(base_url, token)
            mcap_p = make_mcap_payload(mcap_id, t_start, t_end, idx, owner, cfg)
            if request_json_retry(c, "POST", "/api/v1/mcap-files", {201}, mcap_p) is None:
                record_seed_error()
                return ""

        structural_type = pick_asset_type(cfg, idx)
        asset_p = make_asset_payload(mcap_id, t_start, t_end, structural_type, cfg, owner)
        asset_data = request_json_retry(c, "POST", "/api/v1/assets", {201}, asset_p)
        if asset_data is None:
            record_seed_error()
            return ""
        asset_id = asset_data["asset_id"]

        tags = make_tags(cfg)
        for tag in tags:
            if request_json_retry(c, "POST", f"/api/v1/assets/{asset_id}/tags", {200}, tag) is None:
                record_seed_error()
                return ""

        if not run_algo_pipeline(c, asset_id, ALGO_ENV):
            record_seed_error()
            return ""
        ran_algos: set[str] = {ALGO_ENV}

        if cfg.hand_tracking_pct > 0 and should_pick(cfg.hand_tracking_pct):
            if not run_algo_pipeline(c, asset_id, ALGO_HAND):
                record_seed_error()
                return ""
            ran_algos.add(ALGO_HAND)

        if cfg.head_tracking_pct > 0 and should_pick(cfg.head_tracking_pct):
            if not run_algo_pipeline(c, asset_id, ALGO_HEAD):
                record_seed_error()
                return ""
            ran_algos.add(ALGO_HEAD)

        if cfg.body_tracking_pct > 0 and should_pick(cfg.body_tracking_pct):
            if not run_algo_pipeline(c, asset_id, ALGO_BODY):
                record_seed_error()
                return ""
            ran_algos.add(ALGO_BODY)

        if cfg.deface_pct > 0 and should_pick(cfg.deface_pct):
            if not run_algo_pipeline(c, asset_id, ALGO_DEFACE):
                record_seed_error()
                return ""
            ran_algos.add(ALGO_DEFACE)

        if (
            cfg.action_annotation_pct > 0
            and should_pick(cfg.action_annotation_pct)
            and ALGO_HAND in ran_algos
            and ALGO_HEAD in ran_algos
            and ALGO_BODY in ran_algos
        ):
            if not run_algo_pipeline(c, asset_id, ALGO_ACTION):
                record_seed_error()
                return ""
            ran_algos.add(ALGO_ACTION)

        combos = random.sample(_EVAL_COMBOS, k=min(cfg.evals_per_asset, len(_EVAL_COMBOS)))
        if len(combos) < cfg.evals_per_asset:
            combos.extend(random.choices(_EVAL_COMBOS, k=cfg.evals_per_asset - len(combos)))
        for ei, (ename, ever) in enumerate(combos[: cfg.evals_per_asset]):
            biased = cfg.bias_metrics_search and ei == 0
            eval_p = make_eval_payload(ename, ever, biased_high_quality=biased)
            if request_json_retry(c, "POST", f"/api/v1/assets/{asset_id}/eval-results", {201}, eval_p) is None:
                record_seed_error()
                return ""

        committed_deliveries = 0
        if cfg.delivery_enable:
            delivery_n = pick_delivery_count(cfg)
            for _ in range(delivery_n):
                customer_id = pick_customer_id(cfg)
                if not commit_delivery(c, asset_id, owner, customer_id):
                    record_seed_error()
                    return ""
                committed_deliveries += 1

        record_seed_success(
            structural_type=structural_type,
            lifecycle_state=asset_p["lifecycle_state"],
            tags=tags,
            ran_algos=ran_algos,
            committed_deliveries=committed_deliveries,
        )
        return asset_id

    except Exception as ex:  # noqa: BLE001
        bump_error(f"seed_one:{type(ex).__name__}")
        record_seed_error()
        return ""


def seed_bundle(bundle_idx: int, start_idx: int, count: int, base_url: str, token: str, cfg: SeedConfig) -> list[str]:
    """Create one MCAP then attach multiple assets to it."""
    if count <= 0:
        return []
    owner = pick_owner(cfg)
    window_start, _ = sample_recent_window_ns()
    window_span_ns = random.randint(int(20 * 60 * 1_000_000_000), int(3 * 60 * 60 * 1_000_000_000))
    window_end = window_start + window_span_ns
    mcap_id = generate_mcap_file_id()
    c = get_client(base_url, token)
    mcap_p = make_mcap_payload(mcap_id, window_start, window_end, bundle_idx, owner, cfg)
    if request_json_retry(c, "POST", "/api/v1/mcap-files", {201}, mcap_p) is None:
        # One MCAP failure invalidates the whole bundle.
        record_seed_error(count)
        return []
    out: list[str] = []
    shared_window = (window_start, window_end)
    for i in range(count):
        aid = seed_one(
            start_idx + i,
            base_url,
            token,
            cfg,
            shared_mcap_id=mcap_id,
            shared_owner=owner,
            shared_window=shared_window,
        )
        if aid:
            out.append(aid)
    return out


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
        "--assets-per-mcap",
        type=int,
        default=4,
        metavar="N",
        help="Attach N assets to one MCAP file (default 4)",
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
    p.add_argument(
        "--profile",
        choices=["business", "uniform"],
        default="business",
        help="Data distribution profile: business (weighted realistic) or uniform (legacy)",
    )
    p.add_argument(
        "--head-tracking-pct",
        type=int,
        default=25,
        metavar="PCT",
        help="Probability 1–100 to also run head_tracking@1.0.0 (default 25)",
    )
    p.add_argument(
        "--body-tracking-pct",
        type=int,
        default=20,
        metavar="PCT",
        help="Probability 1–100 to also run body_tracking@1.0.0 (default 20)",
    )
    p.add_argument(
        "--deface-pct",
        type=int,
        default=30,
        metavar="PCT",
        help="Probability 1–100 to also run deface@2.0.0 (default 30)",
    )
    p.add_argument(
        "--action-annotation-pct",
        type=int,
        default=12,
        metavar="PCT",
        help="Probability 1–100 to run action_annotation@1.0.0 after hand/head/body pass (default 12)",
    )
    p.add_argument("--no-deliveries", action="store_true", help="Disable delivery creation stage")
    p.add_argument(
        "--quality-tag-pct",
        type=int,
        default=92,
        metavar="PCT",
        help="Probability 1–100 to attach quality tag (default 92)",
    )
    p.add_argument(
        "--priority-tag-pct",
        type=int,
        default=92,
        metavar="PCT",
        help="Probability 1–100 to attach priority tag (default 92)",
    )
    p.add_argument(
        "--scene-tag-pct",
        type=int,
        default=78,
        metavar="PCT",
        help="Probability 1–100 to attach scene tag (default 78)",
    )
    p.add_argument(
        "--batch-tag-pct",
        type=int,
        default=88,
        metavar="PCT",
        help="Probability 1–100 to attach batch tag (default 88)",
    )
    p.add_argument(
        "--task-tag-pct",
        type=int,
        default=45,
        metavar="PCT",
        help="Probability 1–100 to attach task tag when rich_tags enabled (default 45)",
    )
    p.add_argument(
        "--notes-tag-pct",
        type=int,
        default=20,
        metavar="PCT",
        help="Probability 1–100 to attach notes tag when rich_tags enabled (default 20)",
    )
    p.add_argument("--seed", type=int, default=0, help="Optional random seed for reproducible runs (0=disabled)")
    return p.parse_args()


def main():
    global _RUN_TIME_SALT_NS
    args = parse_args()
    if args.seed > 0:
        random.seed(args.seed)
    counters["ok"] = 0
    counters["err"] = 0
    error_buckets.clear()
    type_counts.clear()
    lifecycle_counts.clear()
    tag_key_counts.clear()
    algo_counts.clear()
    delivery_hist.clear()
    ev = max(1, min(args.evals_per_asset, 8))
    ht = max(0, min(args.hand_tracking_pct, 100))
    hd = max(0, min(args.head_tracking_pct, 100))
    bd = max(0, min(args.body_tracking_pct, 100))
    df = max(0, min(args.deface_pct, 100))
    ac = max(0, min(args.action_annotation_pct, 100))
    cfg = SeedConfig(
        evals_per_asset=ev,
        hand_tracking_pct=ht,
        head_tracking_pct=hd,
        body_tracking_pct=bd,
        deface_pct=df,
        action_annotation_pct=ac,
        delivery_enable=not args.no_deliveries,
        rich_tags=not args.no_rich_tags,
        bias_metrics_search=not args.no_metrics_bias,
        include_all_ids=args.include_all_ids,
        profile=args.profile,
        quality_tag_pct=max(0, min(args.quality_tag_pct, 100)),
        priority_tag_pct=max(0, min(args.priority_tag_pct, 100)),
        scene_tag_pct=max(0, min(args.scene_tag_pct, 100)),
        batch_tag_pct=max(0, min(args.batch_tag_pct, 100)),
        task_tag_pct=max(0, min(args.task_tag_pct, 100)),
        notes_tag_pct=max(0, min(args.notes_tag_pct, 100)),
        assets_per_mcap=max(1, args.assets_per_mcap),
    )
    _RUN_TIME_SALT_NS = time.time_ns() % (10**12)
    print(
        f"Seeding {args.total} assets ({args.workers} workers) → {args.base} "
        f"[profile={cfg.profile}, assets/mcap={cfg.assets_per_mcap}, evals/asset={cfg.evals_per_asset}, "
        f"algos(hand/head/body/deface/action)={cfg.hand_tracking_pct}/{cfg.head_tracking_pct}/"
        f"{cfg.body_tracking_pct}/{cfg.deface_pct}/{cfg.action_annotation_pct}, "
        f"deliveries={cfg.delivery_enable}, rich_tags={cfg.rich_tags}]"
    )
    t0 = time.time()
    asset_ids: list[str] = []
    step = max(1, min(50, args.total // 10 or 1))

    bundles: list[tuple[int, int, int]] = []
    start_idx = 0
    bundle_idx = 0
    while start_idx < args.total:
        size = min(cfg.assets_per_mcap, args.total - start_idx)
        bundles.append((bundle_idx, start_idx, size))
        start_idx += size
        bundle_idx += 1

    done_assets = 0
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as pool:
        futures = {
            pool.submit(seed_bundle, b_idx, s_idx, b_size, args.base, args.token, cfg): b_size
            for (b_idx, s_idx, b_size) in bundles
        }
        for fut in concurrent.futures.as_completed(futures):
            done_assets += futures[fut]
            ids = fut.result()
            if ids:
                asset_ids.extend(ids)
            if done_assets % step == 0 or done_assets >= args.total:
                elapsed = time.time() - t0
                rate = done_assets / elapsed if elapsed > 0 else 0
                print(f"  {done_assets}/{args.total}  ok={counters['ok']}  err={counters['err']}  {rate:.1f}/s")

    elapsed = time.time() - t0
    print(
        f"\nDone: ok={counters['ok']}  err={counters['err']}  elapsed={elapsed:.1f}s  "
        f"success={counters['ok'] * 100.0 / max(1, counters['ok'] + counters['err']):.1f}%"
    )
    if type_counts:
        parts = ", ".join(f"{k}={v}" for k, v in sorted(type_counts.items()))
        print(f"By asset_type: {parts}")
    if lifecycle_counts:
        parts = ", ".join(f"{k}={v}" for k, v in sorted(lifecycle_counts.items()))
        print(f"By lifecycle_state: {parts}")
    if tag_key_counts:
        parts = ", ".join(f"{k}={v}" for k, v in sorted(tag_key_counts.items()))
        print(f"Tag key coverage: {parts}")
    if algo_counts:
        parts = ", ".join(f"{k}={v}" for k, v in sorted(algo_counts.items()))
        print(f"Algo coverage: {parts}")
    if delivery_hist:
        parts = ", ".join(f"{k}x={v}" for k, v in sorted(delivery_hist.items()))
        print(f"Deliveries per asset histogram: {parts}")

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

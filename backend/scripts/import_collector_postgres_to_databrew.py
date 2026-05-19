#!/usr/bin/env python3
"""
Import MCAP file rows + segment assets from collector-db `postgres` into Cyber Databrew via HTTP API.
Use --assets-only to skip POST /mcap-files when MCAP rows already exist and only write segment assets.

Sources (pick one):
  - JSONL files from read-only SELECT (see collector_postgres_export.md), or
  - `--pg-dsn` / `COLLECTOR_PG_DSN`: stream SELECTs inside the cluster (recommended: run this
    script from a GKE Job so collector private IP + HTTPS to Cloud Run both work; no kubectl
    redirect noise).

Collector is never written to; only SELECT when using --pg-dsn.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import ssl
import sys
import threading
import time
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterator

MD32 = re.compile(r"^[0-9a-f]{32}$")


def _fmt_s(seconds: float) -> str:
    if seconds >= 120:
        return f"{seconds / 60:.2f}m"
    return f"{seconds:.2f}s"

SQL_MCAP_VIDEOS = """
SELECT to_jsonb(v)
FROM grace_videos v
WHERE v.raw_hash_md5 IS NOT NULL
  AND COALESCE(v.storage_meta::text, '') ILIKE '%gs://%.mcap%'
  AND NOT COALESCE(v.is_deleted, false)
ORDER BY v.id
"""

SQL_MCAP_SEGMENTS = """
SELECT to_jsonb(s)
FROM annotation_segmentations s
JOIN grace_videos v ON v.id = s.video_id
WHERE v.raw_hash_md5 IS NOT NULL
  AND COALESCE(v.storage_meta::text, '') ILIKE '%gs://%.mcap%'
  AND NOT COALESCE(v.is_deleted, false)
  AND NOT COALESCE(s.is_deleted, false)
  AND s.start_timestamp > 0
  AND s.end_timestamp > s.start_timestamp
ORDER BY s.video_id, s.start_timestamp, s.end_timestamp,
  CASE WHEN COALESCE(NULLIF(trim(s.env), ''), '-') IN ('-', '') THEN 1 ELSE 0 END,
  s.segmentation_id
"""


def _decode_pg_json(blob: Any) -> dict[str, Any]:
    if isinstance(blob, dict):
        return blob
    if isinstance(blob, str):
        return json.loads(blob)
    if isinstance(blob, (bytes, memoryview)):
        return json.loads(bytes(blob))
    raise TypeError(f"unexpected json column type {type(blob)}")


def _pg_connect_readonly(dsn: str) -> Any:
    try:
        import psycopg2  # type: ignore[import-untyped]
    except ImportError as e:
        raise RuntimeError(
            "psycopg2 is required for --pg-dsn. Install: pip install -r backend/scripts/requirements-collector-import.txt"
        ) from e
    conn = psycopg2.connect(dsn)
    conn.autocommit = True
    with conn.cursor() as cur:
        cur.execute("SET default_transaction_read_only = on")
    return conn


def _iter_pg_batches(dsn: str, sql: str, fetch_size: int = 300) -> Iterator[dict[str, Any]]:
    """Stream rows using fetchmany (works with autocommit; read-only session)."""
    conn = _pg_connect_readonly(dsn)
    try:
        with conn.cursor() as cur:
            cur.execute(sql)
            while True:
                rows = cur.fetchmany(fetch_size)
                if not rows:
                    break
                for row in rows:
                    yield _decode_pg_json(row[0])
    finally:
        conn.close()


def iter_mcap_videos_pg(dsn: str) -> Iterator[dict[str, Any]]:
    yield from _iter_pg_batches(dsn, SQL_MCAP_VIDEOS, 200)


def iter_mcap_segments_pg(dsn: str) -> Iterator[dict[str, Any]]:
    yield from _iter_pg_batches(dsn, SQL_MCAP_SEGMENTS, 400)


def _iso_to_ns(s: Any) -> int:
    if s is None or s == "":
        return 0
    if not isinstance(s, str):
        return 0
    t = s.strip()
    if not t:
        return 0
    if t.endswith("Z"):
        t = t[:-1] + "+00:00"
    try:
        dt = datetime.fromisoformat(t)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return int(dt.timestamp() * 1e9)
    except ValueError:
        return 0


def _stream_jsonl(path: Path) -> Iterator[dict[str, Any]]:
    with path.open(encoding="utf-8", errors="replace") as f:
        for line_no, line in enumerate(f, 1):
            line = line.strip()
            if not line:
                continue
            if not line.startswith("{"):
                # kubectl / shell noise sometimes lands in redirected output
                print(f"{path}:{line_no}: skip non-JSON line: {line[:80]!r}", file=sys.stderr)
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError as e:
                raise ValueError(f"{path}:{line_no}: invalid JSON: {e}") from e
            if not isinstance(obj, dict):
                raise ValueError(f"{path}:{line_no}: expected object, got {type(obj)}")
            yield obj


def _count_streams(video_info: dict[str, Any]) -> int:
    if not isinstance(video_info, dict):
        return 0
    n = 0
    for k, v in video_info.items():
        if isinstance(k, str) and k.startswith("/") and isinstance(v, dict):
            n += 1
    return n


def _process_state(video: dict[str, Any]) -> dict[str, str] | None:
    pi = video.get("process_info")
    if not isinstance(pi, dict) or not pi:
        return None
    out: dict[str, str] = {}
    for k, v in pi.items():
        if v is None:
            continue
        out[str(k)] = v if isinstance(v, str) else str(v)
    return out or None


def _scene_id(collection_meta: dict[str, Any]) -> str:
    s = collection_meta.get("scene")
    if s is None or s == "" or s == "null":
        return ""
    return str(s)


def _first_vendor_ref(collection_meta: dict[str, Any]) -> str:
    refs = collection_meta.get("external_refs")
    if not isinstance(refs, list):
        return ""
    for r in refs:
        if isinstance(r, dict):
            rid = r.get("id")
            if rid:
                return str(rid)
    return ""


def build_mcap_body(
    video: dict[str, Any],
    *,
    import_batch: str,
    source_db: str,
    owner: str,
) -> dict[str, Any]:
    md5 = (video.get("raw_hash_md5") or "").strip().lower()
    if not MD32.match(md5):
        raise ValueError("raw_hash_md5 must be 32 lowercase hex chars")

    sm = video.get("storage_meta")
    if not isinstance(sm, dict):
        sm = {}
    gcs = sm.get("gcs") if isinstance(sm.get("gcs"), dict) else {}
    gcs_path = ""
    if isinstance(gcs.get("video"), str):
        gcs_path = gcs["video"].strip()
    if not gcs_path.lower().endswith(".mcap"):
        raise ValueError("storage_meta.gcs.video must be an .mcap gs:// URI")

    mcap_file_id = md5[:8].upper()
    vi = video.get("video_info")
    if not isinstance(vi, dict):
        vi = {}
    size_bytes = 0
    vs = vi.get("video_size")
    if isinstance(vs, int):
        size_bytes = vs
    elif isinstance(vs, float):
        size_bytes = int(vs)

    file_duration_ms = 0
    ds = video.get("duration_sec")
    if ds is not None:
        try:
            file_duration_ms = int(float(ds) * 1000)
        except (TypeError, ValueError):
            pass

    cm = video.get("collection_meta")
    if not isinstance(cm, dict):
        cm = {}

    start_ns = _iso_to_ns(cm.get("started_at"))
    end_ns = _iso_to_ns(cm.get("ended_at"))

    channel_count = _count_streams(vi)

    camera_model = video.get("camera_model")
    camera_model_s = str(camera_model).strip() if camera_model is not None else ""

    device_id = cm.get("device_id")
    device_id_s = str(device_id).strip() if device_id is not None else ""

    data_source = cm.get("data_source")
    data_source_s = str(data_source).strip() if data_source is not None else ""

    collection_method = cm.get("collection_method")
    collection_method_s = str(collection_method).strip() if collection_method is not None else ""

    scene_id = _scene_id(cm)
    vendor_id = _first_vendor_ref(cm)

    collector_id = ""
    session = cm.get("collection_session_id")
    if session:
        collector_id = str(session)

    task_id = ""
    part = cm.get("collection_part_index")
    if part is not None:
        task_id = f"collection_part_index:{part}"

    sha256 = video.get("raw_hash_sha256")
    sha256_s = str(sha256).strip() if sha256 else ""

    meta: dict[str, Any] = {
        "import_batch": import_batch,
        "source_db": source_db,
        "source_table": "grace_videos",
        "grace_video_snapshot": video,
    }
    if video.get("d_type") is not None:
        meta["grace_d_type"] = video.get("d_type")

    body: dict[str, Any] = {
        "mcap_file_id": mcap_file_id,
        "gcs_path": gcs_path,
        "raw_hash_md5": md5,
        "ingest_state": "summarized",
        "size_bytes": size_bytes,
        "owner": owner,
        "metadata": meta,
    }

    if sha256_s:
        body["raw_hash_sha256"] = sha256_s
    if file_duration_ms > 0:
        body["file_duration_ms"] = file_duration_ms
    if start_ns > 0:
        body["start_timestamp_ns"] = start_ns
    if end_ns > 0:
        body["end_timestamp_ns"] = end_ns
    if channel_count > 0:
        body["channel_count"] = channel_count
    if camera_model_s:
        body["camera_model"] = camera_model_s
    if device_id_s:
        body["device_id"] = device_id_s
    if data_source_s:
        body["data_source"] = data_source_s
    if collection_method_s:
        body["collection_method"] = collection_method_s
    if scene_id:
        body["scene_id"] = scene_id
    if vendor_id:
        body["vendor_id"] = vendor_id
    if collector_id:
        body["collector_id"] = collector_id
    if task_id:
        body["task_id"] = task_id

    ps = _process_state(video)
    if ps:
        body["process_state"] = ps

    return body


def build_mcap_reference_maps(
    videos: Iterator[dict[str, Any]],
    *,
    import_batch: str,
    source_db: str,
    owner: str,
    max_videos: int,
) -> tuple[dict[str, str], dict[str, str], int]:
    """Build video_id -> mcap_file_id and gcs_path from collector rows only (no HTTP).

    Same mcap_file_id and gcs_path as build_mcap_body(), matching Databrew after a prior
    successful POST /mcap-files so segment assets can be imported alone.
    """
    mcap_by_video: dict[str, str] = {}
    gcs_by_video: dict[str, str] = {}
    skipped = 0
    i = 0
    for video in videos:
        i += 1
        if max_videos and i > max_videos:
            break
        vid = video.get("id")
        if vid is None:
            skipped += 1
            continue
        vid_s = str(vid)
        try:
            body = build_mcap_body(
                video,
                import_batch=import_batch,
                source_db=source_db,
                owner=owner,
            )
        except ValueError:
            skipped += 1
            continue
        mcap_by_video[vid_s] = body["mcap_file_id"]
        gcs_by_video[vid_s] = body["gcs_path"]
    print(
        f"[assets-only] local mcap maps: videos={len(mcap_by_video)} skipped_rows={skipped}",
        flush=True,
    )
    return mcap_by_video, gcs_by_video, skipped


def build_asset_body(
    seg: dict[str, Any],
    *,
    mcap_file_id: str,
    gcs_path: str,
    reviewer: str,
    owner: str,
    import_batch: str,
    source_db: str,
    segment_index: int,
) -> dict[str, Any]:
    vid = seg.get("video_id")
    if vid is None:
        raise ValueError("segment missing video_id")
    video_id_s = str(vid)

    st = int(seg.get("start_timestamp") or 0)
    et = int(seg.get("end_timestamp") or 0)
    if st <= 0 or et <= st:
        raise ValueError("invalid segment time range")

    env = seg.get("env")
    env_s = str(env).strip() if env is not None else ""
    if env_s in ("", "-"):
        env_s = source_db

    task = seg.get("task")
    task_s = str(task).strip() if task is not None else ""
    if task_s in ("", "-"):
        task_s = "collector-postgres-seg-import"

    stype = seg.get("type")
    type_s = str(stype) if stype is not None else "segment"

    cstatus = seg.get("status")
    status_s = ""
    if isinstance(cstatus, str) and cstatus.lower() == "done":
        status_s = "approved"

    meta: dict[str, Any] = {
        "import_batch": import_batch,
        "source_db": source_db,
        "source_table": "annotation_segmentations",
        "source_video_id": video_id_s,
        "gcs_mcap_path": gcs_path,
        "annotation_segmentation_snapshot": seg,
    }

    body: dict[str, Any] = {
        "mcap_file_id": mcap_file_id,
        "start_timestamp_ns": st,
        "end_timestamp_ns": et,
        "reviewer": reviewer,
        "owner": owner,
        "asset_type": "segment",
        "type": type_s,
        "env": env_s,
        "task": task_s,
        "metadata": meta,
        "segment_index": segment_index,
        "files": {"raw_mcap": mcap_file_id},
    }
    if status_s:
        body["status"] = status_s
    if gcs_path:
        body["storage_uri"] = gcs_path

    return body


def _retry_after_seconds(retry_after: str | None, attempt: int) -> float:
    """Honor Retry-After (seconds); fall back to capped exponential backoff."""
    if retry_after:
        s = retry_after.strip()
        if s.isdigit():
            return min(60.0, float(s))
    return min(30.0, 0.5 * (2**attempt))


def http_json(
    method: str,
    url: str,
    payload: dict[str, Any] | None,
    headers: dict[str, str],
    ctx: ssl.SSLContext,
    retries: int = 10,
) -> tuple[int, Any]:
    data = None if payload is None else json.dumps(payload, ensure_ascii=False).encode("utf-8")
    last_err: Exception | None = None
    for attempt in range(retries):
        req = urllib.request.Request(url, data=data, method=method, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=120, context=ctx) as resp:
                raw = resp.read().decode("utf-8") or "{}"
                try:
                    return resp.status, json.loads(raw)
                except json.JSONDecodeError:
                    return resp.status, {"raw": raw}
        except urllib.error.HTTPError as e:
            raw = e.read().decode("utf-8", errors="replace") or "{}"
            try:
                parsed: Any = json.loads(raw)
            except json.JSONDecodeError:
                parsed = {"raw": raw}
            if e.code == 429 and attempt + 1 < retries:
                delay = _retry_after_seconds(e.headers.get("Retry-After"), attempt)
                time.sleep(delay)
                continue
            return e.code, parsed
        except Exception as e:
            last_err = e
            time.sleep(0.5 * (attempt + 1))
    return 0, {"code": "NETWORK_ERROR", "message": str(last_err)}


def _flush_mcap_http_batch(
    batch: list[tuple[str, dict[str, Any]]],
    *,
    url: str,
    headers: dict[str, str],
    ctx: ssl.SSLContext,
    mcap_workers: int,
    dry_run: bool,
    mcap_by_video: dict[str, str],
    gcs_by_video: dict[str, str],
    lock: threading.Lock,
    counters: list[int],  # [created, exists, fail, processed]
    http_acc: dict[str, float | int],
    progress_clock: dict[str, float],
) -> None:
    """Apply one HTTP batch; updates maps only on 201/409."""
    if not batch:
        return
    t_batch0 = time.perf_counter()
    if dry_run:
        with lock:
            for vid_s, body in batch:
                print(json.dumps({"POST": "/api/v1/mcap-files", "body": body}, ensure_ascii=False))
                mcap_by_video[vid_s] = body["mcap_file_id"]
                gcs_by_video[vid_s] = body["gcs_path"]
                counters[0] += 1
                counters[3] += 1
                if counters[3] % 500 == 0:
                    now = time.perf_counter()
                    dt = now - progress_clock["t"]
                    progress_clock["t"] = now
                    r = 500.0 / dt if dt > 0 else 0.0
                    print(
                        f"... mcap progress {counters[3]} created={counters[0]} exists={counters[1]} "
                        f"skip={counters[4]} fail={counters[2]} "
                        f"last500_wall={_fmt_s(dt)} ~{r:.1f}/s",
                        flush=True,
                    )
        http_acc["wall_s"] += time.perf_counter() - t_batch0
        http_acc["n"] += 1
        return

    def handle_result(vid_s: str, body: dict[str, Any], code: int, resp: Any) -> None:
        mid = body.get("mcap_file_id", "")
        with lock:
            if code == 201:
                counters[0] += 1
                mcap_by_video[vid_s] = mid
                gcs_by_video[vid_s] = body["gcs_path"]
            elif code == 409:
                counters[1] += 1
                mcap_by_video[vid_s] = mid
                gcs_by_video[vid_s] = body["gcs_path"]
            else:
                counters[2] += 1
                print(
                    f"[mcap FAIL] video={vid_s} mcap_file_id={mid} code={code} resp={resp}",
                    file=sys.stderr,
                )
            counters[3] += 1
            if counters[3] % 500 == 0:
                now = time.perf_counter()
                dt = now - progress_clock["t"]
                progress_clock["t"] = now
                r = 500.0 / dt if dt > 0 else 0.0
                print(
                    f"... mcap progress {counters[3]} created={counters[0]} exists={counters[1]} "
                    f"skip={counters[4]} fail={counters[2]} "
                    f"last500_wall={_fmt_s(dt)} ~{r:.1f}/s",
                    flush=True,
                )

    if mcap_workers <= 1:
        for vid_s, body in batch:
            code, resp = http_json("POST", url, body, headers, ctx)
            handle_result(vid_s, body, code, resp)
    else:
        with ThreadPoolExecutor(max_workers=mcap_workers) as ex:
            future_map: dict[Any, tuple[str, dict[str, Any]]] = {}
            for vid_s, body in batch:
                fut = ex.submit(http_json, "POST", url, body, headers, ctx)
                future_map[fut] = (vid_s, body)
            for fut in as_completed(future_map):
                vid_s, body = future_map[fut]
                try:
                    code, resp = fut.result()
                except Exception as e:
                    with lock:
                        counters[2] += 1
                        counters[3] += 1
                        print(
                            f"[mcap FAIL] video={vid_s} mcap_file_id={body.get('mcap_file_id')} err={e}",
                            file=sys.stderr,
                        )
                        if counters[3] % 500 == 0:
                            now = time.perf_counter()
                            dt = now - progress_clock["t"]
                            progress_clock["t"] = now
                            r = 500.0 / dt if dt > 0 else 0.0
                            print(
                                f"... mcap progress {counters[3]} created={counters[0]} exists={counters[1]} "
                                f"skip={counters[4]} fail={counters[2]} "
                                f"last500_wall={_fmt_s(dt)} ~{r:.1f}/s",
                                flush=True,
                            )
                    continue
                handle_result(vid_s, body, code, resp)

    http_acc["wall_s"] += time.perf_counter() - t_batch0
    http_acc["n"] += 1
    bn = int(http_acc["n"])
    if bn % 10 == 0 or bn <= 3:
        avg_ms = 1000.0 * float(http_acc["wall_s"]) / max(1, bn)
        print(
            f"[timing] mcap_http_batch #{bn} size={len(batch)} workers={mcap_workers} "
            f"cum_http_wall={_fmt_s(float(http_acc['wall_s']))} avg_batch={avg_ms:.0f}ms",
            flush=True,
        )


def run_mcap_import(
    videos: Iterator[dict[str, Any]],
    *,
    base_url: str,
    headers: dict[str, str],
    ctx: ssl.SSLContext,
    import_batch: str,
    source_db: str,
    owner: str,
    max_videos: int,
    dry_run: bool,
    mcap_workers: int = 8,
) -> tuple[dict[str, str], dict[str, str], int, int, int, int]:
    """Returns (mcap_by_video, gcs_by_video, created, exists, skip, fail)."""
    t_phase0 = time.perf_counter()
    mcap_by_video: dict[str, str] = {}
    gcs_by_video: dict[str, str] = {}
    mcap_workers = max(1, int(mcap_workers))
    chunk_size = max(32, mcap_workers * 4)
    url = f"{base_url}/api/v1/mcap-files"
    lock = threading.Lock()
    counters = [0, 0, 0, 0, 0]  # created, exists, fail, processed, skip
    http_acc: dict[str, float | int] = {"wall_s": 0.0, "n": 0}
    progress_clock = {"t": t_phase0}
    body_build_s = 0.0

    batch: list[tuple[str, dict[str, Any]]] = []
    i = 0
    for video in videos:
        i += 1
        if max_videos and i > max_videos:
            break
        vid = video.get("id")
        if vid is None:
            with lock:
                counters[4] += 1
            continue
        vid_s = str(vid)
        try:
            tb = time.perf_counter()
            body = build_mcap_body(
                video,
                import_batch=import_batch,
                source_db=source_db,
                owner=owner,
            )
            body_build_s += time.perf_counter() - tb
        except ValueError as e:
            print(f"[mcap skip] video={vid_s} reason={e}", file=sys.stderr)
            with lock:
                counters[4] += 1
            continue

        batch.append((vid_s, body))
        if len(batch) >= chunk_size:
            _flush_mcap_http_batch(
                batch,
                url=url,
                headers=headers,
                ctx=ctx,
                mcap_workers=mcap_workers,
                dry_run=dry_run,
                mcap_by_video=mcap_by_video,
                gcs_by_video=gcs_by_video,
                lock=lock,
                counters=counters,
                http_acc=http_acc,
                progress_clock=progress_clock,
            )
            batch = []

    _flush_mcap_http_batch(
        batch,
        url=url,
        headers=headers,
        ctx=ctx,
        mcap_workers=mcap_workers,
        dry_run=dry_run,
        mcap_by_video=mcap_by_video,
        gcs_by_video=gcs_by_video,
        lock=lock,
        counters=counters,
        http_acc=http_acc,
        progress_clock=progress_clock,
    )

    mcap_created, mcap_exists, mcap_fail, proc, mcap_skip = (
        counters[0],
        counters[1],
        counters[2],
        counters[3],
        counters[4],
    )
    wall = time.perf_counter() - t_phase0
    nb = max(1, int(http_acc["n"]))
    print(
        f"mcap_files: created={mcap_created} exists={mcap_exists} skipped={mcap_skip} failed={mcap_fail}",
        flush=True,
    )
    print(
        f"[timing] mcap_phase wall={_fmt_s(wall)} build_body={_fmt_s(body_build_s)} "
        f"http_batches_wall={_fmt_s(float(http_acc['wall_s']))} batches={nb} "
        f"avg_batch={1000.0 * float(http_acc['wall_s']) / nb:.0f}ms "
        f"processed_http={proc} skipped_pre_http={mcap_skip}",
        flush=True,
    )
    return mcap_by_video, gcs_by_video, mcap_created, mcap_exists, mcap_skip, mcap_fail


def _flush_asset_http_batch(
    batch: list[tuple[dict[str, Any], str, tuple[str, int, int]]],
    *,
    url: str,
    headers: dict[str, str],
    ctx: ssl.SSLContext,
    asset_workers: int,
    dry_run: bool,
    dedupe_segment_windows: bool,
    seen_window: set[tuple[str, int, int]],
    per_video_seg_index: dict[str, int],
    lock: threading.Lock,
    counters: list[int],  # [created, fail, seg_n]
    http_acc: dict[str, float | int],
    progress_clock: dict[str, float],
) -> None:
    """POST pre-built asset bodies; seg_n counts successful creates."""
    if not batch:
        return
    t_batch0 = time.perf_counter()

    def _mark_success(body: dict[str, Any], vid_s: str, win_key: tuple[str, int, int]) -> None:
        si = body.get("segment_index")
        if isinstance(si, int):
            per_video_seg_index[vid_s] = si
        if dedupe_segment_windows:
            seen_window.add(win_key)

    if dry_run:
        with lock:
            for body, vid_s, win_key in batch:
                print(json.dumps({"POST": "/api/v1/assets", "body": body}, ensure_ascii=False))
                counters[0] += 1
                counters[2] += 1
                _mark_success(body, vid_s, win_key)
                if counters[2] % 500 == 0:
                    now = time.perf_counter()
                    dt = now - progress_clock["t"]
                    progress_clock["t"] = now
                    r = 500.0 / dt if dt > 0 else 0.0
                    print(
                        f"... asset progress {counters[2]} ok={counters[0]} fail={counters[1]} "
                        f"last500_wall={_fmt_s(dt)} ~{r:.1f}/s",
                        flush=True,
                    )
        http_acc["wall_s"] += time.perf_counter() - t_batch0
        http_acc["n"] += 1
        return

    def handle_one(body: dict[str, Any], vid_s: str, win_key: tuple[str, int, int], code: int, resp: Any) -> None:
        with lock:
            if 200 <= code < 300:
                counters[0] += 1
                counters[2] += 1
                _mark_success(body, vid_s, win_key)
            else:
                counters[1] += 1
                print(f"[asset FAIL] video={vid_s} code={code} resp={resp}", file=sys.stderr)
            if counters[2] % 500 == 0 and counters[2] > 0:
                now = time.perf_counter()
                dt = now - progress_clock["t"]
                progress_clock["t"] = now
                r = 500.0 / dt if dt > 0 else 0.0
                print(
                    f"... asset progress {counters[2]} ok={counters[0]} fail={counters[1]} "
                    f"last500_wall={_fmt_s(dt)} ~{r:.1f}/s",
                    flush=True,
                )

    if asset_workers <= 1:
        for body, vid_s, win_key in batch:
            code, resp = http_json("POST", url, body, headers, ctx)
            handle_one(body, vid_s, win_key, code, resp)
    else:
        with ThreadPoolExecutor(max_workers=asset_workers) as ex:
            future_map: dict[Any, tuple[dict[str, Any], str, tuple[str, int, int]]] = {}
            for body, vid_s, win_key in batch:
                fut = ex.submit(http_json, "POST", url, body, headers, ctx)
                future_map[fut] = (body, vid_s, win_key)
            for fut in as_completed(future_map):
                body, vid_s, win_key = future_map[fut]
                try:
                    code, resp = fut.result()
                except Exception as e:
                    with lock:
                        counters[1] += 1
                        print(f"[asset FAIL] video={vid_s} err={e}", file=sys.stderr)
                    continue
                handle_one(body, vid_s, win_key, code, resp)

    http_acc["wall_s"] += time.perf_counter() - t_batch0
    http_acc["n"] += 1
    bn = int(http_acc["n"])
    if bn % 10 == 0 or bn <= 3:
        avg_ms = 1000.0 * float(http_acc["wall_s"]) / max(1, bn)
        print(
            f"[timing] asset_http_batch #{bn} size={len(batch)} workers={asset_workers} "
            f"cum_http_wall={_fmt_s(float(http_acc['wall_s']))} avg_batch={avg_ms:.0f}ms",
            flush=True,
        )


def run_asset_import(
    segments: Iterator[dict[str, Any]],
    *,
    mcap_by_video: dict[str, str],
    gcs_by_video: dict[str, str],
    base_url: str,
    headers: dict[str, str],
    ctx: ssl.SSLContext,
    import_batch: str,
    source_db: str,
    reviewer: str,
    owner: str,
    max_segments: int,
    dry_run: bool,
    dedupe_segment_windows: bool = True,
    asset_workers: int = 8,
) -> tuple[int, int, int]:
    """Returns (asset_created, asset_fail, dup_skipped)."""
    t_phase0 = time.perf_counter()
    asset_workers = max(1, int(asset_workers))
    chunk_size = max(32, asset_workers * 4)
    url = f"{base_url}/api/v1/assets"
    lock = threading.Lock()
    counters = [0, 0, 0]  # created, fail, seg_n (success count toward max_segments)
    http_acc: dict[str, float | int] = {"wall_s": 0.0, "n": 0}
    progress_clock = {"t": t_phase0}
    body_build_s = 0.0
    dup_skipped = 0
    per_video_seg_index: dict[str, int] = {}
    seen_window: set[tuple[str, int, int]] = set()
    batch: list[tuple[dict[str, Any], str, tuple[str, int, int]]] = []

    for seg in segments:
        if max_segments and counters[2] >= max_segments:
            break

        vid = seg.get("video_id")
        if vid is None:
            continue
        vid_s = str(vid)
        mcap_id = mcap_by_video.get(vid_s)
        if not mcap_id:
            continue
        gcs_path = gcs_by_video.get(vid_s, "")

        st = int(seg.get("start_timestamp") or 0)
        et = int(seg.get("end_timestamp") or 0)
        if st <= 0 or et <= st:
            continue

        win_key = (vid_s, st, et)
        if dedupe_segment_windows and win_key in seen_window:
            dup_skipped += 1
            continue

        idx = per_video_seg_index.get(vid_s, 0) + 1

        try:
            tb = time.perf_counter()
            body = build_asset_body(
                seg,
                mcap_file_id=mcap_id,
                gcs_path=gcs_path,
                reviewer=reviewer,
                owner=owner,
                import_batch=import_batch,
                source_db=source_db,
                segment_index=idx,
            )
            body_build_s += time.perf_counter() - tb
        except ValueError as e:
            print(f"[asset skip] video={vid_s} reason={e}", file=sys.stderr)
            continue

        batch.append((body, vid_s, win_key))

        if len(batch) >= chunk_size:
            _flush_asset_http_batch(
                batch,
                url=url,
                headers=headers,
                ctx=ctx,
                asset_workers=asset_workers,
                dry_run=dry_run,
                dedupe_segment_windows=dedupe_segment_windows,
                seen_window=seen_window,
                per_video_seg_index=per_video_seg_index,
                lock=lock,
                counters=counters,
                http_acc=http_acc,
                progress_clock=progress_clock,
            )
            batch = []

    _flush_asset_http_batch(
        batch,
        url=url,
        headers=headers,
        ctx=ctx,
        asset_workers=asset_workers,
        dry_run=dry_run,
        dedupe_segment_windows=dedupe_segment_windows,
        seen_window=seen_window,
        per_video_seg_index=per_video_seg_index,
        lock=lock,
        counters=counters,
        http_acc=http_acc,
        progress_clock=progress_clock,
    )

    asset_created, asset_fail, okn = counters[0], counters[1], counters[2]
    wall = time.perf_counter() - t_phase0
    nb = max(1, int(http_acc["n"]))
    print(
        f"assets: created={asset_created} failed={asset_fail} dedupe_skipped={dup_skipped}",
        flush=True,
    )
    print(
        f"[timing] asset_phase wall={_fmt_s(wall)} build_body={_fmt_s(body_build_s)} "
        f"http_batches_wall={_fmt_s(float(http_acc['wall_s']))} batches={nb} "
        f"avg_batch={1000.0 * float(http_acc['wall_s']) / nb:.0f}ms "
        f"posted_ok={okn}",
        flush=True,
    )
    return asset_created, asset_fail, dup_skipped


def main() -> int:
    ap = argparse.ArgumentParser(description="Import collector postgres data into Databrew via API.")
    ap.add_argument("--videos-jsonl", type=Path, default=None, help="MCAP grace_videos export (JSONL).")
    ap.add_argument("--segments-jsonl", type=Path, default=None, help="Segment rows export (JSONL).")
    ap.add_argument(
        "--pg-dsn",
        default="",
        help="Read-only Postgres DSN to collector-db. Env COLLECTOR_PG_DSN if flag omitted. Ignored when --videos-jsonl is set.",
    )
    ap.add_argument("--base-url", default=os.environ.get("DATABREW_BASE_URL", "").rstrip("/"))
    ap.add_argument("--token", default=os.environ.get("GRACE_TOKEN", ""))
    ap.add_argument("--reviewer", required=True)
    ap.add_argument("--owner", default="")
    ap.add_argument("--import-batch", required=True)
    ap.add_argument("--source-db", default="postgres", help="Logical source DB name (collector-db).")
    ap.add_argument("--max-videos", type=int, default=0)
    ap.add_argument("--max-segments", type=int, default=0)
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--videos-only", action="store_true")
    ap.add_argument(
        "--assets-only",
        action="store_true",
        help="Skip POST /mcap-files; derive mcap_file_id from collector rows and import segment assets only. "
        "Use after MCAP rows already exist in Databrew. Env IMPORT_ASSETS_ONLY=1 (in-cluster) does the same.",
    )
    ap.add_argument(
        "--no-dedupe-segment-windows",
        action="store_true",
        help="Post one asset per collector row even when (video_id,start,end) repeats (not recommended).",
    )
    ap.add_argument(
        "--mcap-workers",
        type=int,
        default=int(os.environ.get("IMPORT_MCAP_WORKERS") or 8),
        help="Parallel HTTP workers for POST /mcap-files (default 8; env IMPORT_MCAP_WORKERS).",
    )
    ap.add_argument(
        "--asset-workers",
        type=int,
        default=int(os.environ.get("IMPORT_ASSET_WORKERS") or 8),
        help="Parallel HTTP workers for POST /assets after bodies are built in order (default 8; env IMPORT_ASSET_WORKERS).",
    )
    args = ap.parse_args()
    assets_only = bool(args.assets_only) or (os.environ.get("IMPORT_ASSETS_ONLY") or "").strip() == "1"

    use_jsonl = args.videos_jsonl is not None
    dsn = ((args.pg_dsn or os.environ.get("COLLECTOR_PG_DSN") or "").strip() if not use_jsonl else "")
    use_pg = bool(dsn)

    if not use_jsonl and not use_pg:
        print(
            "Provide either --videos-jsonl (JSONL export) or --pg-dsn / COLLECTOR_PG_DSN (in-cluster stream).",
            file=sys.stderr,
        )
        return 2
    if use_jsonl and not args.videos_jsonl.exists():
        print(f"Missing file: {args.videos_jsonl}", file=sys.stderr)
        return 2
    if use_jsonl and not args.videos_only and not args.segments_jsonl:
        print("With JSONL mode, pass --segments-jsonl or --videos-only.", file=sys.stderr)
        return 2
    if use_jsonl and args.segments_jsonl and not args.segments_jsonl.exists():
        print(f"Missing file: {args.segments_jsonl}", file=sys.stderr)
        return 2

    if not args.base_url:
        print("Set --base-url or DATABREW_BASE_URL", file=sys.stderr)
        return 2
    if not args.token:
        print("Set --token or GRACE_TOKEN", file=sys.stderr)
        return 2
    if args.videos_only and assets_only:
        print("--videos-only and --assets-only are mutually exclusive.", file=sys.stderr)
        return 2

    ctx = ssl.create_default_context()
    headers = {"X-Grace-Token": args.token, "Content-Type": "application/json"}
    owner = args.owner or args.reviewer

    t_run0 = time.perf_counter()

    if use_pg:
        video_iter: Iterator[dict[str, Any]] = iter_mcap_videos_pg(dsn)
    else:
        video_iter = _stream_jsonl(args.videos_jsonl)

    if assets_only:
        mcap_by_video, gcs_by_video, _ = build_mcap_reference_maps(
            video_iter,
            import_batch=args.import_batch,
            source_db=args.source_db,
            owner=owner,
            max_videos=args.max_videos,
        )
        mcap_fail = 0
    else:
        mcap_by_video, gcs_by_video, _, _, _, mcap_fail = run_mcap_import(
            video_iter,
            base_url=args.base_url,
            headers=headers,
            ctx=ctx,
            import_batch=args.import_batch,
            source_db=args.source_db,
            owner=owner,
            max_videos=args.max_videos,
            dry_run=args.dry_run,
            mcap_workers=args.mcap_workers,
        )

    t_after_mcap = time.perf_counter()

    if args.videos_only:
        print(
            f"[timing] run_total={_fmt_s(t_after_mcap - t_run0)} (mcap_only, no asset phase)",
            flush=True,
        )
        return 0 if mcap_fail == 0 else 1

    if use_pg:
        seg_iter: Iterator[dict[str, Any]] = iter_mcap_segments_pg(dsn)
    else:
        assert args.segments_jsonl is not None
        seg_iter = _stream_jsonl(args.segments_jsonl)

    _, asset_fail, _ = run_asset_import(
        seg_iter,
        mcap_by_video=mcap_by_video,
        gcs_by_video=gcs_by_video,
        base_url=args.base_url,
        headers=headers,
        ctx=ctx,
        import_batch=args.import_batch,
        source_db=args.source_db,
        reviewer=args.reviewer,
        owner=owner,
        max_segments=args.max_segments,
        dry_run=args.dry_run,
        dedupe_segment_windows=not args.no_dedupe_segment_windows,
        asset_workers=args.asset_workers,
    )
    t_end = time.perf_counter()
    mcap_label = "mcap_maps_slice" if assets_only else "mcap_slice"
    print(
        f"[timing] run_total={_fmt_s(t_end - t_run0)} {mcap_label}={_fmt_s(t_after_mcap - t_run0)} "
        f"asset_slice={_fmt_s(t_end - t_after_mcap)}"
        + (" (--assets-only, no POST /mcap-files)" if assets_only else ""),
        flush=True,
    )
    return 0 if mcap_fail == 0 and asset_fail == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())

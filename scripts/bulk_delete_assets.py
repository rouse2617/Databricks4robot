#!/usr/bin/env python3
"""
Bulk manage assets via API (delete/upload).

Supports:
- delete:
  - --all: discover all visible assets by paging POST /api/v1/queries/run
  - --from-json ids.json
  - --from-csv ids.csv
- upload:
  - --from-json payloads.json (array of create-asset payloads)
  - --from-csv payloads.csv (see --csv-format)

Auth:
- Uses X-Databrew-Token header.

Examples:
  BASE=https://api-cyber-databrew-dev.cyberorigin.ai TOKEN=dev-token \
    python3 scripts/bulk_delete_assets.py --action delete --all --execute

  BASE=https://api-cyber-databrew-dev.cyberorigin.ai TOKEN=dev-token \
    python3 scripts/bulk_delete_assets.py --action upload --from-json ./assets.json --execute
"""

from __future__ import annotations

import argparse
import csv
import json
import os
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from typing import Iterable


@dataclass
class ApiClient:
    base: str
    token: str
    timeout_sec: int = 30

    def _request(self, method: str, path: str, body: dict | None = None) -> tuple[int, dict]:
        url = f"{self.base.rstrip('/')}{path}"
        cmd = [
            "curl",
            "-sS",
            "--max-time",
            str(self.timeout_sec),
            "-w",
            "\\n%{http_code}",
            "-X",
            method,
            url,
            "-H",
            f"X-Databrew-Token: {self.token}",
            "-H",
            "Content-Type: application/json",
        ]
        if body is not None:
            cmd.extend(["--data", json.dumps(body, separators=(",", ":"))])
        try:
            p = subprocess.run(cmd, capture_output=True, text=True, check=False)
            if p.returncode != 0 and not p.stdout:
                return 0, {"error": p.stderr.strip() or "curl failed"}
            out = p.stdout
            lines = out.splitlines()
            if not lines:
                return 0, {"error": "empty response"}
            status_raw = lines[-1].strip()
            body_raw = "\n".join(lines[:-1]).strip()
            try:
                status = int(status_raw)
            except ValueError:
                status = 0
                body_raw = out.strip()
            if not body_raw:
                return status, {}
            try:
                return status, json.loads(body_raw)
            except json.JSONDecodeError:
                return status, {"raw": body_raw}
        except Exception as e:  # noqa: BLE001
            return 0, {"error": str(e)}

    def list_asset_ids(self, page_size: int = 100) -> list[str]:
        ids: list[str] = []
        page = 1
        while True:
            status, payload = self._request(
                "POST",
                "/api/v1/queries/run",
                {
                    "schema_version": "v1",
                    "mode": "structured",
                    "scope": {"resource": "assets"},
                    "page": {"page": page, "page_size": page_size},
                },
            )
            if status != 200:
                raise RuntimeError(f"list assets failed at page={page}: http={status} body={payload}")
            items = payload.get("items") or []
            if not items:
                break
            page_ids = [it.get("asset_id", "") for it in items if it.get("asset_id")]
            ids.extend(page_ids)
            print(f"listed page={page} count={len(page_ids)} total_so_far={len(ids)}")
            page += 1
        # Keep order while removing duplicates.
        return list(dict.fromkeys(ids))

    def delete_one(self, asset_id: str) -> tuple[str, bool, int, dict]:
        status, payload = self._request("DELETE", f"/api/v1/assets/{asset_id}")
        ok = status == 200 and payload.get("deleted") is True
        return asset_id, ok, status, payload

    def create_one(self, create_payload: dict) -> tuple[str, bool, int, dict]:
        status, payload = self._request("POST", "/api/v1/assets", create_payload)
        ok = status in (200, 201)
        asset_id = ""
        if isinstance(payload, dict):
            asset_id = str(payload.get("asset_id") or create_payload.get("asset_id") or "")
        return asset_id, ok, status, payload


def load_ids_from_json(path: str) -> list[str]:
    with open(path, "r", encoding="utf-8") as f:
        obj = json.load(f)
    if isinstance(obj, list):
        if obj and isinstance(obj[0], dict):
            ids = [x.get("asset_id", "") for x in obj]
        else:
            ids = [str(x) for x in obj]
    elif isinstance(obj, dict):
        if isinstance(obj.get("asset_ids"), list):
            ids = [str(x) for x in obj["asset_ids"]]
        elif isinstance(obj.get("items"), list):
            ids = [x.get("asset_id", "") for x in obj["items"] if isinstance(x, dict)]
        else:
            raise ValueError("JSON must be list, {asset_ids:[...]}, or {items:[{asset_id:...}]}")
    else:
        raise ValueError("Unsupported JSON shape")
    return [x for x in ids if x]


def load_ids_from_csv(path: str) -> list[str]:
    ids: list[str] = []
    with open(path, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        if reader.fieldnames and "asset_id" in reader.fieldnames:
            for row in reader:
                val = (row.get("asset_id") or "").strip()
                if val:
                    ids.append(val)
        else:
            f.seek(0)
            raw = csv.reader(f)
            for row in raw:
                if not row:
                    continue
                val = row[0].strip()
                if val and val.lower() != "asset_id":
                    ids.append(val)
    return ids


def load_payloads_from_json(path: str) -> list[dict]:
    with open(path, "r", encoding="utf-8") as f:
        obj = json.load(f)
    if not isinstance(obj, list):
        raise ValueError("Upload JSON must be an array of POST /api/v1/assets payload objects")
    payloads = [x for x in obj if isinstance(x, dict)]
    if len(payloads) != len(obj):
        raise ValueError("Upload JSON array must contain only objects")
    return payloads


def _drop_empty_values(d: dict) -> dict:
    out = {}
    for k, v in d.items():
        if v is None:
            continue
        if isinstance(v, str) and not v.strip():
            continue
        out[k] = v
    return out


def load_payloads_from_csv(path: str) -> list[dict]:
    payloads: list[dict] = []
    with open(path, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        if not reader.fieldnames:
            raise ValueError("CSV must contain header row")
        for i, row in enumerate(reader, start=2):
            if "payload_json" in row and (row.get("payload_json") or "").strip():
                try:
                    payload = json.loads(row["payload_json"])
                except json.JSONDecodeError as e:
                    raise ValueError(f"invalid payload_json at csv line {i}: {e}") from e
                if not isinstance(payload, dict):
                    raise ValueError(f"payload_json at csv line {i} must decode to object")
                payloads.append(payload)
                continue

            payload = {
                "asset_id": (row.get("asset_id") or "").strip(),
                "mcap_id": (row.get("mcap_id") or "").strip(),
                "title": (row.get("title") or "").strip(),
                "description": (row.get("description") or "").strip(),
            }
            if (row.get("tags_json") or "").strip():
                payload["tags"] = json.loads(row["tags_json"])
            else:
                scene = (row.get("scene") or "").strip()
                if scene:
                    payload["tags"] = {"scene": scene}
            if (row.get("custom_metadata_json") or "").strip():
                payload["custom_metadata"] = json.loads(row["custom_metadata_json"])
            payloads.append(_drop_empty_values(payload))
    return payloads


def parse_args(argv: Iterable[str]) -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Bulk delete assets through API")
    p.add_argument("--base", default=os.getenv("BASE", "http://localhost:8080"))
    p.add_argument("--token", default=os.getenv("TOKEN", "dev-token"))
    p.add_argument("--action", choices=["delete", "upload"], default="delete")
    p.add_argument("--all", action="store_true", help="Discover all visible assets via queries/run")
    p.add_argument("--from-json", dest="from_json", help="Path to JSON containing asset IDs")
    p.add_argument("--from-csv", dest="from_csv", help="Path to CSV containing asset IDs")
    p.add_argument("--workers", type=int, default=8, help="Concurrent delete workers")
    p.add_argument("--execute", action="store_true", help="Actually execute deletion")
    return p.parse_args(list(argv))


def main(argv: Iterable[str]) -> int:
    args = parse_args(argv)
    client = ApiClient(base=args.base, token=args.token)

    if not any([args.all, args.from_json, args.from_csv]):
        print("choose one source: --all or --from-json or --from-csv", file=sys.stderr)
        return 2

    if args.action == "delete":
        ids: list[str] = []
        if args.all:
            ids = client.list_asset_ids(page_size=100)
        elif args.from_json:
            ids = load_ids_from_json(args.from_json)
        elif args.from_csv:
            ids = load_ids_from_csv(args.from_csv)

        ids = list(dict.fromkeys(ids))
        print(f"target_count={len(ids)}")
        if not ids:
            print("nothing to delete")
            return 0

        if not args.execute:
            print("dry-run only. add --execute to perform deletion.")
            for aid in ids[:20]:
                print(f"- {aid}")
            if len(ids) > 20:
                print(f"... and {len(ids)-20} more")
            return 0

        ok_count = 0
        fail_count = 0
        failed: list[tuple[str, int, dict]] = []

        with ThreadPoolExecutor(max_workers=max(1, args.workers)) as ex:
            futs = [ex.submit(client.delete_one, aid) for aid in ids]
            for i, fut in enumerate(as_completed(futs), start=1):
                aid, ok, status, payload = fut.result()
                if ok:
                    ok_count += 1
                else:
                    fail_count += 1
                    failed.append((aid, status, payload))
                if i % 25 == 0 or i == len(ids):
                    print(f"progress={i}/{len(ids)} ok={ok_count} fail={fail_count}")

        print(f"done ok={ok_count} fail={fail_count}")
        if failed:
            print("failed items (first 20):")
            for aid, status, payload in failed[:20]:
                print(f"- {aid} http={status} body={payload}")
            return 1
        return 0

    payloads: list[dict] = []
    if args.from_json:
        payloads = load_payloads_from_json(args.from_json)
    elif args.from_csv:
        payloads = load_payloads_from_csv(args.from_csv)
    else:
        print("upload action requires --from-json or --from-csv", file=sys.stderr)
        return 2

    print(f"target_count={len(payloads)}")
    if not payloads:
        print("nothing to upload")
        return 0
    if not args.execute:
        print("dry-run only. add --execute to perform upload.")
        for obj in payloads[:3]:
            print(json.dumps(obj, ensure_ascii=False))
        if len(payloads) > 3:
            print(f"... and {len(payloads)-3} more")
        return 0

    ok_count = 0
    fail_count = 0
    failed: list[tuple[str, int, dict]] = []
    with ThreadPoolExecutor(max_workers=max(1, args.workers)) as ex:
        futs = [ex.submit(client.create_one, obj) for obj in payloads]
        for i, fut in enumerate(as_completed(futs), start=1):
            aid, ok, status, payload = fut.result()
            if ok:
                ok_count += 1
            else:
                fail_count += 1
                failed.append((aid, status, payload))
            if i % 25 == 0 or i == len(payloads):
                print(f"progress={i}/{len(payloads)} ok={ok_count} fail={fail_count}")

    print(f"done ok={ok_count} fail={fail_count}")
    if failed:
        print("failed items (first 20):")
        for aid, status, payload in failed[:20]:
            print(f"- {aid or '<unknown>'} http={status} body={payload}")
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

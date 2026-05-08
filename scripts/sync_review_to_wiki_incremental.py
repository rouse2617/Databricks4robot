#!/usr/bin/env python3
"""Incrementally sync docs/review markdown files to a Feishu wiki parent node.

- Uses SHA256 hash per file to skip unchanged docs.
- Keeps a state file with docx/wiki token mapping.
- Updates existing docs via overwrite+append chunks.
- Creates and moves new docs into target wiki node.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import time
from pathlib import Path

DEFAULT_EXCLUDE_FILES = {
    "outbox-test-plan.md",
}


def run(cmd: str, cwd: Path | None = None, timeout: int = 300) -> tuple[int, str]:
    p = subprocess.run(
        cmd,
        shell=True,
        cwd=str(cwd) if cwd else None,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        timeout=timeout,
    )
    return p.returncode, p.stdout


def parse_json(out: str) -> dict:
    out = out.strip()
    m = re.search(r"(\{[\s\S]*\})\s*$", out)
    if m:
        return json.loads(m.group(1))
    return json.loads(out)


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        while True:
            b = f.read(1024 * 1024)
            if not b:
                break
            h.update(b)
    return h.hexdigest()


def first_title(md_text: str, fallback: str) -> str:
    for line in md_text.splitlines():
        if line.startswith("# "):
            return line[2:].strip()
    return fallback


def split_md_safe(text: str, max_chars: int = 15000) -> list[str]:
    lines = text.split("\n")
    units: list[str] = []
    buf: list[str] = []
    in_code = False

    def flush_text() -> None:
        nonlocal buf
        if not buf:
            return
        s = "\n".join(buf)
        buf = []
        for p in re.split(r"\n{2,}", s):
            if p.strip():
                units.append(p)

    for line in lines:
        if in_code:
            buf.append(line)
            if line.strip() == "```":
                units.append("\n".join(buf))
                buf = []
                in_code = False
            continue

        if line.strip().startswith("```"):
            flush_text()
            in_code = True
            buf.append(line)
            continue

        buf.append(line)

    if in_code:
        units.append("\n".join(buf))
    else:
        flush_text()

    chunks: list[str] = []
    cur: list[str] = []
    n = 0
    for u in units:
        sep = 2 if cur else 0
        if len(u) > max_chars:
            if cur:
                chunks.append("\n\n".join(cur))
                cur = []
                n = 0
            chunks.append(u)
            continue
        if cur and n + sep + len(u) > max_chars:
            chunks.append("\n\n".join(cur))
            cur = [u]
            n = len(u)
        else:
            cur.append(u)
            n += sep + len(u)
    if cur:
        chunks.append("\n\n".join(cur))
    return chunks


def safe_dir(name: str) -> str:
    base = name.replace(".md", "")
    base = re.sub(r"[^0-9A-Za-z._-]+", "_", base).strip("._-")
    return base or "doc"


def load_children(space_id: str, parent_node: str) -> dict[str, dict]:
    cmd = (
        "lark-cli wiki nodes list "
        f"--params '{{\"space_id\":\"{space_id}\",\"parent_node_token\":\"{parent_node}\",\"page_size\":50}}' "
        "--page-all --format json"
    )
    rc, out = run(cmd)
    if rc != 0:
        raise RuntimeError(out)
    j = parse_json(out)
    items = j["data"]["items"]
    return {it["title"]: it for it in items if it.get("obj_type") == "docx"}


def sync(args: argparse.Namespace) -> int:
    source_dir = Path(args.source_dir).resolve()
    work_root = Path(args.work_dir).resolve()
    state_path = Path(args.state_file).resolve()
    work_root.mkdir(parents=True, exist_ok=True)

    if state_path.exists():
        state = json.loads(state_path.read_text(encoding="utf-8"))
    else:
        state = {"files": {}, "meta": {}}

    wiki_by_title = load_children(args.space_id, args.parent_node)

    updated: list[str] = []
    created: list[str] = []
    skipped: list[str] = []
    excluded: list[str] = []
    failed: list[tuple[str, str]] = []

    for md in sorted(source_dir.glob("*.md")):
        name = md.name
        if name in DEFAULT_EXCLUDE_FILES:
            excluded.append(name)
            continue
        text = md.read_text(encoding="utf-8")
        title = first_title(text, md.stem)
        digest = sha256_file(md)

        entry = state["files"].get(name, {})
        docx = entry.get("docx_token")
        node = entry.get("wiki_node_token")
        wiki_url = entry.get("wiki_url")

        if not docx and title in wiki_by_title:
            it = wiki_by_title[title]
            docx = it["obj_token"]
            node = it["node_token"]
            wiki_url = f"https://www.feishu.cn/wiki/{node}"

        if entry.get("sha256") == digest and docx:
            skipped.append(name)
            continue

        chunks = split_md_safe(text, max_chars=args.max_chars)
        wd = work_root / safe_dir(name)
        wd.mkdir(parents=True, exist_ok=True)
        for i, ch in enumerate(chunks):
            (wd / f"part{i:02d}.md").write_text(ch, encoding="utf-8")

        if docx:
            rc, out = run(
                f"lark-cli docs +update --api-version v2 --doc {docx} --command overwrite --doc-format markdown --content @./part00.md",
                cwd=wd,
                timeout=240,
            )
            if rc != 0:
                failed.append((name, "overwrite"))
                continue
            ok = True
            for i in range(1, len(chunks)):
                rc, out = run(
                    f"lark-cli docs +update --api-version v2 --doc {docx} --command append --doc-format markdown --content @./part{i:02d}.md",
                    cwd=wd,
                    timeout=240,
                )
                if rc != 0:
                    failed.append((name, f"append_{i}"))
                    ok = False
                    break
            if not ok:
                continue
            updated.append(name)
        else:
            rc, out = run(
                "lark-cli docs +create --api-version v2 --doc-format markdown --content @./part00.md",
                cwd=wd,
                timeout=240,
            )
            if rc != 0:
                failed.append((name, "create"))
                continue
            cj = parse_json(out)
            docx = cj["data"]["document"]["document_id"]

            ok = True
            for i in range(1, len(chunks)):
                rc, out = run(
                    f"lark-cli docs +update --api-version v2 --doc {docx} --command append --doc-format markdown --content @./part{i:02d}.md",
                    cwd=wd,
                    timeout=240,
                )
                if rc != 0:
                    failed.append((name, f"append_new_{i}"))
                    ok = False
                    break
            if not ok:
                continue

            rc, out = run(
                f"lark-cli wiki +move --as user --obj-type docx --obj-token {docx} --target-space-id {args.space_id} --target-parent-token {args.parent_node} --jq .data.node_token",
                timeout=240,
            )
            if rc != 0:
                failed.append((name, "move_to_wiki"))
                continue
            node = out.strip().splitlines()[-1].strip()
            wiki_url = f"https://www.feishu.cn/wiki/{node}"
            created.append(name)

        state["files"][name] = {
            "title": title,
            "sha256": digest,
            "docx_token": docx,
            "wiki_node_token": node,
            "wiki_url": wiki_url,
            "updated_at": time.time(),
        }

    state["meta"] = {
        "source_dir": str(source_dir),
        "space_id": args.space_id,
        "parent_node": args.parent_node,
        "last_run_at": time.time(),
    }
    state_path.write_text(json.dumps(state, ensure_ascii=False, indent=2), encoding="utf-8")

    print("SYNC_OK")
    print("updated", len(updated), updated)
    print("created", len(created), created)
    print("skipped", len(skipped))
    print("excluded", len(excluded), excluded)
    print("failed", len(failed), failed)
    print("state_file", state_path)

    return 1 if failed else 0


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--source-dir", required=True)
    p.add_argument("--space-id", required=True)
    p.add_argument("--parent-node", required=True)
    p.add_argument("--state-file", required=True)
    p.add_argument("--work-dir", required=True)
    p.add_argument("--max-chars", type=int, default=15000)
    args = p.parse_args()
    return sync(args)


if __name__ == "__main__":
    raise SystemExit(main())

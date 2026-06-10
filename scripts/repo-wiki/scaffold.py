#!/usr/bin/env python3
"""Scaffold repo-wiki pages from manifest.yaml using the Qoder page template.

Creates any page declared in docs/repo-wiki/manifest.yaml that does not yet exist
on disk, pre-filling the <cite> "Referenced Files" block (from the page's
`sources`), a Table of Contents, and the standard empty section skeleton. It
NEVER overwrites an existing page — fill pages by hand (or via sub-agents)
following docs/agents/skills/repo-wiki/references/page-conventions.md.

Usage:
    python3 scripts/repo-wiki/scaffold.py [--src DIR] [--list] [--force PATH ...]
"""
from __future__ import annotations

import argparse
import glob
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
DEFAULT_SRC = REPO_ROOT / "docs" / "repo-wiki"

# Standard section skeleton (Qoder layout). Titles -> ToC anchors are derived.
SECTIONS = [
    "Introduction",
    "Project Structure",
    "Core Components",
    "Architecture Overview",
    "Detailed Component Analysis",
    "Dependency Analysis",
    "Performance Considerations",
    "Troubleshooting Guide",
    "Conclusion",
    "Appendices",
]

PLACEHOLDER = "<!-- TODO: fill from sources; see docs/agents/skills/repo-wiki/references/page-conventions.md -->"


def anchor(title: str) -> str:
    return title.lower().replace(" ", "-")


_SKIP = ("_test.go", ".test.tsx", ".test.ts", "_test.py")


def _is_noise(rel: str) -> bool:
    return rel.endswith(_SKIP) or "__pycache__" in rel or "/testdata/" in rel


def expand_sources(src_root: Path, sources):
    """Expand globs/dirs in `sources` to concrete repo-relative files (sorted, capped)."""
    out = []
    seen = set()

    def add(rel: str):
        if rel not in seen and not _is_noise(rel):
            seen.add(rel)
            out.append(rel)

    for s in sources or []:
        p = REPO_ROOT / s
        if p.is_dir():
            files = sorted(q for q in p.rglob("*") if q.is_file())
        else:
            files = [Path(m) for m in sorted(glob.glob(str(p), recursive=True)) if Path(m).is_file()]
        if files:
            for m in files:
                add(m.resolve().relative_to(REPO_ROOT).as_posix())
        else:
            add(s)  # keep literal (file may be created later / intentional glob)
    # cap the cite block so scaffolds stay readable; author trims/extends later
    return out[:40]


def cite_block(files):
    if not files:
        return "<cite>\n**Referenced Files in This Document**\n</cite>"
    # blank line after the header so the list renders as a list (not lazy paragraph)
    lines = ["<cite>", "**Referenced Files in This Document**", ""]
    lines += [f"- [{f}](file://{f})" for f in files]
    lines.append("</cite>")
    return "\n".join(lines)


def toc_block():
    lines = ["## Table of Contents"]
    for i, s in enumerate(SECTIONS, 1):
        lines.append(f"{i}. [{s}](#{anchor(s)})")
    return "\n".join(lines)


def body_block():
    out = []
    for s in SECTIONS:
        out.append(f"## {s}")
        out.append(PLACEHOLDER)
        out.append("")
    return "\n".join(out).rstrip() + "\n"


def scaffold_page(title: str, files) -> str:
    return (
        f"# {title}\n\n"
        f"{cite_block(files)}\n\n"
        f"{toc_block()}\n\n"
        f"{body_block()}"
    )


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--src", type=Path, default=DEFAULT_SRC)
    ap.add_argument("--list", action="store_true", help="list pages and status, create nothing")
    ap.add_argument("--force", nargs="*", default=[], help="page paths (rel to src) to overwrite")
    args = ap.parse_args()

    try:
        import yaml
    except ImportError:
        print("error: PyYAML required: pip install pyyaml", file=sys.stderr)
        sys.exit(1)

    src = args.src.resolve()
    manifest = src / "manifest.yaml"
    if not manifest.exists():
        print(f"error: manifest not found: {manifest}", file=sys.stderr)
        sys.exit(1)
    data = yaml.safe_load(manifest.read_text(encoding="utf-8")) or {}
    pages = data.get("pages", [])

    created = existing = 0
    force = set(args.force)
    for entry in pages:
        rel = entry.get("path")
        if not rel:
            continue
        dest = src / rel
        is_new = not dest.exists()
        if args.list:
            print(f"{'NEW ' if is_new else 'have'}  {rel}")
            continue
        if not is_new and rel not in force:
            existing += 1
            continue
        files = expand_sources(src, entry.get("sources"))
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(scaffold_page(entry.get("title") or dest.stem, files), encoding="utf-8")
        created += 1
        print(f"created {rel}")

    if not args.list:
        print(f"\nscaffold: {created} created, {existing} already existed ({len(pages)} pages in manifest)")


if __name__ == "__main__":
    main()

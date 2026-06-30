#!/usr/bin/env python3
"""Wiki-sync engine — detect code↔wiki divergence from docs/repo-wiki/manifest.yaml.

The manifest declares, per page, the `sources:` globs the page is derived from.
This module inverts that into a code-file → owning-pages index and uses it to:

  owners <file>...        Print which wiki page(s) document each given file.
  check  [--base REF | --from REF | --files -]
                          Report code files whose owning wiki page(s) were NOT
                          updated in the same diff. Coverage is LENIENT: a changed
                          code file is "covered" if ANY one of its owning pages is
                          also in the diff. Exit 1 if divergence found, else 0.

Files matched by no page's `sources` are not wiki-relevant and are ignored, as are
paths in scripts/repo-wiki/.wiki-sync-ignore. Source of truth stays the markdown
under docs/repo-wiki/.

Consumers: the CI gate (.github/workflows/repo-wiki-divergence.yml), the pre-push
advisory (scripts/ci-local.sh), and coding agents (repo-wiki skill update mode).
"""
from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
DOCS_DIR = REPO_ROOT / "docs" / "repo-wiki"
MANIFEST = DOCS_DIR / "manifest.yaml"
IGNORE_FILE = Path(__file__).resolve().parent / ".wiki-sync-ignore"
WIKI_PREFIX = "docs/repo-wiki/"

# ── undocumented "new area" advisory config ──────────────────────────────────
# When a PR ADDS code in a brand-new directory that no wiki page's `sources`
# covers, nudge the author to add/extend a page. Advisory only — never changes
# the check's exit code (so it never blocks, even in Phase B enforcement).
NEW_AREA_MODE = "directory"        # "directory": only flag new dirs (not files in existing dirs); "off": disable
NEW_AREA_MIN_FILES = 1             # threshold for a TOP-LEVEL new area (parent dir depth <= TOP_LEVEL_PARENT_DEPTH)
NESTED_NEW_AREA_MIN_FILES = 3      # threshold for a deeper/nested new area
TOP_LEVEL_PARENT_DEPTH = 1         # parent path with <= this many segments counts as top-level (0=repo root, 1=e.g. backend/)
UNOWNED_NEW_AREA_SEVERITY = "advisory"   # "advisory" (never affects exit) — reserved for future "block" option

_WILDCARD = re.compile(r"[*?\[]")


def die(msg: str, code: int = 2):
    print(f"wiki_sync: {msg}", file=sys.stderr)
    sys.exit(code)


# ── glob/source matching ────────────────────────────────────────────────────

def _compile(src: str) -> re.Pattern:
    """Compile a `sources` entry into a regex matched against repo-relative paths.

    - No wildcard → file-or-directory prefix: `a/b` matches `a/b` and `a/b/...`
      but NOT `a/bc`.
    - `**` → any chars (crosses `/`); `*` → any non-`/`; `?` → one non-`/`.
    """
    src = src.strip().rstrip("/")
    if not _WILDCARD.search(src):
        return re.compile(re.escape(src) + r"(/.*)?$")
    out, i, n = [], 0, len(src)
    while i < n:
        c = src[i]
        if c == "*" and i + 1 < n and src[i + 1] == "*":
            out.append(".*")
            i += 2
        elif c == "*":
            out.append("[^/]*")
            i += 1
        elif c == "?":
            out.append("[^/]")
            i += 1
        else:
            out.append(re.escape(c))
            i += 1
    return re.compile("".join(out) + r"$")


def load_pages():
    try:
        import yaml
    except ImportError:
        die("PyYAML required: pip install -r scripts/repo-wiki/requirements-mkdocs.txt")
    if not MANIFEST.exists():
        die(f"manifest not found: {MANIFEST}")
    data = yaml.safe_load(MANIFEST.read_text(encoding="utf-8")) or {}
    pages = []
    for entry in data.get("pages", []):
        path = entry.get("path")
        if not path:
            continue
        patterns = [_compile(s) for s in (entry.get("sources") or [])]
        pages.append((path, patterns))
    return pages


def load_ignore():
    pats = []
    if IGNORE_FILE.exists():
        for line in IGNORE_FILE.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if line and not line.startswith("#"):
                pats.append(_compile(line))
    return pats


def owners_of(path: str, pages) -> list[str]:
    """Wiki page paths (relative to docs/repo-wiki/) that document `path`."""
    return [page for page, pats in pages if any(p.match(path) for p in pats)]


# ── git ───────────────────────────────────────────────────────────────────

def _git(*args) -> str:
    try:
        return subprocess.check_output(
            ["git", *args], cwd=REPO_ROOT, text=True, stderr=subprocess.DEVNULL
        )
    except subprocess.CalledProcessError:
        die(f"git {' '.join(args)} failed; is the base ref fetched?", 2)


def resolve_range(base=None, frm=None):
    """Return (diff_range_args, base_rev) or (None, None) for explicit-files mode.

    base → three-dot (REF...HEAD), base_rev = merge-base(REF, HEAD).
    frm  → two-dot   (REF..HEAD),  base_rev = REF.
    """
    if base:
        base_rev = _git("merge-base", base, "HEAD").strip() or base
        return [f"{base}...HEAD"], base_rev
    if frm:
        return [frm, "HEAD"], frm
    die("check needs --base, --from, or --files")


def changed_files(diff_range=None, files=None) -> list[str]:
    if files is not None:
        return [f.strip() for f in files if f.strip()]
    out = _git("diff", "--name-only", *diff_range)
    return [l for l in out.splitlines() if l.strip()]


def added_files(diff_range) -> list[str]:
    out = _git("diff", "--name-only", "--diff-filter=A", *diff_range)
    return [l for l in out.splitlines() if l.strip()]


def base_dir_set(base_rev) -> set[str]:
    """All directory paths present in the tree at base_rev."""
    out = _git("ls-tree", "-r", "-d", "--name-only", base_rev)
    return {l for l in out.splitlines() if l.strip()}


def new_area_root(path: str, base_dirs: set[str]) -> str | None:
    """Shallowest ancestor directory of `path` that did NOT exist at base.

    Returns None when every ancestor dir already existed (file added to an
    existing directory — not a "new area" in directory mode).
    """
    segs = path.split("/")[:-1]  # directory components only
    prefix = ""
    for seg in segs:
        prefix = seg if not prefix else f"{prefix}/{seg}"
        if prefix not in base_dirs:
            return prefix
    return None


def find_new_areas(diff_range, base_rev, pages, ignore):
    """Group UNOWNED added files by their new-area root dir; apply thresholds.

    Returns a sorted list of (area_root, [files]) that meet their threshold.
    """
    if NEW_AREA_MODE != "directory":
        return []
    base_dirs = base_dir_set(base_rev)
    grouped: dict[str, list[str]] = {}
    for f in added_files(diff_range):
        if f.startswith(WIKI_PREFIX):
            continue
        if any(p.match(f) for p in ignore):
            continue
        if owners_of(f, pages):          # owned → handled by divergence, not "new area"
            continue
        root = new_area_root(f, base_dirs)
        if root is None:                 # added to an existing dir → skip in directory mode
            continue
        grouped.setdefault(root, []).append(f)

    flagged = []
    for root, files in sorted(grouped.items()):
        parent = root.rsplit("/", 1)[0] if "/" in root else ""
        parent_depth = 0 if parent == "" else parent.count("/") + 1
        threshold = NEW_AREA_MIN_FILES if parent_depth <= TOP_LEVEL_PARENT_DEPTH else NESTED_NEW_AREA_MIN_FILES
        if len(files) >= threshold:
            flagged.append((root, sorted(files)))
    return flagged


# ── commands ────────────────────────────────────────────────────────────────

def cmd_owners(args) -> int:
    pages = load_pages()
    for f in args.files:
        owners = owners_of(f, pages)
        print(f"{f}: " + (", ".join(owners) if owners else "(no wiki page documents this)"))
    return 0


def cmd_check(args) -> int:
    pages = load_pages()
    ignore = load_ignore()
    explicit = sys.stdin.read().splitlines() if args.files == ["-"] else (args.files or None)
    if explicit is not None:
        diff_range, base_rev = None, None
        changed = changed_files(files=explicit)
    else:
        diff_range, base_rev = resolve_range(base=args.base, frm=getattr(args, "from"))
        changed = changed_files(diff_range=diff_range)

    wiki_changed = {
        f[len(WIKI_PREFIX):] for f in changed
        if f.startswith(WIKI_PREFIX) and f.endswith(".md")
    }

    violations = []  # (code_file, [owning pages])
    for f in changed:
        if f.startswith(WIKI_PREFIX):
            continue                          # a wiki file, not code
        if any(p.match(f) for p in ignore):
            continue                          # never-documented path
        owners = owners_of(f, pages)
        if not owners:
            continue                          # not wiki-relevant (see new-area advisory)
        if not (set(owners) & wiki_changed):  # lenient: any one owner suffices
            violations.append((f, owners))

    # advisory: brand-new undocumented areas (needs a git range, not --files mode)
    new_areas = find_new_areas(diff_range, base_rev, pages, ignore) if diff_range else []

    _report(violations, new_areas, args.format)
    return 1 if violations else 0           # advisory never affects exit code


def _report(violations, new_areas, fmt):
    if not violations and not new_areas:
        msg = "✅ Wiki sync OK — no documented code changed without its wiki page."
        print(msg if fmt == "text" else f"<!-- wiki-sync -->\n{msg}")
        return

    if fmt == "markdown":
        lines = ["<!-- wiki-sync -->"]
        if violations:
            pages_to_touch = sorted({p for _, owners in violations for p in owners})
            lines += [
                "### 📝 Wiki may be out of date",
                "",
                "These changed files are documented by wiki pages that were **not** updated "
                "in this PR. Update the relevant page(s) under `docs/repo-wiki/`, or add the "
                "`wiki-exempt` label if this change needs no doc update.",
                "",
                "| Changed file | Documented by (update one) |",
                "| --- | --- |",
                *[f"| `{f}` | {', '.join(f'`{o}`' for o in owners)} |" for f, owners in violations],
                "",
                f"**Suggested pages to review:** {', '.join(f'`{p}`' for p in pages_to_touch)}",
            ]
        if new_areas:
            if violations:
                lines.append("")
            lines += [
                "### ℹ️ New code with no wiki coverage (advisory)",
                "",
                "These newly added directories aren't documented by any wiki page "
                "(no `sources` glob in `manifest.yaml` matches them). Consider adding a "
                "page, or extending an existing page's `sources`. **Advisory only — does "
                "not block.**",
                "",
                "| New area | Added files |",
                "| --- | --- |",
                *[f"| `{root}/` | {len(files)} |" for root, files in new_areas],
            ]
        lines.append("")
        lines.append("_Tip: `python3 scripts/repo-wiki/wiki_sync.py owners <file>` lists a file's pages._")
        print("\n".join(lines))
        return

    # text
    if violations:
        print(f"Wiki may be out of date — {len(violations)} documented file(s) changed "
              "without their wiki page:")
        for f, owners in violations:
            print(f"  {f}\n    → update one of: {', '.join(owners)}")
        print("Add the `wiki-exempt` label (or update the pages) to clear this.")
    if new_areas:
        print("New code with no wiki coverage (advisory — does not block):")
        for root, files in new_areas:
            print(f"  {root}/  ({len(files)} added file(s)) — add a page or extend a page's sources")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)

    po = sub.add_parser("owners", help="list wiki pages that document the given file(s)")
    po.add_argument("files", nargs="+")
    po.set_defaults(func=cmd_owners)

    pc = sub.add_parser("check", help="report code changes whose wiki pages weren't updated")
    g = pc.add_mutually_exclusive_group()
    g.add_argument("--base", help="base ref; diff BASE...HEAD (three-dot)")
    g.add_argument("--from", dest="from", help="base ref; diff FROM..HEAD (two-dot)")
    pc.add_argument("--files", nargs="*", help="explicit file list (or '-' to read stdin)")
    pc.add_argument("--format", choices=["text", "markdown"], default="text")
    pc.set_defaults(func=cmd_check)

    args = ap.parse_args()
    sys.exit(args.func(args))


if __name__ == "__main__":
    main()

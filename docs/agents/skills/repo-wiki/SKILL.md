---
name: repo-wiki
description: Use when creating, refreshing, or reviewing the repository wiki under docs/repo-wiki/. Markdown is the single source of truth; the HTML view is a regenerable MkDocs Material build (scripts/repo-wiki/gen_mkdocs.py + mkdocs).
---

# Repo Wiki

**Core principle:** Markdown is the **single source of truth**. Every claim in a page must be verifiable against real code in this repo. HTML is a throwaway view — never edit it, always regenerate it.

The wiki lives at [`docs/repo-wiki/`](../../../repo-wiki/). Its shape and nav order are declared in [`docs/repo-wiki/manifest.yaml`](../../../repo-wiki/manifest.yaml). The HTML view is a MkDocs Material build generated into `docs/repo-wiki-site/` (gitignored), and published to GitHub Pages by `.github/workflows/repo-wiki-pages.yml`.

**Layout:** pages follow the **Qoder repo-wiki format** in English, at a deep level of detail (250–700 lines/page): a `<cite>` "Referenced Files" block, a Table of Contents, the standard section skeleton (Introduction → Project Structure → Core Components → Architecture Overview → Detailed Component Analysis → Dependency Analysis → Performance Considerations → Troubleshooting Guide → Conclusion → Appendices), mermaid diagrams, and `**Section sources**` / `**Diagram sources**` citations with real `file://path#Lx-Ly` line ranges. The full template and rules are in [`references/page-conventions.md`](references/page-conventions.md).

---

## Iron laws

```
1. SOURCE OF TRUTH IS MARKDOWN — never hand-edit docs/repo-wiki-site/ or mkdocs.yml.
2. EVERY CLAIM IS VERIFIED — read the real file before you write the sentence.
3. NO INVENTED PATHS, FLAGS, OR ENDPOINTS — if you can't find it in code, it doesn't go in the page.
```

---

## Three modes

Pick the mode that matches the request.

### Mode: bootstrap — generate the wiki from scratch

Use when there is no `docs/repo-wiki/` yet, or the user asks to regenerate it.

1. Survey the repo top-down: read root `README.md`, each component README (`backend/`, `Frontend/`, `sdk/`, `services/*`), `api/openapi.yaml`, `Makefile`, `deploy/`, `docs/review/`.
2. Derive the catalog tree: top-level pages (Overview, Quick Start, Troubleshooting) plus one `section` per major domain, each section holding an index page + deep sub-pages. A section may nest one more level via an optional `subsection` field (e.g. `Database Design` → `Data Model Design` → per-domain pages); see [`references/manifest-schema.md`](references/manifest-schema.md). Mirror the codebase's real modules — do not force-fit another repo's domains.
3. Write [`manifest.yaml`](references/manifest-schema.md) FIRST — declare each page's `title`, `path`, `section`, and `sources:` (the code paths/globs the page is derived from). The `sources` must be specific per page so citations and drift detection are accurate.
4. Run `python3 scripts/repo-wiki/scaffold.py` to create every page from the template (cite block + TOC + section skeleton).
5. Fill each page following [`references/page-conventions.md`](references/page-conventions.md): read the real `sources`, write detailed prose + mermaid, cite real `file://path#Lx-Ly` ranges. For large trees, fan out sub-agents per page/section.
6. Write/refresh `docs/repo-wiki/README.md` as the table of contents (mirror `manifest.yaml` order).
7. Run the **preview** mode to confirm it builds.

### Mode: update — refresh against current code

Use when the code has changed, or a page is stale/thin. This is the default for an existing wiki.

1. Read `manifest.yaml`. For each page (or just the page(s) in scope), collect its `sources:`.
2. Detect drift: `git log --oneline -- <sources>` since the wiki was last touched, or `git diff` a known-good ref. Anything changed in `sources` → that page is suspect.
3. For each suspect page: re-read the real source files, then correct/expand the markdown. Delete claims you can no longer verify.
4. Update that page's `sources:` in `manifest.yaml` if the code moved.
5. Keep README TOC and `manifest.yaml` order in sync.
6. Run **preview**.

Never "refresh" by trusting the existing prose — re-derive from code.

**Trigger — update the wiki in the SAME PR as the code.** When your change touches a
file that a wiki page documents, that page must be updated alongside the code (this is
checked by `.github/workflows/repo-wiki-divergence.yml` and warned locally by the
pre-push hook). To find which pages own your changed files:

```bash
python3 scripts/repo-wiki/wiki_sync.py owners <changed-file>...   # lists the pages to update
python3 scripts/repo-wiki/wiki_sync.py check --base origin/<base> # what the CI gate sees
```

Coverage is **lenient** — updating any one owning page clears a file. If the change
genuinely needs no doc update (pure refactor, test-only, generated code), add the
`wiki-exempt` (or `docs-only` / `infra-ci-only`) label to the PR instead.

### Mode: preview — review as HTML

Use to review rendered output, or whenever you finish editing. The HTML view is a
**MkDocs Material** build (search, dark mode, collapsible nav, live mermaid), reading
the same markdown (source of truth unchanged):

```bash
pip install -r scripts/repo-wiki/requirements-mkdocs.txt
python3 scripts/repo-wiki/gen_mkdocs.py             # manifest -> mkdocs.yml (re-run after manifest edits)
mkdocs serve                                        # http://localhost:8000
mkdocs build                                        # -> docs/repo-wiki-site/ (gitignored)
```

`gen_mkdocs.py` projects the manifest's sections/order into `mkdocs.yml`; `mkdocs_hooks.py` makes the `<cite>` blocks render. The hosted site is published to GitHub Pages by `.github/workflows/repo-wiki-pages.yml` on pushes to `main`. Never commit `docs/repo-wiki-site/` or `mkdocs.yml` (both generated).

---

## Layout

```
docs/repo-wiki/            # SOURCE OF TRUTH (committed)
  manifest.yaml            # nav order + per-page `sources:` mapping
  README.md                # table of contents
  overview.md              # top-level pages
  <section>/index.md       # per-catalog index page
  <section>/<page>.md      # deep sub-pages
docs/repo-wiki-site/       # generated MkDocs Material build, gitignored — never edit
mkdocs.yml                 # generated from manifest by gen_mkdocs.py, gitignored
scripts/repo-wiki/
  scaffold.py              # manifest -> empty templated pages (never overwrites)
  gen_mkdocs.py            # manifest -> mkdocs.yml (Material viewer)
  mkdocs_hooks.py          # build hook: render <cite> blocks in MkDocs
  requirements-mkdocs.txt  # mkdocs-material, pymdown-extensions, pyyaml
.github/workflows/repo-wiki-pages.yml   # builds + publishes to GitHub Pages
```

See [`references/manifest-schema.md`](references/manifest-schema.md) and [`references/page-conventions.md`](references/page-conventions.md).

---

## Done means

- Every edited page's claims are backed by a file in its `sources:`.
- `manifest.yaml` order, README TOC, and the on-disk pages agree.
- `python3 scripts/repo-wiki/gen_mkdocs.py && mkdocs build --strict` exits 0.
- `docs/repo-wiki-site/` and `mkdocs.yml` are NOT staged for commit.

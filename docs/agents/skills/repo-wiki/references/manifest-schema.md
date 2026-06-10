# manifest.yaml schema

`docs/repo-wiki/manifest.yaml` is the structural backbone of the wiki. It is the
single place that declares nav order and — critically — which code each page is
derived from. The HTML build reads it for navigation; the `update` mode reads it
to know what to re-verify.

## Shape

```yaml
title: cyber-databrew Repository Wiki   # site title (HTML header + index)
generated: 2026-06-01                    # ISO date of last full refresh (informational)

pages:
  - title: Quick Start
    path: quick-start.md                 # relative to docs/repo-wiki/
    sources:                             # code/docs this page is derived from
      - README.md
      - Makefile
      - deploy/local/README.md

  - title: Architecture Overview
    path: architecture/overview.md
    section: Architecture                # optional nav group label
    sources:
      - backend/**
      - docs/review/README.md

  - title: Asset Model
    path: data-model/asset-model.md
    section: Database Design             # level-1 nav group
    subsection: Data Model Design        # optional level-2 nav group (nests under section)
    sources:
      - backend/migrations/000_initial.sql
```

## Field rules

| Field | Required | Meaning |
|-------|----------|---------|
| `title` (top) | yes | Wiki title shown in HTML header and `index.html`. |
| `generated` | no | Date string. Convert relative dates to absolute (`YYYY-MM-DD`). |
| `pages[].title` | yes | Nav label and page `<h1>` fallback. |
| `pages[].path` | yes | Path relative to `docs/repo-wiki/`. Must exist on disk. |
| `pages[].section` | no | Groups pages under a heading in the sidebar. Pages with no `section` render at the top level. Order within a section follows list order. |
| `pages[].subsection` | no | Second-level nav group, nested under `section`. Requires `section` to be set. Pages sharing a `subsection` must be contiguous within their section; the sub-group opens at its first page. |
| `pages[].sources` | yes | List of repo-relative paths or globs the page is derived from. This is what `update` mode re-reads to detect drift. Keep it honest — list only what the page actually documents. |

## Conventions

- **Order = list order.** The sidebar and README TOC follow the order pages appear in `pages:`.
- **One page = one `path`.** Don't point two entries at the same file.
- **`sources` are real.** Every glob must match something. If a page documents code in three packages, list all three. Vague entries (`backend/**` for a page about one handler) defeat drift detection — be specific.
- **Keep it in sync.** Adding a page means adding a `pages:` entry AND a README TOC line. The build will still render an unlisted `.md` (filesystem fallback) but it won't be ordered or grouped.

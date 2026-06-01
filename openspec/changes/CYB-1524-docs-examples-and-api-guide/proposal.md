---
change-id: CYB-1524
slug: docs-examples-and-api-guide
title: Add schema examples and publish api-guide to doc site
type: fix
---

## Problem

Two gaps remain after CYB-1523:

1. OpenAPI schema properties lack `example` values — users see field names and types but don't know valid values for key fields like `lifecycle_state`, `split_method`, `asset_type`, etc.
2. `docs/review/api-guide.md` (2400+ lines of curl examples with success/error paths) exists only in the repo — not published on the Docusaurus doc site.

## Scope

| # | File | Change |
|---|------|--------|
| 1 | `api/openapi.yaml` | Add `example` values to request schema properties |
| 2 | `docs-site/docs/reference/api-guide.md` | Copy from `docs/review/api-guide.md` |
| 3 | `docs-site/sidebars.ts` | Add api-guide to sidebar |

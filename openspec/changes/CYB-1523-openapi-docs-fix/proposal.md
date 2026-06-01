---
change-id: CYB-1523
slug: openapi-docs-fix
title: Fix OpenAPI requestBody gaps and doc site usability
type: fix
---

## Problem

Users browsing the API reference at `/doc/api/reference` (Scalar UI from `api/openapi.yaml`) cannot effectively call the API. Four gaps prevent a developer from succeeding with just the online docs:

1. **5 POST endpoints lack `requestBody`** — Scalar UI shows no request schema, users don't know what to send.
2. **`batch_get` response YAML indentation error** — `items` is sibling of `properties` instead of child.
3. **`hideModels: true`** — shared schemas (Asset, Delivery, etc.) hidden from UI.
4. **Base URL inconsistency** — auth docs use `api-cyber-databrew.cyberorigin.ai` vs. actual `cyber-databrew.cyberorigin.ai`.

## Scope

| # | File | Change |
|---|------|--------|
| 1 | `api/openapi.yaml` | Add `requestBody` with `$ref` to existing schemas for 5 POST endpoints; fix batch_get YAML indent |
| 2 | `docs-site/docusaurus.config.ts` | Set `hideModels: false` |
| 3 | `docs-site/docs/getting-started/authentication.md` | Replace `api-cyber-databrew.cyberorigin.ai` → `cyber-databrew.cyberorigin.ai` |

## Non-goals

- Not migrating `docs/review/api-guide.md` into Docusaurus (separate CYB)
- Not adding OpenAPI `example` values to schemas (deferred)
- Not touching backend handler code

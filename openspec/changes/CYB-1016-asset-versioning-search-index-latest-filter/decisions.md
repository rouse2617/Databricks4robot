# Decisions — CYB-1016

## OpenSpec approval

- **2026-05-22**: User said 「ok，我们继续 刚才的开发」 — treated as checkpoint approval to implement on branch `feat/CYB-1016-asset-versioning-search-index-latest-filter`.

## Terminology (CYB-1013 alignment)

| Linear / old docs | Implementation |
|-------------------|----------------|
| `is_latest` | `is_current` |
| content `version` | `revision` (row `version` remains optimistic-lock) |

## Default current-only filter

- **PG**: append `(COALESCE(is_current, TRUE) = TRUE)` to list WHERE (legacy NULL `is_current` still visible).
- **ES**: bool filter — `is_current:true` OR `logical_asset_id` field missing (pre-versioning / legacy docs).
- Opt-out: `scope.include_history: true` or query param `?include_history=true`.

## Prior revision ES sync

- Do **not** change `outbox/es_subscriber.go` (off-limits). After promote, emit `asset_updated` for `prior_asset_id` so the subscriber rebuilds the demoted doc.

## SDK

- Deferred (same as CYB-1014); OpenAPI + api-guide + smoke required.

# Decisions — CYB-3714

## 2026-07-21 — YAML baseline over admin-API registration

- **Context**: `source` and `city` need to become known tag keys. Two ways: (a) commit them to `backend/config/tag_registry.yaml` (loaded at startup, refreshed on deploy), or (b) POST them via `POST /api/v1/admin/tag-registry` at runtime.
- **Decision**: Commit to YAML.
- **Alternatives**: Admin API — reversible, no deploy, but leaves the change out of git; harder to audit and roll back.
- **Rationale**: `tag_registry.yaml` is the code-owned baseline; the admin API layers on top for ad-hoc adds without a redeploy. `source` and `city` are foundational — every producer should be able to assume they exist. Committing to YAML makes them part of the platform contract, visible to code reviewers.

## 2026-07-21 — Backfill via shell script, not migration or one-off Go binary

- **Context**: 140 mcap need tag rows inserted. Options: (a) SQL migration (INSERT INTO asset_tags), (b) Go one-off command under `scripts/`, (c) Bash + curl calling the existing `POST /assets/:id/tags` endpoint.
- **Decision**: Bash + curl.
- **Alternatives**: SQL migration would bypass the tag-registry validation, source-type-name invariants, and audit-log emission — those live in the handler. A Go one-off works but adds a new binary target to build.
- **Rationale**: The existing HTTP endpoint already enforces every invariant. Reusing it means the backfill is exactly what the producer would do — no divergence in behavior. Bash + curl is minimal-surface and re-runnable against any environment.

## 2026-07-21 — Producer coordination deferred to a separate ticket

- **Context**: The proper long-term fix is VibeCap calling `POST /tags` after every mcap upload. That requires a VibeCap client change (external team).
- **Decision**: This ticket closes the databrew side only — YAML + backfill. Producer coordination is tracked in the ticket description as "Out of scope" and left for VibeCap owners to sequence.
- **Alternatives**: Modify the mcap-files `POST` handler to accept an inline `tags` array and expand server-side. Cleaner one-round-trip UX for the producer, but still requires VibeCap to send the field.
- **Rationale**: Backfill is the load-bearing deliverable — without it, existing mcap have no tags regardless of what future producers do. The producer-side change is an incremental improvement that can land whenever VibeCap is ready, without blocking or being blocked by this ticket.

## 2026-07-21 — city parsing is best-effort, tolerant of address format drift

- **Context**: `metadata.location.address` for pangzi's mcap looks like `"合肥新民医院, 合瓦路, 上城, 上城社区, 杏林街道, 庐阳区, 合肥市, 安徽省, 230061, 中国"`. The city is `"合肥市"`, but the string has 10 comma-separated segments and no fixed schema.
- **Decision**: Extract the first segment matching `<something>市` — Chinese city suffix. Skip city tag if no match.
- **Alternatives**: Geocode via a reverse geocoder — expensive, needs a network call and API key. Assume position of city segment — brittle across different addresses.
- **Rationale**: Best-effort with a graceful skip beats overfitting to one address format. Missing city tags on a subset is acceptable — the primary carrier of location is the tag itself; the JSONB address is preserved intact for later re-parsing.

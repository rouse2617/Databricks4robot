# CYB-3797 — mcap POST auto-extract metadata → tag

## Why

CYB-3714 backfill runs against a moving target: pangzi uploads ~1-2 mcap/min, so any coverage number falls immediately (76% → 74% → 60% → 50% on 2026-07-21). The only durable fix is to write the tags **at ingest time**, in the same transaction as the mcap create, so newly uploaded mcap arrive already tagged.

## What Changes

In `backend/internal/handlers/mcap/handler.go` `createFileTx`, after `assetRepo.InsertNew(txCtx, placeholder)`, call a new `extractAndInsertMetadataTags` step that reads the mcap's `metadata` JSONB and writes tag rows via `AssetTagRepository.Upsert`. All writes participate in the existing transaction so a failed tag insert rolls back the whole mcap create — but see "defensive skip" below.

Three rules (whitelist, hardcoded in Go):

| Metadata source | Tag key | Extraction |
|---|---|---|
| `metadata.vibecap_tasks` | `task` | Array-or-JSON-string of strings; one tag row per entry (multi-value) |
| `metadata.source_platform` | `source` | Plain string |
| `metadata.location.address` | `city` | Comma-split, iterate segments from back, take first `X市` that is ≥3 chars AND not ending in `超市`/`大厦`/`商店`/`商场`/`广场`/`医院` (POI filter) |

Every tag is written with `source_type="system"`, `source_name="mcap_ingest"` so downstream can distinguish auto-generated tags from human/algo/compliance labels.

**Whitelist strategy** (as clarified in chat 2026-07-21):
- Keys **not** in the three-rule set are ignored — never auto-become tags. `metadata` JSONB blob is preserved intact on the mcap row.
- Value shape mismatch (e.g. `vibecap_tasks` is a number, `location.address` is null) → skip that specific tag, log warning at INFO level, **do not** fail the mcap create.
- Adding a new producer / new metadata field to the rule set is a databrew-side PR that extends the three-rule slice — controlled evolution.

## Impact

**Runtime**
- Every `POST /api/v1/mcap-files` gets 0-N (typically 3-5) tag rows written alongside. Adds ~2-4 SQL statements per request; the tx overhead is small vs the existing mcap insert + asset insert + outbox append.
- New mcap upload → immediately queryable via `tags.task=X` / `tags.source=vibecap` / `tags.city=合肥市`. Coverage stays at 100% permanently.
- Historical mcap unaffected — the backfill script (CYB-3714) already covered ~488 records; the ~200 mcap uploaded between backfill end and this PR merge will remain untagged and can be caught by one final backfill run.

**API contract**
- Response body of `POST /mcap-files` is unchanged. No new fields in the request / response.
- `docs/review/api-guide.md` gets a one-line note under the mcap-files section.
- No OpenAPI change needed (the tag side-effect is not part of the mcap-files response shape).

**Off-limits**
- Not touched: `middleware/auth*`, `outbox/`, `migrations/`, `.env*`, `schemas/pg-phase0.sql`.
- Modifies: `handlers/mcap/handler.go` + adds `handlers/mcap/metadata_tags.go` + wires `assetTagRepo` in `cmd/server/core.go`.

## Acceptance

- New mcap POST (with `metadata.vibecap_tasks + source_platform + location.address` populated) triggers ≥3 tag rows written in the same transaction.
- Unit tests cover: 3 happy paths, unknown metadata keys ignored, `vibecap_tasks` shape variants (nil / array / JSON string / non-string non-array — the last must skip cleanly), and city parsing edge cases (single-segment address, `超市` in address, no `X市` in segments).
- Handler test asserts a mocked `AssetTagRepo.Upsert` is called with the expected inputs for a representative pangzi-shaped mcap.
- Post-deploy on dev: create a new mcap with the pangzi metadata shape → `GET /assets/:id` shows `tags: {task: ..., source: vibecap, city: 合肥市}` immediately.
- Post-deploy: `POST /queries/run where: {tags.source: vibecap}` coverage matches pangzi total (100%) for mcap created after this PR lands. Historical drift is handled by a follow-up one-shot backfill catch-up (CYB-3714 script re-run).

## Out of scope

- Backfill of the ~300 mcap uploaded between CYB-3714 backfill end and this PR merge — one-off CYB-3714 script re-run covers it.
- Producer-side change (VibeCap sending explicit `tags` field on the mcap-files POST) — deferred; this ticket makes the producer-side change unnecessary.
- Extending the rule set (grace_missions, cybercap_x) — separate PR when new producers arrive.
- Full-text search on `metadata.location.address` for terms like "新民医院" — that is CYB-3716.

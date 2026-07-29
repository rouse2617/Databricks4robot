# Decisions — CYB-4445

> Append-only. Each entry dated.

## 2026-07-29 — Match style: exact 8-char alphanumeric only

- **Context**: `mcap_file_id` filter could plausibly be substring (`ILIKE`), exact match, or auto-switch (exact if 8 chars, otherwise ignore). User picked exact 8-char.
- **Decision**: Server enforces `^[A-Za-z0-9]{8}$`; non-empty non-matching returns 400. Client validates the same pattern; below 8 chars, the debounced filter value is empty (param not sent).
- **Alternatives considered**:
  - `ILIKE '%q%'` — rejected: 8-char substring is too permissive (collisions), and breaks PK constant-time expectation.
  - Auto-switch with no client validation — rejected: confusing UX (half-typed → no result).
- **Rationale**: mirrors existing `CmdKSearch.isAssetId` (Frontend/src/components/CmdKSearch.tsx:35) and the schema `chk_mcap_file_id_format` CHECK; prevents accidental `SHORT`-style requests returning an empty list silently; PK lookup stays O(log n) (constant-time on PK).
- **How to apply**: same regex on both client and server; non-matching → `400 INVALID_ARGUMENT`. Document in `api-guide.md`.

## 2026-07-29 — Interface signature change strategy: append-only, no struct

- **Context**: Adding `mcapFileID` to `McapFileRepository.List` could be done by adding a 6th positional param or by introducing a `ListOptions` struct.
- **Decision**: Append `mcapFileID string` as the last positional parameter.
- **Alternatives considered**:
  - Replace existing positional args with `ListOptions { Page, PageSize, IngestState, Owner, McapFileID }` — rejected for scope: this is a single-feature PR, not a refactor.
  - Make `mcapFileID` a method on a new `McapFileFilter` type — rejected: not enough filters to warrant a new type yet.
- **Rationale**: matches AI-RULES.md §6 "new interface methods" rule: update all implementations and test stubs in the same commit. Only two stubs (`mockMcapRepo` in `handler_test.go:21`, `stubMcapRepo` in `builder_test.go:97`), so the blast radius is small. If/when a third filter is added later, we revisit a struct.
- **How to apply**: keep call-sites minimal — handler passes `c.Query("mcap_file_id")` straight through to repo.

## 2026-07-29 — Out of scope: `/mcap-files/${id}` route is missing

- **Context**: During exploration we confirmed that `CmdKSearch.tsx:94-99` "navigates" to `/mcap-files/${id}` for the 8-char MCAP quick-match, but `App.tsx:61` only registers `/mcap-files` (no `:id` subroute); therefore the `<Route path="*">` catches and redirects to `/dashboard`. CmdK's MCAP direct-link is effectively broken.
- **Decision**: Do NOT fix this in CYB-4445. Add the URL `?mcap_file_id=...` flow as the canonical "I know the ID" entry point.
- **Rationale**: bundling unrelated UI-route fixes into a backend-facing filter PR enlarges review surface and risk. The user's request was specifically about the `/mcap-files` list page's missing filter, not CmdK navigation.
- **How to apply**: file a separate Linear follow-up issue after CYB-4445 closes. Reference it in the PR description so future readers know it's deferred.

## 2026-07-29 — OpenSpec checkpoint approved

- **Context**: Per `AI-RULES.md §4`, after writing `proposal.md + tasks.md + design.md` the agent must stop and ask the user to confirm OpenSpec is OK.
- **Decision**: User replied "ok" → checkpoint cleared.
- **How to apply**: proceed with implementation in the order written in `tasks.md`. Verification tier: **L** (API surface + multi-module: backend handler/repo + frontend page + OpenAPI/api-guide).

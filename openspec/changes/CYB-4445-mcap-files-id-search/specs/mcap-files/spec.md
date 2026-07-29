# Spec delta — CYB-4445 — `mcap-files`

## ADDED Requirements

### Requirement: mcap-files.ID filter

`GET /api/v1/mcap-files` SHALL accept an optional `mcap_file_id` query parameter that, when set, restricts the list to the single row whose `mcap_file_id` exactly equals the parameter. When omitted or empty, this filter SHALL NOT be applied.

**Priority**: P1 (High) — improves operator retrieval on a 21k+ row table.

**Rationale**: `mcap_file_id` is the canonical 8-char identifier of an MCAP file; users frequently know the ID (e.g. from a Grace dashboard link or a previous session) and need to jump straight to its detail without paging through thousands of rows. The page already exposes `owner` (substring) and `ingest_state` (exact-match); ID-by-equality is the missing third axis.

#### Scenario: exact match returns the one row
- **Given** an MCAP file with `mcap_file_id = "LEMpjOmB"` exists in `mcap_files`
- **When** the client sends `GET /api/v1/mcap-files?mcap_file_id=LEMpjOmB`
- **Then** the response `items` array has length 1 and the single item's `mcap_file_id` equals `"LEMpjOmB"`, and `total` equals `1`

#### Scenario: combine with owner filter
- **Given** an MCAP file `LEMpjOmB` whose `owner` equals `"grace-pu"` exists, and at least one other MCAP file exists with `owner = "grace-pu"` but a different ID
- **When** the client sends `GET /api/v1/mcap-files?mcap_file_id=LEMpjOmB&owner=grace-pu`
- **Then** the response contains only `LEMpjOmB` (the intersection), and `total` reflects that single row

#### Scenario: non-8-char alphanumeric value rejected
- **Given** an arbitrary `mcap_file_id` query value, e.g. `"SHORT"` (5 chars) or `"ABCDEFGH "` (8 chars with trailing space), that does not match `^[A-Za-z0-9]{8}$`
- **When** the client sends the request
- **Then** the server responds `400` with error envelope `code: "INVALID_ARGUMENT"` and a message stating `mcap_file_id must be exactly 8 alphanumeric characters`

#### Scenario: below 8 characters is ignored by the FE
- **Given** the user has typed `LEMpjO` (7 chars) into the page filter input
- **When** the page's debounced filter commits
- **Then** the page does NOT include `mcap_file_id` in the request URL; the list is shown un-filtered (or as filtered by any other controls). The user must NOT see an empty list just because they have not finished typing.

#### Scenario: existing `?mcap_file_id=…` URL drawer-trigger unchanged
- **Given** a user lands on `/mcap-files?mcap_file_id=LEMpjOmB`
- **When** the page mounts
- **Then** the existing detail-drawer auto-open behaviour fires (calls `mcapFilesApi.get(...)` and opens `McapDetailDrawer`), and the list table loads independently of this URL flow

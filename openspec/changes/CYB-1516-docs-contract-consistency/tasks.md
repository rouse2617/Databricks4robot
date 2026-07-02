# Tasks — CYB-1516

## Context files
- `api/openapi.yaml` — public REST contract and generated docs source
- `backend/internal/httpresp/response.go` — standard error response helper
- `backend/routes/routes.go` — auth compatibility routes with ad-hoc errors
- `backend/routes/routes_test.go` — request id and route behavior tests
- `sdk/src/cyber_databrew_sdk/_requestor.py` — SDK error body parsing
- `sdk/src/cyber_databrew_sdk/exceptions.py` — SDK typed exceptions and fields
- `docs-site/docs/getting-started/quickstart.md` — quickstart SDK examples
- `docs-site/docs/guides/*.md` and `docs-site/docs/core-concepts/*.md` — user docs examples
- `docs/review/api-guide.md` — reviewer-facing API contract notes

## Implementation
- [x] [docs-site] Replace nonexistent SDK method examples with supported SDK/API patterns.
- [x] [docs-site] Add user-facing SDK error field guidance and examples.
- [x] [docs-site] Correct stale lifecycle/status enum references where they conflict with code/OpenAPI.
- [x] [openapi] Replace invalid `ErrorBody` schema references with the canonical error schema.
- [x] [docs/review] Document the canonical error envelope and SDK field mapping.
- [x] [backend] Return standard error envelopes from ad-hoc auth route failures without changing auth behavior.
- [x] [backend] Register existing documented event stream, audit search, and delivery lifecycle routes.
- [x] [backend] Add/adjust route tests for the standard error envelope.

## API contract sync
- [x] `api/openapi.yaml` updated for error schema consistency.
- [x] `docs/review/api-guide.md` updated with canonical error envelope and SDK mapping.
- [x] SDK behavior reviewed; no public SDK API change required.
- [x] Backend route tests cover changed runtime error response shape.
- [x] OpenSpec delta written for API/docs behavior.

## Verification
- [x] Run `cd docs-site && npm run build`.
- [x] Run backend route tests covering changed auth errors.
- [x] Run targeted OpenAPI schema reference check for invalid `ErrorBody`.
- [x] Run SDK unit tests or targeted SDK error tests if SDK code changes.

## Deploy verification
- [ ] If backend runtime changes are included, deploy backend dev and verify a failing auth request returns `code/message/request_id`.
- [ ] If docs-site changes are deployed, verify `/doc/overview` and relevant docs pages render without broken links.

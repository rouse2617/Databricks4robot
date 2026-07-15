# Tasks — CYB-3417 security hardening

## A2 — api_key scope guard
- [x] Add `PrivilegedScopes` set in `backend/internal/handlers/apikey/handler.go`
- [x] Insert scope-validation loop before `repo.Create`; reject privileged scopes for non-static-token callers with 403
- [x] Reject whitespace-only scope entries with 400
- [x] Unit tests: JWT+wildcard → 403, JWT+apikeys:manage → 403, StaticToken+wildcard → 201, JWT+normal scopes → 201, empty scope → 400

## A3 — constant-time token compare
- [x] Add unexported `tokenEquals` helper in `backend/internal/middleware/auth.go`
- [x] Replace `!=` in `StaticTokenAuth` (line 64)
- [x] Replace `!=` in `JWTAuth` X-Databrew-Token path (line 85)
- [x] Replace `!=` in `JWTAuth` JWT-fallback-to-static path (line 120)
- [x] Replace `!=` in `AdminTokenAuth` (line 149)
- [x] Unit tests: empty want fails closed, length mismatch rejects, exact match succeeds, Bearer prefix stripped

## A1 — email-login note only
- [x] Add security block comment in `routes/routes.go` explaining time-limited operator session semantics; behavior unchanged

## Off-limits declaration
- [x] Confirm user gave explicit approval to touch `backend/internal/middleware/auth*` (chat log)
- [ ] PR body flags off-limits + requests ≥2 reviewers

## Verification (Tier L required per off-limits + auth change)
- [ ] `cd backend && go build ./...`
- [ ] `cd backend && go test ./...` (full suite, not just touched packages)
- [ ] Local smoke via `scripts/api-guide-smoke.sh` against dev after deploy
- [ ] After deploy: curl POST `/admin/api-keys` with JWT + `scopes:["*"]` → expect 403; retry with static admin token → expect 201

## Contract sync
- [ ] `docs/review/api-guide.md` — add one line noting `*` / `apikeys:manage` requires static admin token
- [ ] `api/openapi.yaml` — add `403 FORBIDDEN` response to `/admin/api-keys` create operation
- [ ] `scripts/api-guide-smoke.sh` (or a targeted script) — add the JWT-vs-static privileged-scope assertion

## Out of scope (documented in proposal / decisions)
- [ ] tag_registry idempotency (immutable migration principle) — deferred
- [ ] rate limiting on email-login / api-keys — separate ticket
- [ ] error message sanitization (`err.Error()` → generic) — separate ticket
- [ ] `last_used_at` unbounded goroutine — separate ticket

## Deploy
- [ ] Merge to dev via PR (auto-merge OK; needs 1 review — off-limits requires 2 per policy)
- [ ] `deploy-dev.yml` runs (backend Cloud Run + frontend CF Worker; no migration this PR)
- [ ] Post-deploy: rerun 4 verify curls (see Verification)
- [ ] Update Linear CYB-3417 to Done + attach PR

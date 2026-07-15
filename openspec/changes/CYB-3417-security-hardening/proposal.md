# CYB-3417 — Security hardening: api-key scope guard + constant-time token compare

## Why

Pre-prod review of PR #282 (dev → main release) surfaced two related auth-surface weaknesses that must be closed before the release lands in production. Both live in the same PR window as CYB-3154 (dbk_ API keys) and the email-login admin path.

### A2 — `POST /admin/api-keys` accepts privileged scopes from any admin caller

`backend/internal/handlers/apikey/handler.go` binds `Scopes []string` from the request body straight into `models.APIKey.Scopes` with no validation. This means:

1. An admin JWT obtained via `POST /api/v1/auth/email-login` (24h TTL, **no proof of email ownership** — the endpoint only checks that the email's domain matches `ALLOWED_DOMAIN`) can call `POST /admin/api-keys` with `scopes:["*"]` or `scopes:["apikeys:manage"]`.
2. The resulting API key outlives the JWT session and, in the case of `apikeys:manage`, can mint further keys with any scope — a self-perpetuating admin credential that survives revocation of the original session.
3. Admin emails are effectively public (git commit `author.email`, LinkedIn, Slack), so the attack surface is not narrow.

### A3 — Non-constant-time comparison on legacy token paths

`backend/internal/middleware/auth.go` uses plain `!=` string comparisons at four sites when checking `DATABREW_TOKEN` / `ADMIN_TOKEN` / SDK static-token — leaking byte-boundary timing information about the configured secret. `authz.go` already uses `crypto/subtle.ConstantTimeCompare`, and `ArgoWebhookAuth` in the same file uses the safe pattern inline; the four legacy sites drifted from that discipline.

## What Changes

**A2 — Privileged-scope allowlist gated by AuthMethod (`internal/handlers/apikey/handler.go`)**

- New unexported set `PrivilegedScopes = {"*", "apikeys:manage"}`.
- On `POST /admin/api-keys`, iterate `req.Scopes`; reject the entire request with **403 FORBIDDEN** if any scope is privileged **and** the authenticated principal's `AuthMethod` is not `AuthMethodStaticToken`. Static admin token requires proof-of-possession of the shared secret and remains the intended bootstrap path for provisioning wildcard-scope keys.
- Also reject whitespace-only / empty scope entries with **400 INVALID_ARGUMENT** before hitting the repository.

**A3 — Constant-time `tokenEquals` helper (`internal/middleware/auth.go`)**

- New unexported `tokenEquals(got, want string) bool` that:
  - Fails closed on empty `want` (unconfigured secret must never authenticate anyone).
  - Strips an optional `Bearer ` prefix from `got` for RFC 6750 compatibility (the four call sites already accepted `Bearer <token>` shape).
  - Length-checks first, since `subtle.ConstantTimeCompare` returns `0` on unequal lengths.
  - Uses `subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1`.
- Replaces the four plain-`!=` sites in `StaticTokenAuth`, `JWTAuth` (both X-Databrew-Token path and JWT-fallback-to-static path), and `AdminTokenAuth`.
- `ArgoWebhookAuth` already used the safe pattern inline; left as-is (minimal-blast-radius edit).

**A1 — email-login security comment only (no behavior change)**

- The admin-elevation branch in `routes/routes.go` `POST /api/v1/auth/email-login` is retained per user requirement (internal-use convenience, not a proof-of-ownership path in this iteration).
- A block comment is added explicitly framing the resulting admin JWT as a **time-limited operator session** that MUST NOT be usable to mint long-lived credentials — the invariant enforced by the A2 change.

**Out of scope (deferred / documented)**

- `tag_registry.sql` idempotency (missing `IF NOT EXISTS`): finding is real but AI-RULES `already-applied migrations are immutable` wins. Left as-is; documented in `decisions.md`.
- Email-login rate limiting + audit log (agent finding A4/A5): follow-up ticket, not in this hotfix.
- `last_used_at` unbounded goroutine (agent finding A6): follow-up ticket.
- Frontend `ApiKeysPage` (already added under CYB-3154): no change here.

## Impact

**Runtime**
- `POST /api/v1/admin/api-keys` gains a **403 FORBIDDEN** response for JWT-authenticated callers requesting `*` or `apikeys:manage` scope. Existing wildcard-scope keys already issued from JWT sessions remain valid but cannot be renewed via that path — the operational workflow shifts to using the static admin token for privileged key issuance.
- Timing-side-channel surface on `DATABREW_TOKEN` / `ADMIN_TOKEN` / SDK static-token is closed. Behavior for correct tokens is unchanged; misconfigured empty tokens now fail closed (previously would accept an empty request).

**Off-limits zones touched (explicit approval)**
- `backend/internal/middleware/auth.go` — added helper + 4 non-behavior-changing call-site refactors.
- `backend/internal/handlers/apikey/handler.go` — new validation branch.
- Requires ≥2 reviewers per `AI-RULES.md#off-limits-zones`.

**API contract**
- No new endpoints; no OpenAPI shape change (only a new 403 branch on an existing endpoint). `docs/review/api-guide.md` for `/admin/api-keys` gains one line about the AuthMethod restriction.

**No migrations, no schema, no `outbox/`, no `Frontend/`.**

## Rollback

- `git revert` the merge commit — restores plain-`!=` compare and unrestricted scope issuance. No data changes.
- Existing api_keys rows are untouched; keys issued under the tightened rules remain valid.

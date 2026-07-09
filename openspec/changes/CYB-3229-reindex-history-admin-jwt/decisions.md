# CYB-3229 Decisions

## 2026-07-09 — Scope: read-only only (user-confirmed)
- **Context**: `adminAuth` guards hard-delete + destructive reindex ops, not just
  read views. Naively accepting admin-role JWT on the whole group would expose
  hard delete to every ADMIN_EMAILS web user.
- **Decision**: new `AdminTokenOrAdminRole` middleware applied ONLY to read-only
  admin search GETs; destructive routes keep `AdminTokenAuth`.
- **Alternatives**: (a) widen AdminTokenAuth globally — rejected (privilege
  escalation to hard delete); (b) replace adminAuth with RequireScope on read
  routes — rejected (would drop static-AdminToken support and change token
  semantics).

## 2026-07-09 — No regression for token callers
- `AdminTokenOrAdminRole` composes over `AdminTokenAuth`: admin-role principal →
  allow; else delegate to the unchanged token check. DatabrewToken/JWT-admin keep
  working; a static-`AdminToken`-only caller already couldn't pass the parent `api`
  group's `Authenticate` today, so there is no behavior regression.

## 2026-07-09 — Off-limits acknowledgement
- Adds a function to `backend/internal/middleware/auth.go` (off-limits). Additive,
  no edit to existing `AdminTokenAuth`. PR flags second-reviewer requirement per
  AI-RULES.

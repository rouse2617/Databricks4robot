# CYB-3229 Read-only reindex history reachable by admin-role JWT

## Problem

CYB-3225 stopped the Settings page from logging out on the `GET /admin/search/reindex-jobs`
401, but web admins still can't see reindex history: the browser session (JWT)
carries no static admin token, and the `admin` group is guarded by
`AdminTokenAuth` (static token only).

## Constraint (security)

`AdminTokenAuth` (the `adminAuth` middleware) also guards **destructive**
endpoints:
- `DELETE /internal/assets/:id`, `POST /internal/assets:batch_delete` (hard delete)
- `POST /internal/commit-segments`
- `POST /admin/search/reindex`, `.../reindex-jobs`, `.../:id/stop|resume|abandon`

Widening `AdminTokenAuth` globally to accept admin-role JWT would let any
`ADMIN_EMAILS` web user hard-delete assets. **Not acceptable.** User confirmed the
scope: **read-only reindex history only.**

## Scope

- New middleware `AdminTokenOrAdminRole(adminToken, databrewToken, env)`: allow if
  the `Authenticate`-resolved principal has `Role == "admin"`, else fall back to
  the existing `AdminTokenAuth`.
- Apply it **only** to the read-only admin search views:
  `GET /admin/search/reindex-jobs`, `GET /admin/search/reindex-jobs/:id`,
  `GET /admin/search/outbox-stats`, `GET /admin/search/audit`.
- All destructive `admin`/`internal` routes keep `adminAuth` unchanged.

## Out of Scope

- No change to destructive endpoints' auth.
- No change to `AdminTokenAuth` itself (new middleware composes over it).
- Frontend already tolerates these endpoints (CYB-3225 skipAuthRedirect); no
  frontend change required — the Settings history simply starts returning 200 for
  admin sessions.

## Off-limits / review

Touches `backend/internal/middleware/auth.go` (off-limits auth) — **requires a
second reviewer**. Change is additive (new function); existing token behavior is
unchanged.

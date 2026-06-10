# CYB-1215: rename X-Grace-Token → X-Databrew-Token across the stack

## Motivation

The auth header currently uses the legacy name `X-Grace-Token` (inherited from the deprecated Grace project). As the platform has been fully renamed to **DataBrew**, all references should be updated to `X-Databrew-Token` for consistency and to avoid confusion with the now-unrelated Grace system.

## Scope

Rename **one HTTP header string** `X-Grace-Token` → `X-Databrew-Token` in every layer that emits or consumes it. The rename is purely cosmetic/consistency — no behavioral or security change.

### Rename ALL related identifiers (for internal consistency)

| Current | New |
|---------|-----|
| `X-Grace-Token` (HTTP header) | `X-Databrew-Token` |
| `GRACE_TOKEN` (env var) | `DATABREW_TOKEN` |
| `GraceToken` (Go struct field / function name) | `DatabrewToken` |
| `extractGraceToken` (Go function) | `extractDatabrewToken` |
| `grace_token` (query param in mcap-preview) | `databrew_token` |
| `grace_session` (cookie name) | `databrew_session` |

### Out of scope

- `X-Admin-Token` header (unrelated)
- JWT/OIDC auth (Phase 2, future)
- `GraceToken` Swagger security scheme name → rename to `DatabrewToken` (it is the same scheme, just renamed)

## Risk

- **Off-limits zone touched**: `backend/internal/middleware/auth.go` — requires explicit approval
- **Service-to-service**: mcap-preview service reads the header and forwards it to backend — both sides must be deployed together
- **Cookie rename**: `grace_session` → `databrew_session` means existing browser sessions will break (users need to re-login). Acceptable in dev.
- **Env var rename**: `GRACE_TOKEN` → `DATABREW_TOKEN` — Cloud Run env vars and K8s secrets must be updated. Coordinated deploy.

## Strategy

Single PR, **apply code changes first, deploy + update env vars second**. This avoids a window where deployed code references the old env var but the new header name is active.

The PR will:
1. Rename all code-level references (Go struct fields, Python vars, function names, string literals)
2. Update OpenAPI spec + swagger generated docs (re-run `swag init`)
3. Update all doc references
4. Update nginx configs, deploy yamls, CI configs
5. **Do not rename env vars in K8s Cloud Run env configs** — these are deployed separately
6. **Do not rename K8s secret keys** — these are deployed separately

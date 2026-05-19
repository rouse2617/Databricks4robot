## 2026-05-19 — Keep API status as derived field

- **Context**: Reviewer asked to remove `assets.status` from DB; frontend still filters on `status`.
- **Decision**: Drop DB column; derive `Asset.status` via `SyncLegacyFields()` from `lifecycle_state`; map filter `status` to lifecycle SQL with legacy value sets.
- **Alternatives**: Break API and force clients to use `lifecycle_state` only.
- **Rationale**: Minimizes frontend churn while fixing fresh-DB failures.

## 2026-05-19 — Admin routes production gate

- **Context**: `GRACE_TOKEN` could authorize hard-delete when `ADMIN_TOKEN` empty in production.
- **Decision**: `Config.AdminRoutesEnabled()` unmounts routes; `AdminTokenAuth` returns 403 if invoked without token in production.
- **Alternatives**: Only middleware change without unmounting routes.
- **Rationale**: Defense in depth; matches reviewer deployment guard request.

# CYB-3229 Tasks

- [x] Create Linear issue CYB-3229
- [x] Create branch `fix/CYB-3229-reindex-history-admin-jwt` from origin/dev
- [x] **OpenSpec checkpoint** — user confirmed「开做」(combined checkpoint, 2026-07-09)
- [ ] Add `AdminTokenOrAdminRole` middleware in `middleware/auth.go` (off-limits)
- [ ] routes.go: split read-only admin search GETs onto the new middleware; keep destructive on `adminAuth`
- [ ] Unit test: admin-role principal → allowed; non-admin + no token → 401/403; token path unchanged
- [ ] Tier M/L: `gofmt` + `go vet` + `go build` + `go test ./internal/middleware/... ./internal/handlers/... ./cmd/server/...`
- [ ] Commit + PR (CYB-3229 + OpenSpec; flag off-limits/2nd reviewer) + merge to dev
- [ ] Deploy-verify on dev: admin JWT `GET /admin/search/reindex-jobs` → 200; POST reindex / hard delete via JWT → still 401/403
- [ ] Update Linear CYB-3229 → Done

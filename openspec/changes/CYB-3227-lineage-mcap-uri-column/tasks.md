# CYB-3227 Tasks

- [x] Create Linear issue CYB-3227
- [x] Create branch `fix/CYB-3227-lineage-mcap-uri-column` from origin/dev
- [x] **OpenSpec checkpoint** — user confirmed「开工，基于最新的dev」(2026-07-09); rebased on origin/dev 639deaa9
- [ ] Fix: `storage_uri` → `mcap_uri` in `lineage_response.go` upstream query
- [ ] Add `slog.Warn` on upstream query error (match algo/delivery/eval pattern)
- [ ] Tier S/M: `gofmt` + `go vet` + `go build` + `go test ./internal/handlers/asset/...`
- [ ] Commit + PR (body: CYB-3227 + OpenSpec change-id) + merge to dev
- [ ] Deploy-verify on dev (post-merge): `GET /assets/{id}/lineage` for an asset with an mcap → `upstream.mcap_uri` non-empty
- [ ] Update Linear CYB-3227 → Done with commit hash

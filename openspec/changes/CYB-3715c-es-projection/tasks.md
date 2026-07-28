# CYB-3715c tasks

- [x] `internal/searchindex/builder.go` — emit 7 top-level fields from Asset struct (guarded by non-empty)
- [x] `internal/elasticsearch/query_ir.go` — extend `facetFieldPath` for 4 facet-able fields
- [x] `internal/searchindex/builder_test.go` — populated / empty tests
- [x] `go test ./internal/searchindex/... ./internal/elasticsearch/...` — pass
- [x] `go test ./...` — full suite green
- [x] `go build -ldflags "-w -s" ./...` — clean
- [x] Commit + push + open PR base=dev
- [ ] Post-merge: run `POST /api/v1/admin/search/reindex` on dev to backfill ~800 existing docs
- [ ] Verify: `mode=keyword q=CyberCap2 + camera_model=CyberCap2` → total > 0

## Not in this PR

- Mapping template update (dynamic mapping handles it for now)
- Frontend chips (CYB-3715d)

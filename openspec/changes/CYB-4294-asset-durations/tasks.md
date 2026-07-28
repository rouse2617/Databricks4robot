# CYB-4294 tasks

## Backend
- [ ] `models/asset.go` — request/response types (AssetDurationsRequest, AssetDurationsResponse, DurationLookupItem, DurationStats)
- [ ] `postgres/repos.go` — `AssetRepo.LookupDurations(ids, minMs, maxMs)` returning `[]DurationRow` (asset_id, grace_video_id, duration_ms) plus errors
- [ ] `usecase/asset/usecase.go` — `LookupDurations(ctx, req)` — dedup ids, call repo, compute stats + build response mapping input→row (also emits missing_ids)
- [ ] `handlers/asset/handler.go` — `LookupDurations` gin handler with 5000-cap validation, decodes body, delegates to usecase
- [ ] `routes/routes.go` — register `POST /assets/durations` with `assets:read` scope
- [ ] Tests: repo test (real query path or mock), usecase test (dedup + stats + missing set), handler test (200 / 400 over-cap / 200 empty)

## Frontend
- [ ] `api/assetsApi.ts` — `lookupAssetDurations(req)` typed wrapper
- [ ] `pages/AssetDurationLookup.tsx` — page component (textarea + filters + submit + results)
- [ ] `router` — mount `/assets/durations` + sidebar entry
- [ ] Tests: page test (submit round-trip mocked, stats + bar + table render, CSV export filename)

## Verification
- [ ] `go test ./...`
- [ ] `go build -ldflags "-w -s" ./...`
- [ ] `npx tsc --noEmit`
- [ ] `npx vitest run <touched>`
- [ ] `npx vite build`

## Ship
- [ ] Commit, push, PR base=dev
- [ ] Merge (user auth)
- [ ] Deploy → verify with real mixed batch of 100 ids

## Not in this PR
- Batch-dispatch button (下发) from result set
- Configurable histogram buckets

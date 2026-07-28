# CYB-4303 tasks

## Backend

- [ ] `handlers/dashboard/handler.go` — new `DurationDistribution(c *gin.Context)` handler; parse optional `asset_type` query; delegate to usecase; JSON response
- [ ] `usecase/dashboard/usecase.go` — new method `DurationDistribution(ctx, assetType) (*models.DurationDistribution, error)` calling repo
- [ ] `postgres/repos.go` (or a dashboard-scoped file if convention exists) — new `AssetRepo.DurationDistribution(ctx, assetType string) (*models.DurationDistribution, error)`. Single-query with UNION or two-round; guard `duration_ms IS NOT NULL AND duration_ms >= 0`
- [ ] `models/dashboard.go` (or wherever dashboard models live) — `DurationDistribution` struct + `DurationBucket` struct matching the JSON shape in proposal.md
- [ ] `routes/routes.go` — mount `GET /dashboard/duration-distribution` under the same scope as existing dashboard endpoints (`dashboard:read` or the equivalent)
- [ ] Tests:
  - repo test using existing mock/fake DB pattern → verify SQL parameters + result mapping
  - usecase test → covers all-assets path + asset_type-filtered path
  - handler test → 200 with data, 400 for invalid asset_type, correct JSON shape

## Frontend

- [ ] `api/dashboard.ts` — new `dashboardApi.durationDistribution(assetType?)` returning typed response
- [ ] `components/dashboard/DurationDistributionCard.tsx` — new card component; `Segmented` control for asset_type; flexbox bar chart; stats footer
- [ ] `pages/DashboardPage.tsx` — remove the two event cards (identify them by name first), insert `DurationDistributionCard` in a natural slot
- [ ] Card + api tests: `DurationDistributionCard.test.tsx` covering render + segmented toggle re-fetches

## Verification

- [ ] `go test ./...`
- [ ] `go build -ldflags "-w -s" ./...`
- [ ] `npx tsc --noEmit` (modulo pre-existing DeployPanel error)
- [ ] `npx vitest run` on touched frontend files
- [ ] `npx vite build`

## Ship

- [ ] Commit + push + PR base=dev
- [ ] Merge (user auth)
- [ ] Deploy → dev browser walkthrough: confirm the two event cards are gone, new duration card renders with real numbers, Segmented switching refetches

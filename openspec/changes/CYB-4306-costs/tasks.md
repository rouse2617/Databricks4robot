# CYB-4306 tasks

## Backend

- [ ] `models/asset_costs.go` — request/response types (`AssetCostsRequest`, `AssetCostsResponse`, `AssetCostItem`, `AssetCostByAlgo`, `AssetCostStats`)
- [ ] `postgres/repos.go` — new `AssetRepo.LookupCosts(ctx, req)` returning per-(asset[,algo]) rows. Two-step: (a) resolve ids → known assets, (b) SQL over pipeline_run_nodes joined to pipeline_runs
- [ ] `usecase/asset/usecase.go` — new `LookupCosts(ctx, req)` calling repo, computing per-asset stats (matched/missing/filtered_out counters + total/mean/p50/p90 cost + total gpu_sec/cpu_sec/run_count), building `by_algo` grouping when requested
- [ ] `handlers/asset/handler.go` — new `Handler.LookupCosts` with request validation (5000-id cap, window ≤90d, end>=start, group_by enum). Reuse the id-cap error code from CYB-4294 (`IdListTooLarge`); add `InvalidTimeRange` and `WindowTooLarge` codes
- [ ] `routes/routes.go` — mount `POST /assets/costs` in the same `assets:read` scope block as `/assets/durations`
- [ ] Backend tests:
  - repo test using fakeDB/fakeRows: with an ids list + window, verify the SQL params + the row shape parses correctly
  - usecase test: dedup + resolve + stats + by_algo grouping + missing/filtered_out set
  - handler test: 200 with mock data, 400 empty ids, 400 over-cap, 400 missing dates, 400 end<start, 400 window>90d, 400 bad group_by

## Frontend

- [ ] `api/assets.ts` — extend with `lookupCosts(req)` matching new endpoint contract + types
- [ ] `pages/presets/costs.tsx` — preset config:
  - `filters.initial = { start_at: dayjs().subtract(7,'day').toISOString(), end_at: dayjs().toISOString(), group_by: "asset" }`
  - `filters.render` → DatePicker.RangePicker + Segmented group_by
  - `filters.validate` returns error string when start>=end, window>90d, or either date empty
  - `histogram(items)` → 5 cost buckets
  - `extraStats(response)` → 7 entries (total/mean/p50/p90 USD + total gpu-min + total cpu-min + total run_count)
  - `csv.filename(n) = ` `asset-costs-<yyyy-mm-dd>-<yyyy-mm-dd>-<n>.csv`
  - Columns switch when group_by="asset_algo" (adds algo_key column)
- [ ] `pages/AssetsWorkbenchPage.tsx` — add third tab `成本查询` (icon DollarOutlined), URL `?view=costs`, mounts a small wrapper page like `AssetCostsLookup.tsx` (mirror the `AssetDurationLookup` shim pattern)
- [ ] `pages/AssetCostsLookup.tsx` — 5-line shim: `<BatchAssetLookup preset={costsPreset} />` (lazy chunk boundary parity with durations)
- [ ] `App.tsx` — no route change needed (workbench handles it via ?view=)
- [ ] Frontend tests: `presets/costs.test.tsx` covers histogram bucketing + csv filename + filters.validate cases

## Verification

- [ ] `go test ./...`
- [ ] `go build -ldflags "-w -s" ./...`
- [ ] `npx tsc --noEmit` (modulo pre-existing DeployPanel error)
- [ ] `npx vitest run` on touched files
- [ ] `npx vite build`

## Ship

- [ ] Commit, push, PR base=dev
- [ ] Merge, deploy
- [ ] Dev smoke on a real 100-id sample with a 7-day window

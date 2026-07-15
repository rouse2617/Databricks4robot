# Tasks — CYB-3072

Branch: `feat/CYB-3072-grace-duration-sync` (from `origin/dev`). Verification tier: **M** (new package + background loop; no HTTP API, no migration).

## 1. Reusable Grace client (`internal/grace`)
- [ ] [backend] `grace/client.go`: `Client{baseURL,username,password,http}`, `Config`, `ConfigFromEnv()` (GRACE_API_URL/USERNAME/PASSWORD), `NewClient(cfg)`. Private `get(ctx,path,query)` sets Basic Auth + `User-Agent: curl/8` + Accept json + timeout; returns decoded JSON / typed errors.
- [ ] [backend] `FetchVideoDurations(ctx) (map[string]float64, error)`: paginate `/grace/videos` (size=1000), collect `id → duration_sec` where `duration_sec > 0`.
- [ ] [backend] unit test with an httptest server: pagination + UA header + duration extraction + auth header.

## 2. Repo write method (Scenario: re-sync idempotent)
- [ ] [backend] `VideoDurationRepo.Upsert(ctx, map[string]float64) error` — batched `INSERT ... ON CONFLICT (video_id) DO UPDATE SET duration_sec=EXCLUDED.duration_sec, updated_at=now()`.

## 3. Syncer usecase (`internal/usecase/videoduration`)
- [ ] [backend] `Syncer{client, repo}` with `SyncAll(ctx) error` = fetch + upsert; logs count. Best-effort caller wrappers swallow errors.
- [ ] [backend] `StartSyncLoop(ctx, interval)` background goroutine (Scenario: durations appear automatically; Grace unavailable does not break the product).

## 4. Config + wiring
- [ ] [backend] config: `GraceAPIURL`, `GraceUsername`, `GracePassword`, `VideoDurationSyncIntervalSec` (default 600), enable-gate (empty URL/creds ⇒ disabled).
- [ ] [backend] `cmd/server`: build `grace.Client` + `Syncer`; start loop when configured (Scenario: sync disabled when unconfigured).
- [ ] [backend] best-effort post-creation kick (A): after `CreateBackfill` materializes, async rate-limited `SyncAll` (non-blocking).

## 5. Deploy config
- [ ] [deploy] `backend-dev.sh`: add `GRACE_API_URL`, `GRACE_USERNAME` env + `GRACE_PASSWORD` from Secret Manager `grace-api-dev:AUTH_PASSWORD:latest` (mirror grace-sync's `SECRET_ENVS`). Grant Cloud Run dev SA `secretAccessor` on `grace-api-dev` if missing.

## 6. Verify (dev, after deploy)
- [ ] Deploy backend dev; confirm the sync loop starts (log line) and Grace creds resolve.
- [ ] Confirm `video_durations` row count grows / matches Grace, and spot-check a value against live Grace (Scenario: durations appear automatically; re-sync idempotent).
- [ ] Batch subtask list shows durations for a newly created batch within one cycle.
- [ ] Negative: with creds unset, loop does not start; with Grace blocked, server + batch creation unaffected.

## 7. API contract sync
- N/A — no new/changed HTTP API (internal pull). Record in decisions.md that rows 1–8 don't apply (no HTTP surface).

## 8. Verification tiers (before PR)
- [ ] Tier M: `make fmt && make vet`; `go test ./internal/grace/... ./internal/usecase/videoduration/...`.

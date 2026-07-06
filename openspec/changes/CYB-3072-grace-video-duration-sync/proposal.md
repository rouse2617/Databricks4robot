# Proposal — CYB-3072

## Why

`video_durations` (CYB-3059) is currently populated by a one-off manual backfill. DataBrew should keep it up to date automatically. Rather than have an external system push, DataBrew actively pulls durations from Grace, so durations appear for batches created via any path (UI, API, grace-sync).

## What Changes

### New Capabilities
- **runtime-os**: DataBrew has a reusable Grace API client and automatically keeps `video_durations` in sync with Grace, so the batch subtask list's "video duration" column stays populated without manual intervention.

## Impact
- **Affected code**: new `backend/internal/grace` (reusable client), `backend/internal/postgres/video_duration_repo.go` (add Upsert), new `backend/internal/usecase/videoduration` (syncer), `backend/cmd/server` (wire + background loop), `backend/internal/config` (Grace creds), `deploy/cloudrun/backend-dev.sh` (GRACE_* env + secret).
- **New APIs**: none (internal pull; no new HTTP endpoint).
- **Dependencies**: none new (stdlib HTTP). Reuses the `grace-api-dev` Secret Manager secret.

## Scope
- **In scope**: reusable Grace client (Basic Auth + Cloudflare-safe UA), `VideoDurationRepo.Upsert`, a syncer that fetches Grace video durations and upserts, background reconcile loop (B) + best-effort kick after batch creation (A), config + deploy wiring for Grace creds.
- **Out of scope**: the grace-sync push path (superseded); any new HTTP API; prod creds/rollout; changing the read path (no live Grace call per request).

## Success Criteria
- [ ] After a batch is created (any source), its videos' durations appear in `video_durations` automatically within one sync cycle, without manual SQL.
- [ ] The Grace client is generic enough that a future Grace data need adds a method, not a new client.
- [ ] Grace being slow/down never blocks batch creation or the subtask list read path (best-effort, background).
- [ ] No duplicate rows; re-sync is idempotent (upsert).

## Goals (SLO)
- **Freshness**: a video's duration is stored within ≤ 1 background sync interval (default 10m) of the video being dispatchable.
- **Cost**: steady-state Grace load is a few paginated requests per sync cycle (not per-asset).

# Design — CYB-3072

## Architecture Context
- **Constraints**: backend is Go on Cloud Run (autoscaling, may scale to zero); Grace is external, behind Cloudflare Pages (rejects default Go/urllib UA — needs a browser-like User-Agent), Basic Auth. Grace `/grace/videos` exposes `duration_sec`; there is no `id:in:` batch filter (returns 500), but full pagination is cheap (~4.5k videos, size=1000 → ~5 requests). `video_durations` table already exists (migration 063).
- **Goals**: keep `video_durations` populated automatically for all batch sources; a reusable Grace client for future Grace data; never couple this to the read path or block batch creation.
- **Non-Goals**: a new HTTP API; grace-sync push; per-asset live lookups; prod rollout.

## Affected Modules
- `backend/internal/grace` (NEW) — reusable Grace API client.
- `backend/internal/postgres/video_duration_repo.go` — add `Upsert`.
- `backend/internal/usecase/videoduration` (NEW) — syncer (Grace client + repo).
- `backend/cmd/server/{infra,core}.go` — construct client + syncer, start background loop, wire post-creation kick.
- `backend/internal/config/config.go` — `GRACE_API_URL`, `GRACE_USERNAME`, `GRACE_PASSWORD`, sync interval/enabled.
- `deploy/cloudrun/backend-dev.sh` — GRACE_* env + `GRACE_PASSWORD` from `grace-api-dev:AUTH_PASSWORD` Secret Manager.

## Architecture Decisions

### Decision 1: A reusable, standalone `internal/grace` client (decoupled)
- **Approach**: `grace.Client{ baseURL, username, password, httpClient }` with `ConfigFromEnv()` + `NewClient(cfg)`. A private `get(ctx, path, query)` sets Basic Auth + `User-Agent: curl/8` (Cloudflare bypass) + JSON accept, with timeout. First typed method: `FetchVideoDurations(ctx) (map[string]float64, error)` — paginates `/grace/videos` and returns `id → duration_sec` (skips null/≤0). Future Grace data = add a method; callers depend on `grace.Client`, not on duration specifics.
- **Alternative**: inline Grace calls in the video-duration usecase — rejected: not reusable, couples Grace HTTP details to the feature. Also considered reusing grace-sync's code — rejected: grace-sync is a separate Cloud Run Job binary, not importable cleanly; backend needs its own client.
- **Rationale**: user explicitly wants decoupling for future Grace calls.
- **Rollback**: unused client is inert; feature gated by config (empty Grace URL/creds ⇒ syncer disabled).

### Decision 2: Backend PULLS (not grace-sync push)
- **Approach**: DataBrew fetches from Grace and upserts. Covers batches created via any path.
- **Alternative**: grace-sync POSTs to a new DataBrew endpoint — rejected by user: only covers grace-sync-created batches and adds an HTTP contract; DataBrew-pull is source-agnostic.
- **Trade-off**: backend now needs Grace creds + egress to Grace (Cloudflare). Accepted; creds via the existing `grace-api-dev` secret.

### Decision 3: Sync = periodic full fetch + upsert; triggers = background loop (B) + post-creation kick (A)
- **Approach**: `Syncer.SyncAll(ctx)` = `client.FetchVideoDurations` → `repo.Upsert(map)`. (B) a background goroutine runs `SyncAll` every `VIDEO_DURATION_SYNC_INTERVAL` (default 10m). (A) after `CreateBackfill` materializes items, a best-effort, rate-limited async kick calls `SyncAll` for immediacy. Both call the same cheap paginated fetch; no per-asset requests.
- **Alternative**: per-asset `/grace/videos?filter=id:eq:` at creation — rejected: up to N requests per batch (no `id:in:`); full pagination (~5 req) is cheaper and covers everything. On-read live fetch — rejected: external dep on a hot path.
- **Rationale**: paginated full fetch is cheap and source-agnostic; loop self-heals gaps; kick gives immediacy.
- **Risk**: Grace down → `SyncAll` errors are logged and swallowed (best-effort); next cycle retries. Never blocks creation or reads.
- **Rollback**: config flag disables the loop; empty creds disable the whole feature.

## Data Flow
```
(B) every VIDEO_DURATION_SYNC_INTERVAL (default 10m):
      Syncer.SyncAll → grace.Client.FetchVideoDurations (paginate /grace/videos)
                     → VideoDurationRepo.Upsert(map[video_id]duration_sec)
(A) CreateBackfill materialized → async best-effort Syncer.SyncAll (rate-limited)
read path (batch subtask list) unchanged: LEFT JOIN video_durations (CYB-3059)
```

## Data Model Changes
- **None.** `video_durations` already exists (migration 063). Add repo `Upsert` only.

## Risks / Trade-offs
| 风险 | 影响 | 缓解 |
|------|------|------|
| Grace 慢/宕 | 同步失败 | best-effort，吞错+下轮重试；绝不阻塞建批量/读列表 |
| 后端需 Grace 凭据 | 耦合 + 密钥面扩大 | 复用现有 `grace-api-dev` secret；空凭据=功能关闭 |
| Cloudflare 拦 UA | 请求 403 | 客户端固定 `User-Agent: curl/8`（已验证可过） |
| 全量分页拉取变大 | Grace 负载 | 仅几页（size=1000）；低频（10m）；可调间隔 |
| 多实例重复同步 | 冗余但幂等 | upsert 幂等；可接受（低频、量小） |

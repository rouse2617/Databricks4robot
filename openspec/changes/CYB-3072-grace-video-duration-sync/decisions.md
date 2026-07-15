# Decisions — CYB-3072

## 2026-07-06 — Consolidate into one `internal/grace` package (no separate usecase pkg)
- **Context**: Initial layout put the syncer in `internal/usecase/videoduration`; user pushed back on file/package sprawl for a small feature.
- **Decision**: Keep a single `internal/grace` package: `client.go` (reusable Grace client) + `sync.go` (`Syncer` using a locally-defined `durationRepo` interface, so the package does not import postgres). No `usecase/videoduration` package.
- **Rationale**: Fewer files; the reusable client (which the user explicitly wanted for future Grace data) stays the one home for Grace concerns. The syncer defines its own tiny repo interface so layering stays clean.

## 2026-07-06 — Triggers: background loop (B) only; skip per-batch kick (A)
- **Context**: Proposed C (loop + post-creation kick). Implemented B only.
- **Decision**: A background `StartSyncLoop` runs `SyncAll` once at startup then every `VIDEO_DURATION_SYNC_INTERVAL_SEC` (default 600s). No hook into `CreateBackfill` (A) — avoids cross-package plumbing.
- **Rationale**: The loop is source-agnostic and self-healing; new batches get durations within ≤1 cycle. Durations are static (video transcoded before dispatch) and shown in a list users view over minutes, so ≤10min freshness is fine. A can be added later (a `func()` kick) if immediacy is needed.

## 2026-07-06 — Grace creds from Secret Manager; whole-secret JSON mounted
- **Context**: Grace password lives in Secret Manager `grace-api-dev` (JSON `{AUTH_USERNAME, AUTH_PASSWORD}`), not GKE.
- **Decision**: `backend-dev.sh` mounts the whole secret as `GRACE_PASSWORD` (`--set-secrets GRACE_PASSWORD=grace-api-dev:latest`); the client's `parsePassword` extracts `AUTH_PASSWORD` from the JSON (or accepts a plain password). URL + username are non-sensitive env overrides. The Cloud Run dev SA already had `secretAccessor` on `grace-api-dev` (verified/ensured).
- **Rationale**: Robust regardless of grace-sync's exact `--set-secrets` syntax; no key-extraction dependency in gcloud.

## 2026-07-06 — API contract sync: N/A
- No new/changed HTTP API (internal pull loop). AI-RULES API-contract-sync rows 1–8 do not apply. No migration (video_durations exists, migration 063).

## 2026-07-06 — Best-effort, never on hot paths
- Sync errors are logged and swallowed; a Grace outage cannot crash the server or block batch creation / the subtask list read path. Empty `GRACE_API_URL`/creds disables the loop entirely.

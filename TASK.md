# Task: Fix Backend MAJOR Bugs

Fix these issues found by code review in the cyber-databrew Go backend.

## Issues to Fix

### 1. Pipeline test nil pointer panic
- **File:** `backend/internal/handlers/pipeline/handler_test.go` 
- **Problem:** Some pipeline test causes nil pointer dereference panic
- **Fix.** Find the nil pointer cause and fix it. Run `go test ./internal/handlers/pipeline/` to verify.

### 2. CSRF vulnerability
- **Files:** `backend/routes/routes.go` + `Frontend/src/api/pipelineClient.ts`
- **Problem:** No CSRF protection on POST routes; cookie auth (`credentials: "include"`) lacks SameSite setting
- **Fix (backend):** Add CSRF middleware or ensure SameSite=Strict/Lax on cookie settings. A light approach: require `X-Requested-With: XMLHttpRequest` header on mutating API endpoints, verified via middleware.
- **Fix (frontend):** Add `X-Requested-With: XMLHttpRequest` header to all mutating API calls in `pipelineClient.ts`

### 3. Migration scripts not idempotent
- **Files:** `backend/migrations/` + `backend/scripts/apply_pg_deltas.sh`
- **Problem:** `apply_pg_deltas.sh` re-applies all migrations without checking if they already ran. Will fail on existing DB.
- **Fix:** Add `IF NOT EXISTS` to migration SQL statements, or make the script check if tables exist before running.

## Tech Stack
- Go 1.25 + Gin framework
- PostgreSQL
- Project root: /tmp/worktree-fix-backend

## Steps
1. Read the failing test and fix the nil pointer
2. Add CSRF protection
3. Fix migration idempotency
4. Run `go build ./cmd/server && go test ./...` to verify
5. Commit: `git add -A && git commit -m "fix: resolve backend MAJOR bugs - nil pointer, CSRF, migration idempotency"`

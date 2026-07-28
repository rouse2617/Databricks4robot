# Tasks — CYB-3713 keyword mode dispatch

## Backend

- [ ] Add `Q string \`json:"q,omitempty"\`` field to `queryir.QueryRequest` (`backend/internal/queryir/types.go`)
- [ ] In `queryir.Normalize` (`compile.go`), trim `Q`; when `Mode ∈ {keyword, semantic, similar}` and `Q != ""` inject a `_fulltext ilike Q` predicate into `Where` (AND-wrap if `Where` already exists)
- [ ] Introduce `isFulltextMode(mode string) bool` helper in `queryir` for the three-mode set (reused by tests + doc)
- [ ] Confirm no change needed in `queryplan.shouldUseESRecall` — injected predicate already triggers `hasFulltextPredicate` path
- [ ] Emit `warnings: ["keyword search degraded to postgres ilike (elasticsearch unavailable)"]` in the handler when ES fallback fires on a keyword request

## Tests

- [ ] `queryir/compile_test.go` — Normalize injects `_fulltext ilike foo` for `mode=keyword q=foo` (empty where)
- [ ] `queryir/compile_test.go` — Normalize AND-wraps existing where for `mode=keyword q=foo where=(owner=x)`
- [ ] `queryir/compile_test.go` — Normalize does **not** inject for `mode=structured q=foo` (structured mode does not participate)
- [ ] `queryir/compile_test.go` — Normalize does not inject when `q=""` regardless of mode
- [ ] `queryplan/planner_test.go` — Plan with keyword+q sets `UseESRecall=true`
- [ ] `handlers/query/handler_test.go` — Handler end-to-end test asserting the compiled ES body for `mode=keyword q=xyz`

## Verification (Tier L required per AI-RULES.md#verification-tiers because it touches shared query types + `App.tsx`-adjacent behavior)

- [ ] `cd backend && go build -ldflags "-w -s" -o /dev/null ./...`
- [ ] `cd backend && go test ./...`
- [ ] Post-deploy dev curl set (all against `cyber-databrew-dev.cyberorigin.ai/api/v1/queries/run`):
  - `mode=keyword q=备餐操作` → hits should be > 0 AND < 242 (proves filter is active)
  - `mode=keyword q=xxxxxxxxxxxx_nonexistent` → total = 0 (proves filter is active)
  - `mode=keyword q=""` → total = 242 (empty q keeps current no-filter behavior)
  - `mode=structured no-filter` → total = 242 (control, unchanged)
  - Inspect `debug_plan.steps[0].engine == "elasticsearch"` on the first three
- [ ] Add the same 4 assertions to `scripts/api-guide-smoke.sh` for regression protection

## Contract sync (mandatory per AI-RULES §API contract sync)

- [ ] `api/openapi.yaml` — add optional `q` field to `QueryRequest` schema, note it is required for `mode=keyword|semantic|similar`
- [ ] `docs/review/api-guide.md` — add a short section under queries covering keyword mode with a `curl` example
- [ ] `scripts/api-guide-smoke.sh` — inserts the 4 assertions above

## Off-limits declaration

- [ ] Confirm the diff does **not** touch `middleware/auth*`, `outbox/`, `migrations/`, `.env*`, `schemas/pg-phase0.sql` (grep before commit)
- [ ] `git status` clean of unrelated changes before staging

## Deploy

- [ ] Merge PR into `dev` (dev branch has no branch protection; auto-merge OK once Tier L passes locally)
- [ ] Wait for `deploy-dev.yml` (backend-dev.sh path) to complete successfully
- [ ] Rerun the 4-assertion post-deploy verify
- [ ] Update Linear CYB-3713 to Done and attach PR link

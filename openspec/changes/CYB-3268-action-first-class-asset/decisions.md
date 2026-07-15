# Decisions — CYB-3268

Append-only decision log (per `docs/agents/AI-RULES.md` § Decision log).

## 2026-07-10 — Frontend action tab: option B (adapter in same PR)

- **Context**: Unifying `/assets/:id/actions` on the `assets` table changes the
  4-method response shape (action-table row → first-class asset row). The
  frontend action tab (`Frontend/src/components/asset-detail/ActionsTimelineTab.tsx`)
  reads old top-level fields (`action_id`, `primary_label`, `labels`, `start_ns`,
  `source_type`, …) and POSTs the old create body. Left unchanged it does not
  merely blank — create breaks (422) and list goes empty-or-crash
  (`a.labels.filter` on `undefined`).
- **Decision**: Option **B** — ship a shape adapter in the same PR, contained to
  `Frontend/src/api/actions.ts` (map new asset row ↔ existing `Action` view model
  in `list`/`create`; component untouched). User approved 2026-07-10.
- **Alternatives**: A — backend-only, known frontend regression fixed by a later
  issue (rejected: leaves a shipped feature functionally dead in production).
- **Rationale**: Adapter is ~1 file / ~40 lines; the only endpoint consumer is
  the tab (verified `PreviewPage.tsx` is a false match — it uses window/stats
  fields, not the actions API).
- **Residual (not fixed by B)**: no backfill → the new GET reads `assets`, so
  pre-3268 rows in the old `actions` table do not appear; the tab shows only
  newly-created first-class actions until a separate backfill issue. B fixes the
  shape/crash/create regression, not the data gap.

## 2026-07-10 — Old actionHandler: retire wiring, keep code (proposal wording fixed)

- **Context**: `proposal.md` prose (What Changes + Impact) said the old
  `actionHandler` group is "退役(代码删除)", but the v2 Linear comment
  (b7469708…, latest + authoritative), `design.md` Decision 1, and `tasks.md`
  all say keep the code and only cut the 4 route wirings.
- **Decision**: Follow the majority + latest — **retire the 4 route wirings,
  keep `internal/handlers/action/`, `internal/usecase/action/`,
  `internal/postgres` action code** for a later backfill issue. Fixed the two
  `proposal.md` spots to say "保留不删" to remove the contradiction.
- **Rationale**: Avoids re-implementing action-table read/write in the backfill
  issue; keeps 3268 scope minimal. Rule precedence #1 (latest explicit user
  decision) wins.

## 2026-07-10 — Compose List/Update/Delete from existing repo methods (no interface expansion)

- **Context**: `tasks.md` sketched new `AssetRepository` methods
  (`ListByParentAndType` / `UpdateWithParentCheck` / `SoftDeleteWithParentCheck`).
  Adding interface methods forces updating every mock (AI-RULES rule 6).
- **Decision**: Do **not** expand the `AssetRepository` interface. Implement the
  3 new usecase methods by composing existing methods:
  - List → `repo.ListWithFilters("parent_asset_id = $1 AND asset_type = $2", …)`
    (already filters `is_deleted=FALSE`, returns items+total).
  - Update → `repo.Get` (parent/type/exist check) → mutate metadata →
    `withMutationTx{ repo.Set + appendAssetEvent("asset_updated") }`
    (mirrors `Usecase.Update`; `prepAssetForWrite` bumps version).
  - Delete → `repo.Get` (check) → `withMutationTx{ repo.SoftDelete + appendAssetEvent("asset_deleted") }`.
- **Rationale**: Zero mock churn, reuses proven paths, smaller diff. Cross-parent
  / missing `aid` → `repo.Get` returns nil (Get already excludes soft-deleted) →
  `ErrNotFound` → 404.

## 2026-07-10 — GET envelope stays `{items, asset_id, total}` (not bare array)

- **Context**: `design.md`/`tasks.md` mentioned "返回 []Asset". The old
  `GET /assets/:id/actions` returned `{items, asset_id, total}` and the frontend
  reads `resp.items`.
- **Decision**: New `ListActions` returns the same envelope `{items, asset_id, total}`
  with first-class asset rows as items. Only item field shapes change.
- **Rationale**: AI-RULES "API contract first" warns against envelope mismatch;
  keeping the envelope means the frontend adapter only maps item fields, and the
  contract is consistent with the prior endpoint.

## 2026-07-10 — lifecycle_state='ready' + ≥1ms duration already covered by shared paths

- **Context**: Decision 2 asks for `lifecycle_state='ready'` on action; §4 asks
  for a ≥1ms duration floor in the usecase.
- **Decision**: `prepAssetForWrite` already defaults empty `lifecycle_state` to
  `"ready"`, and `validateAssetTimeRange` (called by `CreateChildAsset`) already
  rejects `<1ms` with `ErrDurationTooSmall` (→ 422 `INVALID_STATE`). We still set
  `a.LifecycleState = "ready"` explicitly for `asset_type=="action"` in
  `CreateChildAsset` to make the contract intentional (not reliant on a shared
  default). No new duration code needed.
- **Note for smoke/tests**: duration `<1ms` returns **422** (`INVALID_STATE`),
  not 400.

## 2026-07-10 — SDK deferred (out of scope per user)

- **Context**: User instructed "sdk 的你不用管，这部分回滚吧" — do not touch the SDK.
- **Decision**: Reverted the `ActionManager` docstring + `test_managers.py` edits;
  `sdk/` is untouched. Per AI-RULES § API contract sync "Out of scope
  declaration", the SDK rows (3, 4) are deferred; the required minimum rows are
  still met: **1** `api/openapi.yaml`, **2** `docs/review/api-guide.md`, **5**
  `scripts/smoke-asset-actions-dev.sh`, **7** `specs/asset-management/spec.md`.
- **Rationale**: The `ActionManager` is a generic dict passthrough (no shape
  coupling; same endpoints), so deferring it carries no SDK-user break beyond the
  documented API contract change. Record in Linear + PR body.

## 2026-07-10 — No repo CHANGELOG; BREAKING documented elsewhere

- **Context**: `tasks.md` asked for a `CHANGELOG.md` BREAKING entry, but the repo
  has no root `CHANGELOG.md` / changelog convention.
- **Decision**: Document BREAKING in the api-guide §2.7 v2 banner +
  `docs/agents/knowledge/action-first-class.md` + PR body instead of inventing a
  CHANGELOG file.

## 2026-07-10 — Frontend gate is `vite build`, not `tsc -b`

- **Context**: `npm run typecheck` (`tsc -b`) is red on dev HEAD in unrelated
  files (`RangeSlider.tsx`, `StateBand.tsx`, `BatchJobDetailPage.tsx`) — confirmed
  `git diff --quiet HEAD` clean on those, i.e. pre-existing, not from this change.
- **Decision**: Use `npm run build` (Vite, the deploy artifact + AI-RULES Tier-L
  frontend gate) as the verification gate; it passes clean with the adapter.
  `biome check` passes on `actions.ts`. Not fixing unrelated pre-existing type
  errors (out of scope; would touch unrelated files).

## 2026-07-10 — Pre-existing backend test failures (not CYB-3268)

- `internal/handlers/pipeline`: `TestListRuns_ReturnsTotalEstimatedCost` +
  `TestGetRun_ReturnsTotalEstimatedCost` fail (`totalEstimatedCost=<nil>`).
  Proven pre-existing: `git stash push -- backend/` (my changes removed) → both
  still fail at dev HEAD `a20523d3`. Unrelated package (no dependency on my
  changed code); not fixing here. My touched packages
  (`usecase/asset`, `searchindex`, `handlers/asset`) pass; `go build`/`go vet`/
  gofmt clean; `vite build` clean.

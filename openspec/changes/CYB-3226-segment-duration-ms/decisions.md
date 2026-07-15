# CYB-3226 Decisions

## 2026-07-09 — Warn-only parent-range check scope
- **Context**: The asset `Usecase` has no mcap-file repository, and
  `AssetRepository` exposes no mcap accessor. Segment create paths (`Create`,
  `CommitSegments`) do not load the parent mcap file, so a segment-vs-mcap range
  warning would require wiring a new `McapFileRepository` dependency (constructor
  + server.go + every Usecase test mock) purely for a warn-only log.
- **Decision**: Implement the warn-only range check where the parent is already
  loaded for free — `CreateChildAsset` (child `[start,end]` vs parent segment
  `[start,end]`, with tolerance). Defer the **segment-vs-raw_mcap** warn to the
  follow-up issue that will also design hard enforcement.
- **Alternatives**: (a) wire McapFileRepository now — rejected as disproportionate
  for warn-only; (b) drop warn-only entirely — rejected, user asked for it now.
- **Rationale**: child-vs-parent-segment is the frame-consistent, zero-cost
  comparison (both file-relative). segment-vs-mcap has the absolute-vs-relative
  reference-frame problem (see design.md) and belongs with the follow-up design.

## 2026-07-09 — Deploy-verify happens immediately post-merge on dev
- **Context**: `deploy-dev` only triggers on push to `dev` (no branch/`issue_comment`
  trigger exists), and local `backend-dev.sh` would push unmerged branch code onto
  the shared dev Cloud Run. Deploy-before-commit's "deploy the branch first" is not
  mechanically supported for backend here.
- **Decision**: merge to `dev` → `deploy-dev` auto-deploys → verify immediately on
  dev (valid segment `duration_ms>0`; `end<=start` and sub-1ms → 422). Fix forward
  if verification fails.
- **Rationale**: only executable path for shared-dev backend deploy; change is
  low-risk pure logic with full unit coverage (postgres + usecase + handler pass;
  only the pre-existing CYB-3155 pipeline cost tests remain red).

## 2026-07-09 — Duration floor = 1ms
- **Context**: user chose "reject sub-1ms" for the zero/negative-duration gate.
- **Decision**: reject when `(end - start) < 1_000_000 ns`; this is exactly the
  span that rounds to `duration_ms = 0`.

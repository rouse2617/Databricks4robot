# Decisions — CYB-3001

## 2026-06-20 — Continue past OpenSpec checkpoint for autonomous iteration
- **Context**: Repository workflow normally requires stopping after OpenSpec artifacts and waiting for explicit confirmation. The current requirement document explicitly instructs the agent not to stop at planning and to keep iterating code autonomously unless blocked.
- **Decision**: Create OpenSpec artifacts first, record this precedence decision, and continue implementing the first vertical slice without a separate checkpoint prompt.
- **Alternatives**: Stop after OpenSpec and wait for confirmation.
- **Rationale**: The current task explicitly asks for autonomous implementation progress; the first slice is additive and compatibility-preserving.

## 2026-06-20 — Temporary CYB id due missing Linear credentials
- **Context**: `LINEAR_API_KEY` is not present in the environment, so the agent cannot create or verify a real Linear issue.
- **Decision**: Use temporary trace id `CYB-3001` for the OpenSpec change and branch naming until a real Linear issue is provided or credentials are available.
- **Alternatives**: Stop all implementation until Linear is available.
- **Rationale**: Missing Linear credentials should be surfaced, but the user requested continued engineering progress and the code can be developed and tested locally.

## 2026-06-20 — Frontend full-suite blockers are pre-existing
- **Context**: `npm run lint` reports existing Biome issues in untouched CSS and unrelated frontend files. `npm run test -- --run` fails in existing Assets/Events/Pipeline tests whose assertions expect older English labels or placeholders while the rendered UI is already localized/changed.
- **Decision**: Keep unrelated frontend lint/test cleanup out of this Run API slice; verify touched frontend files with Biome, targeted Vitest, and production build.
- **Alternatives**: Fix all existing frontend lint/test debt in this PR.
- **Rationale**: Broad UI test/lint cleanup would materially expand the scope and risk of the Run API Phase 1 change.

## 2026-06-20 — Compact recovery current-work file missing
- **Context**: After context compaction, repository rules require re-reading `.agent/context/current-work.md`; the file is absent in both the original workspace and the temporary implementation clone.
- **Decision**: Continue from the OpenSpec change directory, deployed revision records, and terminal/browser verification evidence instead of blocking on the missing scratchpad file.
- **Alternatives**: Stop until a human recreates `.agent/context/current-work.md`.
- **Rationale**: The required source of truth files under `docs/agents/` are present, and the active work state is captured in `openspec/changes/CYB-3001-run-kernel-phase1/`.

## 2026-06-20 — Public dev domain uses Worker Assets from the frontend build
- **Context**: The public dev domain was stale after Cloud Run frontend deploy because the Cloudflare Worker asset bundle was not refreshed. The repository no longer needs the old `argo-ui/dist` copy path in the Cloud Run wrapper.
- **Decision**: Deploy public dev with Wrangler using `Frontend/dist` as the Worker Assets directory and keep existing Worker vars.
- **Alternatives**: Reintroduce a generated `site/` asset directory or keep relying on Cloud Run-only verification.
- **Rationale**: `Frontend/dist` is the canonical output of the current frontend build, and direct Worker Assets deploy makes `https://cyber-databrew-dev.cyberorigin.ai` match the verified source build without committing generated assets.

## 2026-06-20 — Pre-commit terraform hooks skipped locally
- **Context**: `pre-commit run --all-files` initially failed before hook execution because the Homebrew Python 3.14 `pyexpat` module is broken on this machine. Re-running with `uvx --python 3.12 pre-commit` reached hooks, but the Terraform hooks failed because neither Terraform/OpenTofu nor `tflint` is installed locally.
- **Decision**: Run all non-Terraform pre-commit hooks with `SKIP=terraform_fmt,terraform_validate,terraform_tflint uvx --python 3.12 pre-commit run --all-files`; include the `.secrets.baseline` line-number update produced by `detect-secrets`.
- **Alternatives**: Install Terraform/OpenTofu and `tflint` during this runtime slice, or skip pre-commit entirely.
- **Rationale**: The changed files do not touch Terraform. Non-Terraform hooks passed, and the skipped hooks are infrastructure-tooling checks unrelated to this PR's runtime/API scope.

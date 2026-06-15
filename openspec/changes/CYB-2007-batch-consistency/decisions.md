# Decisions — CYB-2007

## 2026-06-15 — Dev verification image uses a unique suffix
- **Context**: `deploy-before-commit.md` tags images with the current HEAD short SHA, while this runtime change had to be deployed before commit.
- **Decision**: Build and deploy the dev-verification image as `ce5fdf3-cyb2007-r2` plus `cloudrun-dev-latest`, single-platform amd64 with `--provenance=false --sbom=false`.
- **Alternatives**: Reuse plain `:ce5fdf3` — rejected because it would make the clean `origin/dev` baseline and this working-tree verification image ambiguous.
- **Rationale**: The unique suffix keeps the deployed verification artifact traceable without clobbering the clean dev baseline tag.

## 2026-06-15 — Full all-files pre-commit has unrelated site blockers
- **Context**: `pre-commit run --all-files` failed on existing generated `site/` assets and `.secrets.baseline` false positives; the hooks also auto-edited generated `site/` files and `docs/agents/pipeline-module-tasks.md`.
- **Decision**: Restore those unrelated hook edits and run `pre-commit run --files` against the staged CYB-2007 files, which passed.
- **Alternatives**: Include generated `site/` and baseline churn in this PR — rejected as unrelated to the backend batch consistency fix.
- **Rationale**: The full hook was executed as required, but its failures are outside this change's scope; the scoped hook verifies the actual staged files.

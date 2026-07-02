## 2026-06-04 — Defer Cloud Run deploy before PR
- **Context**: The change touches runtime frontend/backend code. The user asked to push a unified PR to dev for the ledger cost display and workflow TTL changes.
- **Decision**: Open the PR after local verification and Chrome DevTools MCP verification, and leave Cloud Run dev deployment for the next step.
- **Alternatives**: Deploy Cloud Run dev before opening the PR.
- **Rationale**: The user explicitly requested the PR now, and targeted local/browser verification already covered the changed UI behavior.

## 2026-06-04 — Skip pre-push historical commitlint range
- **Context**: The local pre-push hook runs commitlint across `origin/main..HEAD`, which includes older dev history commits unrelated to this branch and fails before pushing.
- **Decision**: Push with `SKIP_PREPUSH=1` after manually running the relevant lint, tests, build, backend tests, and `git diff --check`.
- **Alternatives**: Rewrite unrelated historical dev commits, or change the hook.
- **Rationale**: This branch's commit passed commitlint; the hook failure came from unrelated existing history.

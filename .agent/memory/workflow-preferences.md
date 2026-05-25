---
name: workflow-preferences
description: AI agent workflow rules, optimizations adopted, known friction points
metadata:
  type: feedback
---

# Workflow Preferences

## Adopted optimizations

- **Trivial path** for ≤20-line single-file no-API changes: skip OpenSpec checkpoint, lightweight proposal only.
- **Deploy fast path**: pure logic/usecase/repo changes verified by `go test ./...` skip docker build; CSS/text-only skip deploy.
- **current-work.md** is agent-maintained, not human-only. Agent updates on CYB start and merge.
- **stale-changes.sh** detects merged-but-not-archived change dirs. Run at session start and before PR.

## Deploy discipline

- Always tag images with git SHA + `cloudrun-dev-latest`.
- Record revision name + image tag + URL after every deploy.
- Frontend changes → Chrome DevTools MCP is mandatory (not "user clicks around").
- Use `source scripts/dev-backend-env.sh` for canonical dev API URL — never guess.

## Commit discipline

- Conventional Commits: `feat(scope):`, `fix(scope):`, `hotfix(scope):`, `docs(scope):`
- Run `git diff --stat` before commit — verify actual changes match intent (P1 pitfall).
- API handler changes → API contract sync in same commit (before-commit.sh enforces minimum).

**Why:** These are non-obvious rules discovered through field incidents (CYB-1168 pitfalls). The deploy-tag-revision chain prevents unrecoverable deploys; the diff check catches silent edit-tool failures.

**How to apply:** Follow these automatically on every task. Don't ask permission.

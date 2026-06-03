# Decisions — CYB-1614

## 2026-06-03 — Scope from real complex workflow smoke
- **Context**: Three real Argo workflows exposed fan-in validation, output file contract, shell arg normalization, cost sync, and log formatting issues.
- **Decision**: Treat these as one high-priority pipeline guardrails bugfix because they share the same user flow: component authoring → pipeline design → run → inspect logs/cost.
- **Alternatives**: Split into five separate issues.
- **Rationale**: The issues are tightly coupled in the same workflow validation and execution-inspection path, and fixing them together enables one end-to-end smoke.

## 2026-06-03 — OpenSpec checkpoint approved
- **Context**: User asked what the change primarily covers, then said "开干".
- **Decision**: Proceed with runtime edits after OpenSpec checkpoint.
- **Alternatives**: Wait for an exact "OpenSpec OK" phrase.
- **Rationale**: The user's latest instruction explicitly approved starting implementation after the scope explanation.

## 2026-06-03 — PR before dev deploy
- **Context**: Runtime changes touch backend and frontend. The standard gate calls for Cloud Run dev deploy and browser MCP verification before commit, but the user interrupted browser smoke and explicitly asked to "先提个pr".
- **Decision**: Submit the PR with completed automated verification and leave Cloud Run/browser verification as a PR follow-up item.
- **Alternatives**: Continue deploy/browser verification before opening the PR.
- **Rationale**: Current user instruction takes priority for handoff; PR evidence will clearly state deploy/browser verification was not completed in this pass.

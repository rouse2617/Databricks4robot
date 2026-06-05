## 2026-06-05 — PR before dev deploy
- **Context**: CYB-1685 changes frontend and backend runtime code. Repository deploy-before-commit guidance normally requires Cloud Run dev deploy and user approval before commit/push.
- **Decision**: Create and push the PR first per the user's explicit request, without deploying dev in this agent turn.
- **Alternatives**: Build/push/deploy backend and frontend images before PR; defer PR until dev revision verification is complete.
- **Rationale**: The user explicitly asked to submit the PR now. Local frontend/backend tests, build, lint, and browser validation have already passed; dev deployment can be run from the PR or by the user before merge.

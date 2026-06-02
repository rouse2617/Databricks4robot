# Decisions

## 2026-06-02 — OpenSpec approved

- **Context**: Runtime frontend change requires OpenSpec checkpoint approval.
- **Decision**: User approved with "可以的，我先看看前端".
- **Alternatives**: Wait for backend summary API first.
- **Rationale**: A frontend P0 prototype can validate the UX using existing
  workflow node data before committing to backend response shape.

## 2026-06-02 — PR before dev deploy

- **Context**: The diff includes Frontend runtime changes, but the user asked to
  submit the PR for review/testing after local MCP verification.
- **Decision**: Do not deploy Cloud Run dev in this agent turn; document local
  verification and leave dev deployment to the user.
- **Alternatives**: Build/push/deploy frontend dev before committing.
- **Rationale**: The user explicitly requested PR handoff, and prior workflow in
  this thread has used user-managed deployment for frontend validation.

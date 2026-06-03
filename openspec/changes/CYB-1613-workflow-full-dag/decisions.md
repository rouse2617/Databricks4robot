# Decisions — CYB-1613

## 2026-06-03 — Defer deploy verification for direct PR
- **Context**: The change touches backend runtime code, and the standard workflow asks for dev deploy verification before commit.
- **Decision**: Open the PR after local backend verification and leave deploy verification deferred.
- **Alternatives**: Build and deploy backend dev before commit.
- **Rationale**: The user explicitly requested “直接提个pr”; local targeted and full backend tests cover the handler behavior, and no frontend code changed.

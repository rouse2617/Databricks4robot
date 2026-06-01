# Decisions — CYB-1516 Docs Contract Consistency

## 2026-06-01 — Dev deploy blocked by existing Docker process
- **Context**: The change includes backend route registration and should normally be deployed to backend dev before commit.
- **Decision**: Proceed with local Tier L verification and PR creation, while recording that backend dev deploy verification was blocked.
- **Alternatives**: Kill the pre-existing Docker build process and deploy from this worktree.
- **Rationale**: The existing Docker process belongs to another session; killing it from this side conversation risks disrupting unrelated work. Local backend full tests, targeted route tests, docs build, SDK error tests, and OpenAPI reference checks passed.

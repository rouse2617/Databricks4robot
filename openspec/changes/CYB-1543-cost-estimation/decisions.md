## 2026-06-02 — Local verification before push
- **Context**: The change touches backend runtime code and a migration, while the user asked to test locally and push the branch for their deployment flow.
- **Decision**: Verified against the local backend and database instead of performing Cloud Run dev deploy before this commit.
- **Alternatives**: Build and deploy the backend dev Cloud Run image before committing.
- **Rationale**: The user explicitly requested local testing first; local API smoke confirmed persisted node cost and aggregate run cost, and full backend tests plus pre-commit passed.

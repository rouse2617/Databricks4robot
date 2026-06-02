# Decisions — CYB-1536 Asset Validation

## 2026-06-02 — Shared validator across run-like APIs
- **Context**: Pipeline deploy validates assets but algo-runs do not.
- **Decision**: Add a shared batch validator and apply it to both pipeline deploy/run creation and algo-run creation.
- **Alternatives**: Keep pipeline-only validation or duplicate validation logic in each usecase.
- **Rationale**: Consistent error behavior and one batch query are needed before asset-driven execution becomes the default flow.

## 2026-05-23 — Off-limits approval
- **Context**: CYB-1110 implementation needs changes under `backend/internal/outbox/`, which is an off-limits zone in `docs/agents/AI-RULES.md`.
- **Decision**: Proceed with focused Pub/Sub EventSource changes after user confirmed "可以,开始弄吧" in chat.
- **Alternatives**: Avoid code changes and leave only OpenSpec artifacts, or implement a wrapper outside `backend/internal/outbox/`.
- **Rationale**: The existing Pub/Sub subscriber lives in `backend/internal/outbox/`; adding ACK/NACK test seams there is the smallest safe change.

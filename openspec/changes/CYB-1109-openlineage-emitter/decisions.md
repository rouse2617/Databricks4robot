## 2026-05-23 — OpenSpec approval and Pub/Sub subscription
- **Context**: CYB-1109 needs an optional OpenLineage emitter without changing default runtime behavior.
- **Decision**: Proceed after user confirmed "可以,开始弄吧"; wire the emitter behind `OPENLINEAGE_EMITTER_ENABLED=false` and consume from a dedicated `OPENLINEAGE_SUBSCRIPTION`.
- **Alternatives**: Reuse `OUTBOX_ES_SUBSCRIPTION` or the in-memory internal bus.
- **Rationale**: A dedicated Pub/Sub subscription avoids competing with the ES consumer and keeps disabled-mode behavior unchanged.

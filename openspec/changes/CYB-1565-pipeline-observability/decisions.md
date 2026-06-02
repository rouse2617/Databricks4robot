# Decisions — CYB-1565

## 2026-06-03 — Estimated asset-node precision
- **Context**: Current workflows do not yet emit true per-asset progress inside every Pod.
- **Decision**: Derive P0.2 asset-node rows from run assets × Argo node snapshots and label cost/status source.
- **Alternatives**: Wait for component-level callbacks before building UI.
- **Rationale**: Users need an operational matrix now; source labels prevent overclaiming precision.

## 2026-06-03 — Polling remains fallback
- **Context**: CYB-1564 watcher is polling-based.
- **Decision**: Add persisted watcher state and bounded scans in this PR; leave full Argo watch stream for a later controller-grade change.
- **Alternatives**: Replace polling with watch streams immediately.
- **Rationale**: The current Cloud Run backend can ship bounded polling safely, while watch streams require more lifecycle and backpressure design.

## 2026-06-03 — Push before dev deploy
- **Context**: The user asked to batch the remaining observability changes and PR/test together. Dev migration/deploy will be handled after branch review.
- **Decision**: Complete Tier L local verification and push the branch before applying dev migrations or deploying Cloud Run.
- **Alternatives**: Apply dev migrations and deploy before commit.
- **Rationale**: This keeps the branch available for review while avoiding an uncoordinated dev schema/runtime change during the user's test window.

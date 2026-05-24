# Design — CYB-1110

## Architecture Context
- **Constraints**: Existing outbox publisher/subscriber code already supports Pub/Sub for ES indexing. Avoid schema and migration changes.
- **Goals**: Make Pub/Sub event consumption reusable and well-tested for non-ES consumers.
- **Non-Goals**: Provisioning cloud resources or replacing the internal event bus.

## Affected Modules
- `backend/internal/outbox/pubsub_subscriber.go` — Pub/Sub receive behavior and testable seams.
- `backend/internal/outbox/bus.go` — shared subscriber/event-source contract if needed.
- `backend/cmd/server/optional.go` — consume the shared surface without duplicating transport logic.

## Architecture Decisions

### Decision 1: Build on existing outbox subscriber contracts
- **Approach**: Reuse `EventSubscriber` semantics and expose/verify Pub/Sub as a first-class event source.
- **Alternative**: Add a separate parallel event source package.
- **Rationale**: The outbox package already owns at-least-once transport semantics and avoids a new abstraction with nearly identical behavior.
- **Trade-off**: The event source remains tied to outbox payload format.
- **Rollback**: Revert to existing direct `NewPubSubSubscriber` usage.

### Decision 2: Test with an interface seam
- **Approach**: Add enough seams/fakes to test ACK/NACK behavior without connecting to GCP.
- **Alternative**: Integration tests against a Pub/Sub emulator.
- **Rationale**: Unit tests are stable in CI; emulator coverage can be added later if needed.

## Data Flow

```text
Pub/Sub subscription
  -> PubSub event source
  -> consumer handler(ctx, payload)
  -> ACK on nil error / NACK on error
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Duplicate delivery | Consumers see repeated events | Preserve at-least-once contract and require idempotent consumers |
| Unbounded concurrency | Backend pressure | Keep receive settings bounded |
| Config drift | Runtime startup failures | Fail fast with explicit missing config errors |

# Proposal — CYB-1110

## Why
The event pipeline already supports Pub/Sub for outbox delivery, but downstream consumers need a reusable Pub/Sub event source surface rather than one-off subscriber wiring.

## What Changes

### New Capabilities
- Provide a reusable Pub/Sub-backed event source for asset event consumers.
- Preserve at-least-once delivery semantics with ack on success and nack on handler error.

### Modified Capabilities
- Existing Pub/Sub subscriber behavior is documented and covered for event-source semantics.

## Impact
- **Affected code**: `backend/internal/outbox/`, `backend/internal/config/`, optional consumer wiring/tests
- **New APIs**: None
- **Dependencies**: Existing `cloud.google.com/go/pubsub`

## Scope
- **In scope**: reusable Pub/Sub event source abstraction or subscriber hardening, configuration validation, tests with fakes.
- **Out of scope**: creating GCP topics/subscriptions, changing producer payloads, touching business event schemas.

## Success Criteria
- [ ] A consumer can receive asset event payloads from Pub/Sub through a stable interface.
- [ ] Handler success ACKs messages; handler failure NACKs messages.
- [ ] Missing Pub/Sub config fails fast with a clear error.
- [ ] Tests cover success, failure, and invalid wiring behavior without a real Pub/Sub service.

## Goals (SLO)
- **Latency**: Pub/Sub delivery path should not add polling delay.
- **Concurrency**: The implementation should bound outstanding messages.
- **Quality**: Existing ES subscriber behavior remains compatible.

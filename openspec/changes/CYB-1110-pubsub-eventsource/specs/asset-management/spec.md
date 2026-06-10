## ADDED Requirements

### Requirement: Pub/Sub asset event source
The system SHALL support Pub/Sub as a reusable event source for asset event consumers.

**Priority**: P2 (Nice-to-have)
**Rationale**: Pub/Sub allows event consumers to run outside the backend process while preserving the same asset event payload contract.

#### Scenario: Message acknowledged on success
- **Given** a Pub/Sub event source receives an asset event payload
- **When** the consumer handler processes the payload successfully
- **Then** the message is acknowledged.

#### Scenario: Message retried on handler failure
- **Given** a Pub/Sub event source receives an asset event payload
- **When** the consumer handler returns an error
- **Then** the message is not acknowledged and remains eligible for retry.

#### Scenario: Missing configuration
- **Given** Pub/Sub event source configuration is incomplete
- **When** the source is constructed or started
- **Then** the system fails fast with a clear configuration error.

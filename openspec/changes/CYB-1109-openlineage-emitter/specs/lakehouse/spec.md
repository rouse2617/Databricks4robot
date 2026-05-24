## ADDED Requirements

### Requirement: Optional OpenLineage emitter
The system SHALL optionally emit supported DataBrew lineage events as OpenLineage-compatible JSON to a configured endpoint.

**Priority**: P2 (Nice-to-have)
**Rationale**: OpenLineage allows DataBrew lineage facts to be consumed by Marquez or compatible governance systems without replacing PostgreSQL as the source of truth.

#### Scenario: Emitter disabled
- **Given** OpenLineage emitter configuration is disabled
- **When** the backend starts and asset events flow
- **Then** no OpenLineage HTTP calls are attempted and existing event consumers behave unchanged.

#### Scenario: Supported event emitted
- **Given** the emitter is enabled with a valid endpoint and a supported asset event is consumed
- **When** the emitter maps the event
- **Then** it sends an OpenLineage-compatible JSON payload with stable run, job, and dataset identifiers.

#### Scenario: Unsupported event skipped
- **Given** the emitter receives an event without enough lineage context
- **When** the event is processed
- **Then** the emitter skips it without failing unrelated backend processing.

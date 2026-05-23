## ADDED Requirements

### Requirement: Cross-asset audit search
The system SHALL provide a read-only cross-asset audit search over catalog event history.

**Priority**: P1 (High)
**Rationale**: Operators need to investigate asset changes by actor, event type, run id, and time window without opening each asset individually.

#### Scenario: Search by event type
- **Given** catalog events exist across multiple assets
- **When** a client searches audit events with an `event_type` filter
- **Then** the response SHALL contain only events matching that event type, ordered newest first

#### Scenario: Search by actor
- **Given** catalog events include actor identities
- **When** a client searches audit events with an `actor` filter
- **Then** the response SHALL contain events whose actor identity matches the filter

#### Scenario: Search by run id
- **Given** catalog events were produced by an algorithm or import run
- **When** a client searches audit events with a `run_id` filter
- **Then** the response SHALL contain only events for that run id

#### Scenario: Search by time window
- **Given** catalog events exist before, inside, and after a requested time window
- **When** a client searches with `time_from` and `time_to`
- **Then** the response SHALL contain only events whose occurrence time is inside the requested window

### Requirement: Audit search pagination
The system SHALL return bounded newest-first pages and support keyset pagination.

**Priority**: P1 (High)
**Rationale**: Cross-asset event history can grow large, so the API must prevent unbounded result sets and stable pagination drift.

#### Scenario: Default bounded page
- **Given** a client omits `limit`
- **When** the client searches audit events
- **Then** the response SHALL use the default page size and SHALL NOT return an unbounded number of rows

#### Scenario: Cursor fetches older rows
- **Given** a search response returns `next_cursor`
- **When** the client repeats the search with that cursor
- **Then** the response SHALL return events older than the cursor event sequence

#### Scenario: Oversized limit is capped
- **Given** a client requests a limit larger than the supported maximum
- **When** the client searches audit events
- **Then** the system SHALL cap the effective limit instead of returning an oversized page

### Requirement: Audit search validation
The system SHALL reject malformed audit search filters with explicit client errors.

**Priority**: P1 (High)
**Rationale**: Invalid time or pagination filters should not fall through into confusing empty results or database errors.

#### Scenario: Invalid time filter
- **Given** a client provides a non-RFC3339 `time_from` or `time_to`
- **When** the client searches audit events
- **Then** the system SHALL return `400 INVALID_ARGUMENT`

#### Scenario: Invalid cursor
- **Given** a client provides a non-integer cursor
- **When** the client searches audit events
- **Then** the system SHALL return `400 INVALID_ARGUMENT`

#### Scenario: Invalid limit
- **Given** a client provides a non-positive or non-integer limit
- **When** the client searches audit events
- **Then** the system SHALL return `400 INVALID_ARGUMENT`

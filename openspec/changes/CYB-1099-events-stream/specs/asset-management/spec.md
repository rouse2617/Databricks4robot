## ADDED Requirements

### Requirement: Asset event SSE stream
The system SHALL provide a server-sent events stream for one asset's recorded event history and new events.

**Priority**: P1 (High)
**Rationale**: Asset detail clients and operators need live updates without repeatedly polling the paginated event API.

#### Scenario: Stream emits asset events
- **Given** an existing asset has recorded events with increasing event sequence numbers
- **When** a client opens the asset event stream
- **Then** the response uses `text/event-stream` and emits frames containing the event sequence, event type, and JSON event data.

#### Scenario: Missing asset is rejected before streaming
- **Given** no active asset exists for the requested asset ID
- **When** a client opens the asset event stream
- **Then** the system returns `404 ASSET_NOT_FOUND` before sending stream frames.

### Requirement: Asset event stream resume
The system SHALL resume asset event streams from the monotonic event sequence supplied in `Last-Event-ID`.

**Priority**: P1 (High)
**Rationale**: SSE clients rely on `Last-Event-ID` to recover from disconnects without replaying already processed events.

#### Scenario: Resume after the last received event
- **Given** an asset has events with sequences 10, 11, and 12
- **When** a client reconnects with `Last-Event-ID: 10`
- **Then** the stream only emits events with sequence greater than 10.

#### Scenario: Invalid resume cursor
- **Given** a client provides a non-integer `Last-Event-ID`
- **When** the client opens the asset event stream
- **Then** the system returns `400 INVALID_ARGUMENT` before opening the stream.

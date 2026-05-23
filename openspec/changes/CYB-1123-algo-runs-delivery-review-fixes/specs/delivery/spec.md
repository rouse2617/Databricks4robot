## MODIFIED Requirements

### Requirement: Draft and retry deliveries do not count as committed deliveries
- **Before**: The system SHALL allow delivery drafts and retries to reuse the same persistence helper as committed deliveries.
- **After**: The system SHALL keep draft and retry deliveries in a non-committed state until final commit, and SHALL NOT increment asset delivery counters or update last-delivered fields before the delivery is actually committed.
- **Reason**: Current implementation mutates asset delivery indexes too early and records pending work as completed delivery activity.

**Priority**: P0 (Critical)
**Rationale**: Premature delivery indexing corrupts delivery analytics and downstream operational behavior.

#### Scenario: Draft creation leaves asset delivery counters unchanged
- **Given** an asset with a known `delivery_count`
- **When** an operator creates a draft delivery that includes the asset
- **Then** the asset SHALL keep the same `delivery_count` and `last_delivered_*` values until final commit

#### Scenario: Retry creation does not count as a new committed delivery
- **Given** a cancelled or failed delivery with existing delivery items
- **When** an operator creates a retry delivery from that record
- **Then** the retry SHALL be created in a non-committed state without incrementing asset delivery indexes

### Requirement: Delivery acknowledgement semantics match the delivery state model
- **Before**: The system SHALL record acknowledgement metadata on a delivered delivery without requiring alignment between handler behavior and the declared delivery state model.
- **After**: The system SHALL make delivery acknowledgement semantics explicit and consistent across handler behavior, state transitions, and published API behavior.
- **Reason**: Current code leaves the `accepted` status unreachable or undocumented, depending on which layer is read.

**Priority**: P1 (High)
**Rationale**: Delivery lifecycle automation depends on a reachable and well-defined acknowledgement state.

#### Scenario: Acknowledgement follows the documented state transition
- **Given** a delivered delivery that is eligible for acknowledgement
- **When** an operator acknowledges that delivery
- **Then** the delivery SHALL transition according to the documented acknowledgement semantics and return the matching response fields

#### Scenario: Non-delivered delivery cannot be acknowledged
- **Given** a delivery that is still pending, cancelled, or otherwise not eligible for acknowledgement
- **When** an operator submits an acknowledgement request
- **Then** the API SHALL reject the request with a state error that matches the documented behavior

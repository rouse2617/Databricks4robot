## MODIFIED Requirements

### Requirement: Delivery C2 item counts reflect persisted unique items
- **Before**: The system SHALL add requested items to a pending delivery and may derive delivery item count from the request payload length.
- **After**: The system SHALL derive pending delivery item count from the persisted unique delivery items after insertion.
- **Reason**: Duplicate item submissions are ignored by storage, so counting request length can inflate delivery counts.

**Priority**: P1 (High)
**Rationale**: Item counts drive delivery review and commit behavior; inflated counts mislead operators and downstream analytics.

#### Scenario: Duplicate item add does not inflate count
- **Given** a pending delivery already contains an asset
- **When** an operator adds the same asset again
- **Then** the delivery item count SHALL remain equal to the number of unique persisted items

#### Scenario: New item add increases count by actual inserted unique rows
- **Given** a pending delivery contains one asset
- **When** an operator adds one new asset and one duplicate asset
- **Then** the delivery item count SHALL increase only for the new persisted asset

### Requirement: Delivery cancellation keeps asset indexes consistent
- **Before**: The system SHALL cancel a delivery and refresh affected asset delivery indexes as separate effects.
- **After**: The system SHALL commit cancellation state and affected asset delivery index refreshes atomically for deliveries that previously counted as delivered.
- **Reason**: A failure between status update and index refresh can leave asset delivery counters inconsistent with delivery state.

**Priority**: P0 (Critical)
**Rationale**: Delivery indexes are operational projections used for customer delivery history and asset readiness decisions.

#### Scenario: Delivered delivery cancel refreshes indexes atomically
- **Given** a delivered delivery has affected assets in its item list
- **When** an operator cancels that delivery and all index refreshes succeed
- **Then** the delivery SHALL become cancelled and all affected asset delivery indexes SHALL reflect the cancelled state

#### Scenario: Index refresh failure rolls back cancellation
- **Given** a delivered delivery has affected assets in its item list
- **When** an operator cancels that delivery and an affected asset index refresh fails
- **Then** the delivery SHALL NOT remain cancelled without the matching asset index refresh

#### Scenario: Pending delivery cancel does not refresh committed indexes
- **Given** a pending delivery has not counted as delivered
- **When** an operator cancels that delivery
- **Then** the system SHALL cancel it without refreshing committed delivery indexes

### Requirement: Delivery idempotency prevents duplicate commit side effects
- **Before**: The system SHALL store idempotent commit responses after creating delivery side effects.
- **After**: The system SHALL serialize or otherwise guard same-key delivery commit attempts before side effects, and SHALL reject same-key mismatched payloads without creating a new delivery.
- **Reason**: Reading idempotency state before side effects and saving it afterward leaves a race where concurrent requests can create duplicate deliveries.

**Priority**: P0 (Critical)
**Rationale**: Idempotency is a core delivery contract; violating it can duplicate customer-visible delivery records.

#### Scenario: Same key and same payload replays existing result
- **Given** a delivery commit already completed for an idempotency key
- **When** a client repeats the same request with the same key and payload
- **Then** the system SHALL return the original result without creating another delivery

#### Scenario: Same key and different payload is rejected before side effects
- **Given** a delivery commit already completed or is reserved for an idempotency key
- **When** a client sends a different payload with the same key
- **Then** the system SHALL reject the request and SHALL NOT create another delivery

#### Scenario: Concurrent same-key commits do not duplicate deliveries
- **Given** two clients concurrently submit a delivery commit with the same idempotency key
- **When** the requests reach the service at the same time
- **Then** at most one delivery side effect SHALL be created for that key

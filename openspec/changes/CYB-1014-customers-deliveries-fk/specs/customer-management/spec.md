## ADDED Requirements

### Requirement: Customer registry
The system SHALL maintain long-lived customer records referenced by deliveries.

**Priority**: P0 (Critical)
**Rationale**: Deliveries need referential integrity and filterable customer metadata.

#### Scenario: Register customer
- **Given** no row exists for `customer_id` `acme_corp`
- **When** client POSTs valid customer payload
- **Then** response is `201` with customer body including `row_version` 1

#### Scenario: Reject duplicate customer
- **Given** `acme_corp` already exists
- **When** client POSTs same `customer_id`
- **Then** response is `409`

### Requirement: Delivery customer guard
The system SHALL reject delivery commit when `customer_id` is not registered.

**Priority**: P0 (Critical)
**Rationale**: FK enforcement at API layer before DB error.

#### Scenario: Unknown customer on commit
- **Given** `customer_id` is not in `customers`
- **When** client POSTs delivery commit with valid `asset_ids` and idempotency key
- **Then** response is `422` with customer not found

### Requirement: Filter deliveries by customer
The system SHALL support listing deliveries filtered by `customer_id` query parameter.

**Priority**: P1 (High)

#### Scenario: List deliveries for one customer
- **Given** deliveries exist for `acme_corp`
- **When** client GETs `/deliveries?customer_id=acme_corp`
- **Then** all returned items have matching `customer_id`

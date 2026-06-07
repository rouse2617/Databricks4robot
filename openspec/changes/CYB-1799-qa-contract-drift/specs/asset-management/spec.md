## MODIFIED Requirements

### Requirement: API contracts match UI-visible asset operations
- **Before**: The system documented only part of the query and operation surface used by asset events, MCAP files, algo runs, and pipeline operations.
- **After**: The system SHALL keep OpenAPI, API guide, SDK helpers, and frontend clients aligned for existing UI-visible asset-management operations.
- **Reason**: Contract drift causes generated clients, SDK users, QA, and frontend code to exercise different behavior.

**Priority**: P1 (High)
**Rationale**: These APIs are already used by production-like UI flows, so incomplete contracts create regression risk without adding new product behavior.

#### Scenario: Asset event filters are discoverable
- **Given** the asset detail UI filters algorithm events by `event_type=algo_*`
- **When** an API consumer reads the asset events contract
- **Then** the contract documents event type filtering, algorithm key filtering, cursor pagination, and the next cursor response.

#### Scenario: SDK exposes UI-equivalent filters
- **Given** the UI can filter MCAP files and algo runs
- **When** an SDK user needs the same filtered result set
- **Then** the SDK exposes the same canonical query parameters documented in OpenAPI.

#### Scenario: Existing pipeline operations are documented
- **Given** the backend exposes pipeline active-version, promote, and batch template-run routes
- **When** an API consumer reads OpenAPI or uses the SDK
- **Then** those operations are available with request and response contracts.

### Requirement: List failures are visible to users
- **Before**: The algo runs list could catch an API failure and render an empty table with no visible error.
- **After**: The algo runs list SHALL surface a user-visible error state when loading fails.
- **Reason**: Users must be able to distinguish "no runs exist" from "the backend request failed".

**Priority**: P1 (High)
**Rationale**: Silent failure states make QA and operational troubleshooting unreliable.

#### Scenario: Algo run list request fails
- **Given** the algo runs list API returns an error or cannot be reached
- **When** the user opens or refreshes the algo runs page
- **Then** the page shows an actionable error message and does not imply that there are zero runs.

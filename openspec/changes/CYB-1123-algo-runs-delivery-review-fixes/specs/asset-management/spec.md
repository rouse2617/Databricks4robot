## MODIFIED Requirements

### Requirement: Algo run list pagination contract
- **Before**: The system SHALL expose algo-runs listing with implementation-defined pagination semantics, and clients may infer pagination state from the current page payload.
- **After**: The system SHALL expose a single documented pagination contract for `GET /api/v1/algo-runs`, including request parameters and response envelope, and SHALL return a `total` value whose meaning is consistent with frontend pagination and API documentation.
- **Reason**: Recent dev work introduced a backend/frontend contract mismatch that broke paging behavior and made the reported `total` unusable.

**Priority**: P0 (Critical)
**Rationale**: Broken pagination hides data, repeats the wrong rows across pages, and makes the algo-runs UI unreliable.

#### Scenario: Second page returns the correct next slice
- **Given** more algo runs exist than fit on the first page
- **When** the client requests the second page using the documented pagination parameters
- **Then** the API SHALL return the next slice of rows instead of repeating the first page

#### Scenario: Total count is stable across pages
- **Given** a filtered algo-runs query matches more rows than the current page size
- **When** the client requests page 1 and page 2 for the same filter
- **Then** the response SHALL report the same total result count semantics on both pages

### Requirement: Algo run creation conflict behavior
- **Before**: The system SHALL treat duplicate `run_id` creation behavior as implementation-defined.
- **After**: The system SHALL return the documented and tested conflict outcome when a client attempts to create an algo run with an already-registered `run_id`.
- **Reason**: Current behavior diverged from the published contract and hid duplicate registration mistakes.

**Priority**: P1 (High)
**Rationale**: Workers and manual operators need deterministic behavior for duplicate run registration.

#### Scenario: Duplicate run registration follows the documented contract
- **Given** an algo run with `run_id=R` already exists
- **When** a client submits `POST /api/v1/algo-runs` again with `run_id=R`
- **Then** the API SHALL return the documented duplicate outcome and error semantics
